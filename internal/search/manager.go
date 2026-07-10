package search

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/utils"
)

const defaultLimit = 10
const searchTimeout = 12 * time.Second

type Manager struct {
	searchers []Searcher
	debouncer *utils.Debouncer
	mu        sync.Mutex
	cancel    context.CancelFunc
	onResult  func(models.SearchResults)
}

func NewManager() *Manager {
	return &Manager{
		debouncer: utils.NewDebouncer(180 * time.Millisecond),
	}
}

func (m *Manager) AddSearcher(s Searcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchers = append(m.searchers, s)
}

func (m *Manager) OnResult(fn func(models.SearchResults)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onResult = fn
}

func (m *Manager) Search(query string) {
	query = trimSpace(query)
	if query == "" {
		m.Cancel()
		m.emit(models.SearchResults{})
		return
	}

	m.debouncer.Reset(func() {
		m.doSearch(query)
	})
}

func (m *Manager) doSearch(query string) {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	m.cancel = cancel
	searchers := append([]Searcher{}, m.searchers...)
	m.mu.Unlock()

	var mu sync.Mutex
	results := make(map[string][]models.Song)
	var wg sync.WaitGroup

	for _, s := range searchers {
		wg.Add(1)
		go func(searcher Searcher) {
			defer wg.Done()
			if ctx.Err() != nil {
				return
			}

			songs, err := searcher.Search(ctx, query, defaultLimit)
			if err != nil || len(songs) == 0 {
				return
			}

			for i := range songs {
				// Score by title + artist for better ranking.
				titleScore := FuzzyFind(query, songs[i].Title).Score
				artistScore := FuzzyFind(query, songs[i].Artist).Score
				if artistScore > titleScore {
					songs[i].Score = artistScore
				} else {
					songs[i].Score = titleScore
				}
				// Exact substring boost.
				if containsFold(songs[i].Title, query) {
					songs[i].Score += 0.5
				}
			}
			sort.Slice(songs, func(i, j int) bool {
				return songs[i].Score > songs[j].Score
			})

			mu.Lock()
			results[searcher.Name()] = songs
			mu.Unlock()
		}(s)
	}

	wg.Wait()

	// Drop results if a newer search superseded this one.
	if ctx.Err() == context.Canceled {
		return
	}

	groups := make([]models.SearchResultGroup, 0, len(results))
	total := 0
	sourceOrder := []string{"YouTube", "Radio", "Navidrome", "Library", "Favorites"}
	for _, name := range sourceOrder {
		songs, ok := results[name]
		if !ok {
			continue
		}
		var source models.SourceType
		switch name {
		case "YouTube":
			source = models.SourceYouTube
		case "Radio":
			source = models.SourceRadio
		case "Navidrome":
			source = models.SourceNavidrome
		case "Library":
			source = models.SourceLocal
		case "Favorites":
			// Keep original source on each song; group label is Favorites.
			if len(songs) > 0 {
				source = songs[0].Source
			}
		}
		groups = append(groups, models.SearchResultGroup{
			Source: source,
			Name:   name,
			Items:  songs,
			Index:  len(groups),
		})
		total += len(songs)
	}

	m.emit(models.SearchResults{
		Query:  query,
		Groups: groups,
		Total:  total,
	})
}

func (m *Manager) emit(results models.SearchResults) {
	m.mu.Lock()
	fn := m.onResult
	m.mu.Unlock()
	if fn != nil {
		fn(results)
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

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	return len(s) >= len(sub) && (FuzzyFind(sub, s).Score > 0 || indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	// Simple ASCII-ish fold contains for boost; fuzzy already covers unicode-ish.
	ls, lsub := make([]rune, 0, len(s)), make([]rune, 0, len(sub))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		ls = append(ls, r)
	}
	for _, r := range sub {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		lsub = append(lsub, r)
	}
	// naive search
	for i := 0; i+len(lsub) <= len(ls); i++ {
		match := true
		for j := 0; j < len(lsub); j++ {
			if ls[i+j] != lsub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
