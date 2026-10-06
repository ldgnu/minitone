package ui

import (
	"image"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/search"
	"github.com/ldgnu/minitone/internal/source/library"
	"github.com/ldgnu/minitone/internal/source/navidrome"
	"github.com/ldgnu/minitone/internal/source/youtube"
	"github.com/ldgnu/minitone/internal/store"
)

// Focus decides where a keystroke goes.
//
// The whole point of this split: letter shortcuts (f, a, d, j…) and free
// typing have to coexist. In FocusSearch every printable key is typed; in
// FocusList the single-key actions live and any other printable key falls
// back to typing.
type Focus int

const (
	FocusSearch Focus = iota
	FocusList
)

func (f Focus) String() string {
	if f == FocusList {
		return "list"
	}
	return "search"
}

// Panel is a full-height overlay. Panels never replace the main layout: the
// player bar stays visible underneath, so context is never lost.
type Panel int

const (
	PanelNone Panel = iota
	PanelQueue
	PanelFavorites
	PanelHistory
	PanelLibrary
	PanelPlaylists
	PanelDetails
	PanelHelp
)

func (p Panel) String() string {
	switch p {
	case PanelQueue:
		return "queue"
	case PanelFavorites:
		return "favorites"
	case PanelHistory:
		return "history"
	case PanelLibrary:
		return "library"
	case PanelPlaylists:
		return "playlists"
	case PanelDetails:
		return "details"
	case PanelHelp:
		return "help"
	default:
		return ""
	}
}

// ── search state ────────────────────────────────────────────────────────────

// searchState groups everything about "what the user is searching for".
type searchState struct {
	query  string
	term   string // query without the optional /source prefix
	source string // source selected by a /prefix, "" for all

	cursor int // row inside the active group
	group  int // active source group

	// bySource holds results per source name, merged as they stream in.
	bySource map[string][]models.Song
	// pending lists sources that have not answered yet.
	pending map[string]bool
	// failures maps a source name to its error.
	failures map[string]error

	running   bool
	startedAt time.Time

	lastQuery string // kept so a failed search can be retried
	restrict  string // source restriction from the selector
}

func newSearchState() searchState {
	return searchState{
		bySource: map[string][]models.Song{},
		pending:  map[string]bool{},
		failures: map[string]error{},
	}
}

// groups builds the ordered result groups from what has arrived so far.
func (s searchState) groups() []models.SearchResultGroup {
	return search.BuildGroups(s.bySource)
}

// total is the number of results received so far.
func (s searchState) total() int {
	n := 0
	for _, songs := range s.bySource {
		n += len(songs)
	}
	return n
}

// pendingSources returns the sources still searching, in display order.
func (s searchState) pendingSources() []string {
	var out []string
	for _, name := range search.SourceOrder() {
		if s.pending[name] {
			out = append(out, name)
		}
	}
	return out
}

// failuresList returns the per-source failures in display order.
func (s searchState) failuresList() []search.SourceError {
	var out []search.SourceError
	for _, name := range search.SourceOrder() {
		if err, ok := s.failures[name]; ok {
			out = append(out, search.SourceError{Source: name, Err: err})
		}
	}
	return out
}

// reset clears everything about the current search.
func (s *searchState) reset() {
	s.cursor = 0
	s.group = 0
	s.bySource = map[string][]models.Song{}
	s.pending = map[string]bool{}
	s.failures = map[string]error{}
	s.running = false
}

// ── messages ────────────────────────────────────────────────────────────────

type searchEventMsg struct{ ev search.Event }

type thumbMsg struct {
	url string
	img image.Image
	err error
}

type tickMsg time.Time

type songEndedMsg struct{}

type playerErrMsg struct{ err error }

type scanMsg struct{ status library.ScanStatus }

type sessionSaveMsg struct{}

// ── model ───────────────────────────────────────────────────────────────────

type Model struct {
	width  int
	height int

	focus  Focus
	search searchState
	queue  *queue.Queue
	lib    libraryState
	panel  Panel

	// panelCursor drives the overlay lists.
	panelCursor int
	// focusBeforePanel restores the focus when a panel closes: closing a panel
	// with esc must return you to the list you came from, not to the search box
	// (where n/p/space would turn into typed characters).
	focusBeforePanel Focus

	player  *player.Player
	sm      SearchRunner
	favs    *store.Favorites
	hist    *store.History
	playls  *store.Playlists
	session *store.SessionStore

	// plLevel / plIndex drive the two-level playlists browser:
	// level 0 = playlists, level 1 = tracks of the selected playlist.
	plLevel int
	plIndex int

	lastSong models.Song // last resolved/played song (for favorite toggling)
	details  models.Song // song shown by the details overlay

	// pendingPlay is the song we asked mpv to load; status may still be loading.
	pendingPlay models.Song

	// playToken identifies the current "play this" request; stale resolve
	// results are discarded when they come back.
	playToken playToken

	youtubeClient   *youtube.Client
	navidromeClient *navidrome.Client
	libraryScanner  *library.Scanner

	sources   []search.Source
	sourceSel int // index in sources (the selector row)

	themeIdx int
	styles   Styles

	notice       notice
	spinnerFrame int

	err        error
	statusText string

	// yt-dlp: YouTube needs it; when missing we say so once, up front.
	ytdlpMissing bool
	mpvMissing   bool

	videoMode bool

	// YouTube thumbnail preview (braille/ANSI art) for the selected result.
	thumbs   map[string]image.Image
	thumbURL string

	// restoreSessionOn controls whether the session is persisted/restored.
	restoreSessionOn bool
	lastSessionAt    time.Time

	// channels bridging the background workers into the Bubble Tea loop.
	searchCh chan search.Event
	endedCh  chan struct{}
	scanCh   chan library.ScanStatus

	keys KeyMap
}

// SearchRunner is the part of search.Manager the UI uses.
//
// Depending on this small interface (instead of the concrete manager) keeps
// the UI honest: it can only search, restrict, describe and cancel, and tests
// can substitute a fake to assert exactly what was asked for.
type SearchRunner interface {
	Search(query string)
	SearchNow(query string)
	RestrictTo(names []string)
	SetDebounce(ms int)
	Describe() []search.Source
	OnEvent(fn func(search.Event))
	Cancel()
}

// Deps are the collaborators the UI needs. Everything optional is tolerated:
// the UI must render (and explain itself) with no player, no search manager
// and no sources at all.
type Deps struct {
	Player   *player.Player
	Queue    *queue.Queue
	Search   SearchRunner
	YouTube  *youtube.Client
	Nav      *navidrome.Client
	Library  *library.Scanner
	Favs     *store.Favorites
	History  *store.History
	Playlists *store.Playlists
	Session  *store.SessionStore
	Theme    string
	Debounce int
	Restore  bool
	// Keys overrides the default bindings (from config keybindings).
	Keys *KeyMap
}

func New(d Deps) Model {
	searchEvents := make(chan search.Event, 64)
	ended := make(chan struct{}, 8)
	scanEvents := make(chan library.ScanStatus, 16)

	if d.Search != nil {
		d.Search.OnEvent(func(ev search.Event) {
			select {
			case searchEvents <- ev:
			default:
				// Drop the oldest event rather than blocking the searcher.
				select {
				case <-searchEvents:
				default:
				}
				select {
				case searchEvents <- ev:
				default:
				}
			}
		})
		d.Search.SetDebounce(d.Debounce)
	}

	if d.Player != nil {
		d.Player.OnEnded(func() {
			select {
			case ended <- struct{}{}:
			default:
			}
		})
		d.Player.OnError(func(err error) {
			if err == nil {
				return
			}
			// Player errors are surfaced by the notice bar; keep them short.
			_ = err
		})
	}

	if d.Favs == nil {
		d.Favs = store.NewFavorites("")
	}
	if d.History == nil {
		d.History = store.NewHistory("", store.DefaultHistoryMax)
	}
	if d.Playlists == nil {
		d.Playlists = store.NewPlaylists("")
	}
	if d.Queue == nil {
		d.Queue = queue.New()
	}

	m := Model{
		player:           d.Player,
		queue:            d.Queue,
		sm:               d.Search,
		favs:             d.Favs,
		hist:             d.History,
		playls:           d.Playlists,
		session:          d.Session,
		search:           newSearchState(),
		searchCh:         searchEvents,
		endedCh:          ended,
		scanCh:           scanEvents,
		youtubeClient:    d.YouTube,
		navidromeClient:  d.Nav,
		libraryScanner:   d.Library,
		themeIdx:         ThemeIndex(d.Theme),
		keys:             NewKeyMap(),
		restoreSessionOn: d.Restore,
		thumbs:           map[string]image.Image{},
		focus:            FocusSearch,
	}
	m.styles = NewStyles(themes[m.themeIdx])

	if d.Library != nil {
		m.lib.sections = d.Library.Groups()
		m.lib.status = "ctrl+l to browse"
	}
	if d.Search != nil {
		m.sources = d.Search.Describe()
	}
	m.ytdlpMissing = d.YouTube != nil && !d.YouTube.Available()
	m.mpvMissing = d.Player == nil || !player.Available()

	return m
}

func (m Model) Init() tea.Cmd {
	if m.mpvMissing {
		m.notice = errorNotice("mpv not installed", "install mpv to play audio", "[?] help")
	} else if m.ytdlpMissing {
		m.notice = warnNotice("yt-dlp not installed — YouTube is disabled",
			"radio, navidrome and the local library still work", "")
	}
	// Index the library in the background and apply a stored session.
	var cmd tea.Cmd
	if m.libraryScanner != nil && len(m.libraryScanner.Dirs()) > 0 {
		_ = m.startScan()
	}
	m.restoreSession()

	return tea.Batch(
		cmd,
		m.waitSearchEvents(),
		m.waitEnded(),
		m.waitScan(),
		tickCmd(),
	)
}

// waitSearchEvents blocks until a source reports results/errors.
func (m Model) waitSearchEvents() tea.Cmd {
	ch := m.searchCh
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return searchEventMsg{ev: ev}
	}
}

// waitEnded blocks until the current track finishes.
func (m Model) waitEnded() tea.Cmd {
	ch := m.endedCh
	return func() tea.Msg {
		<-ch
		return songEndedMsg{}
	}
}

// waitScan blocks until the library scan reports progress.
func (m Model) waitScan() tea.Cmd {
	ch := m.scanCh
	return func() tea.Msg {
		st, ok := <-ch
		if !ok {
			return nil
		}
		return scanMsg{status: st}
	}
}

// tick drives the spinner, the elapsed-time refresh and the periodic session
// save. 200ms keeps the progress bar smooth without much redraw cost.
func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
