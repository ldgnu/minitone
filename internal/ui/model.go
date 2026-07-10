package ui

import (
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

type Panel int

const (
	PanelNone Panel = iota
	PanelQueue
	PanelFavorites
	PanelHistory
	PanelHelp
)

type searchResultsMsg struct {
	results models.SearchResults
}

type tickMsg time.Time

type songEndedMsg struct{}

type playErrMsg struct{ err error }

type playOKMsg struct {
	song models.Song
}

type Model struct {
	width  int
	height int

	searchQuery  string
	searchCursor int
	searchGroup  int
	searchActive bool
	searching    bool

	searchResults   models.SearchResults
	searchResultsCh chan models.SearchResults
	endedCh         chan struct{}

	showPanel   Panel
	panelCursor int

	player *player.Player
	queue  *queue.Queue
	sm     *search.Manager
	favs   *store.Favorites
	hist   *store.History

	// last successfully resolved/played song (for favorite toggle)
	lastSong models.Song

	youtubeClient   *youtube.Client
	navidromeClient *navidrome.Client
	libraryScanner  *library.Scanner

	themeIdx int
	styles   Styles

	err        error
	statusText string

	keys KeyMap
}

type Deps struct {
	Player   *player.Player
	Queue    *queue.Queue
	Search   *search.Manager
	YouTube  *youtube.Client
	Nav      *navidrome.Client
	Library  *library.Scanner
	Favs     *store.Favorites
	History  *store.History
	Theme    string
}

func New(d Deps) Model {
	ch := make(chan models.SearchResults, 16)
	ended := make(chan struct{}, 4)

	if d.Search != nil {
		d.Search.OnResult(func(results models.SearchResults) {
			select {
			case ch <- results:
			default:
				select {
				case <-ch:
				default:
				}
				select {
				case ch <- results:
				default:
				}
			}
		})
	}

	if d.Player != nil {
		d.Player.OnEnded(func() {
			select {
			case ended <- struct{}{}:
			default:
			}
		})
	}

	if d.Favs == nil {
		d.Favs = store.NewFavorites("")
	}
	if d.History == nil {
		d.History = store.NewHistory("", store.DefaultHistoryMax)
	}

	themeIdx := ThemeIndex(d.Theme)
	m := Model{
		player:          d.Player,
		queue:           d.Queue,
		sm:              d.Search,
		favs:            d.Favs,
		hist:            d.History,
		youtubeClient:   d.YouTube,
		navidromeClient: d.Nav,
		libraryScanner:  d.Library,
		themeIdx:        themeIdx,
		keys:            DefaultKeyMap(),
		searchResultsCh: ch,
		endedCh:         ended,
		statusText:      "type to search · f fav · ctrl+f/h panels · ? help · q quit",
	}
	m.styles = NewStyles(themes[m.themeIdx])
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.waitResults(), m.waitEnded(), tickCmd())
}

func (m Model) waitResults() tea.Cmd {
	return func() tea.Msg {
		r, ok := <-m.searchResultsCh
		if !ok {
			return nil
		}
		return searchResultsMsg{results: r}
	}
}

func (m Model) waitEnded() tea.Cmd {
	return func() tea.Msg {
		_, ok := <-m.endedCh
		if !ok {
			return nil
		}
		return songEndedMsg{}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
