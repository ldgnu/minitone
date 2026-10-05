package search

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/utils"
)

const defaultLimit = 10
const searchTimeout = 12 * time.Second

// Debounce bounds (milliseconds).
const (
	defaultDebounceMs = 300
	minDebounceMs     = 100
	maxDebounceMs     = 800
)

// Source names in display order.
const (
	SourceYouTube   = "YouTube"
	SourceRadio     = "Radio"
	SourceNavidrome = "Navidrome"
	SourceLibrary   = "Library"
	SourceFavorites = "Favorites"
)

var sourceOrder = []string{
	SourceYouTube,
	SourceRadio,
	SourceNavidrome,
	SourceLibrary,
	SourceFavorites,
}

// Event is a single incremental search update.
//
// Results are emitted per source as they arrive instead of waiting for every
// source to finish, so a slow (or dead) source never blocks the others.
type Event struct {
	// Query the event belongs to.
	Query string
	// Source that produced the event ("" for a "started" event).
	Source string
	// Songs for this source, ranked. Nil on error/start/finish.
	Songs []models.Song
	// Err is set when this source failed.
	Err error
	// Started is true for the very first event of a query.
	Started bool
	// Done is true once this source finished (with or without results).
	Done bool
}

// SourceError is a per-source failure the UI can offer to retry.
type SourceError struct {
	Source string
	Err    error
}

func (e SourceError) Error() string {
	if e.Err == nil {
		return e.Source
	}
	return e.Source + ": " + e.Err.Error()
}

// Manager fans a query out to every registered searcher and streams results
// back through OnEvent.
type Manager struct {
	searchers []Searcher
	debouncer *utils.Debouncer
	mu        sync.Mutex
	cancel    context.CancelFunc
	onEvent   func(Event)

	// only, when non-empty, restricts the query to these source names.
	only map[string]bool
	// limit is the per-source result cap.
	limit int
}

func NewManager() *Manager {
	return &Manager{
		debouncer: utils.NewDebouncer(300 * time.Millisecond),
		limit:     defaultLimit,
		only:      nil,
	}
}

// SetDebounce configures the keystroke debounce (clamped to 100..800ms).
func (m *Manager) SetDebounce(ms int) {
	switch {
	case ms <= 0:
		ms = defaultDebounceMs
	case ms < minDebounceMs:
		ms = minDebounceMs
	case ms > maxDebounceMs:
		ms = maxDebounceMs
	}
	m.debouncer.SetDelay(time.Duration(ms) * time.Millisecond)
}

// SetLimit changes how many results each source returns (0 resets to default).
func (m *Manager) SetLimit(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n <= 0 {
		n = defaultLimit
	}
	m.limit = n
}

// sourceAliases maps the friendly names users type or configure to the
// canonical source name. Without this, "local" (the prefix, and the obvious
// config value) would silently match nothing.
var sourceAliases = map[string]string{
	"all": "", "any": "", "search": "",
	"youtube": SourceYouTube, "yt": SourceYouTube,
	"radio": SourceRadio, "radiobrowser": SourceRadio, "browser": SourceRadio,
	"navidrome": SourceNavidrome, "nav": SourceNavidrome, "subsonic": SourceNavidrome,
	"library": SourceLibrary, "local": SourceLibrary, "lib": SourceLibrary,
	"favorites": SourceFavorites, "favourites": SourceFavorites, "fav": SourceFavorites,
}

// CanonicalSource resolves a user-facing name (case-insensitive) to a source
// name, and reports whether it matched. "all"/"" mean "no restriction".
func CanonicalSource(name string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return "", true // an empty name means "no restriction"
	}
	if canonical, ok := sourceAliases[key]; ok {
		return canonical, true
	}
	for _, s := range sourceOrder {
		if strings.ToLower(s) == key {
			return s, true
		}
	}
	return "", false
}

// RestrictTo limits searches to the given source names (empty, "all" or an
// unknown name means every source).
func (m *Manager) RestrictTo(names []string) {
	if len(names) == 0 {
		m.mu.Lock()
		m.only = nil
		m.mu.Unlock()
		return
	}
	set := make(map[string]bool, len(names))
	unknown := false
	for _, n := range names {
		canonical, ok := CanonicalSource(n)
		switch {
		case !ok:
			unknown = true // do not silently search nothing
		case canonical == "":
			// "all" / "" → no restriction at all.
			set = nil
			m.mu.Lock()
			m.only = nil
			m.mu.Unlock()
			return
		default:
			set[strings.ToLower(canonical)] = true
		}
	}
	if unknown && len(set) == 0 {
		set = nil
	}
	m.mu.Lock()
	m.only = set
	m.mu.Unlock()
}

func (m *Manager) AddSearcher(s Searcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchers = append(m.searchers, s)
}

// OnEvent registers the streaming update callback (replaces the old
// single-shot OnResult).
func (m *Manager) OnEvent(fn func(Event)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onEvent = fn
}

// Search schedules a debounced search for query.
func (m *Manager) Search(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		m.Cancel()
		m.emit(Event{Done: true})
		return
	}

	m.debouncer.Reset(func() {
		m.doSearch(query)
	})
}

// SearchNow runs a search immediately, skipping the debounce.
func (m *Manager) SearchNow(query string) {
	m.debouncer.Cancel()
	query = strings.TrimSpace(query)
	if query == "" {
		m.Cancel()
		m.emit(Event{Done: true})
		return
	}
	m.doSearch(query)
}

// doSearch fans out to every target source. It never blocks the caller: the
// fan-out runs in its own goroutine and reports through emit as sources finish.
func (m *Manager) doSearch(query string) {
	m.mu.Lock()
	if m.cancel != nil {
		// A newer query supersedes this one: the old context is cancelled so
		// its results are dropped by the generation check below.
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	m.cancel = cancel

	type target struct {
		name string
		fn   Searcher
	}
	var targets []target
	for _, s := range m.searchers {
		if m.only != nil && !m.only[strings.ToLower(s.Name())] {
			continue
		}
		targets = append(targets, target{name: s.Name(), fn: s})
	}
	limit := m.limit
	if limit <= 0 {
		limit = defaultLimit
	}
	onEvent := m.onEvent
	m.mu.Unlock()

	emit := func(ev Event) {
		if onEvent != nil {
			onEvent(ev)
		}
	}

	if len(targets) == 0 {
		emit(Event{Query: query, Started: true, Done: true})
		return
	}

	emit(Event{Query: query, Started: true})

	go func() {
		var wg sync.WaitGroup
		for _, t := range targets {
			wg.Add(1)
			go func(t target) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}

				songs, err := t.fn.Search(ctx, query, limit)

				// A cancelled/expired context means this result belongs to an
				// outdated query (or one already past its deadline) and must
				// not overwrite fresher results.
				if ctx.Err() != nil {
					return
				}

				switch {
				case err != nil:
					emit(Event{Query: query, Source: t.name, Err: err, Done: true})
				case len(songs) == 0:
					emit(Event{Query: query, Source: t.name, Done: true})
				default:
					emit(Event{Query: query, Source: t.name, Songs: rank(query, songs), Done: true})
				}
			}(t)
		}
		wg.Wait()
		cancel()
	}()
}

// rank sorts songs by fuzzy score (title vs artist, with a substring boost).
func rank(query string, songs []models.Song) []models.Song {
	for i := range songs {
		titleScore := FuzzyFind(query, songs[i].Title).Score
		artistScore := FuzzyFind(query, songs[i].Artist).Score
		if artistScore > titleScore {
			songs[i].Score = artistScore
		} else {
			songs[i].Score = titleScore
		}
		if strings.Contains(strings.ToLower(songs[i].Title), strings.ToLower(query)) {
			songs[i].Score += 0.5
		}
	}
	sort.SliceStable(songs, func(i, j int) bool {
		return songs[i].Score > songs[j].Score
	})
	return songs
}

func (m *Manager) emit(ev Event) {
	m.mu.Lock()
	fn := m.onEvent
	m.mu.Unlock()
	if fn != nil {
		fn(ev)
	}
}

func (m *Manager) Cancel() {
	m.debouncer.Cancel()
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.mu.Unlock()
}

// SourceOrder returns the canonical display order of source names.
func SourceOrder() []string {
	return append([]string{}, sourceOrder...)
}

// BuildGroups assembles per-source songs into ordered result groups,
// preserving the canonical source order and skipping empty sources.
func BuildGroups(bySource map[string][]models.Song) []models.SearchResultGroup {
	groups := make([]models.SearchResultGroup, 0, len(bySource))
	total := 0
	for _, name := range sourceOrder {
		songs, ok := bySource[name]
		if !ok || len(songs) == 0 {
			continue
		}
		groups = append(groups, models.SearchResultGroup{
			Source: SourceTypeFor(name),
			Name:   name,
			Items:  songs,
			Index:  len(groups),
		})
		total += len(songs)
	}
	return groups
}

// CountTotal returns the number of songs across groups.
func CountTotal(groups []models.SearchResultGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Items)
	}
	return n
}

// SourceTypeFor maps a source display name to the model type.
func SourceTypeFor(name string) models.SourceType {
	switch name {
	case SourceYouTube:
		return models.SourceYouTube
	case SourceRadio:
		return models.SourceRadio
	case SourceNavidrome:
		return models.SourceNavidrome
	case SourceLibrary:
		return models.SourceLocal
	case SourceFavorites:
		// Favorites keep the original source on each song; the group is just
		// a view over them.
		return models.SourceLocal
	default:
		return models.SourceType(name)
	}
}
