package ui

import (
	"context"
	"fmt"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/queue"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case searchResultsMsg:
		m.searchResults = msg.results
		m.searching = false
		if m.searchGroup >= len(m.searchResults.Groups) {
			m.searchGroup = 0
		}
		if m.searchCursor >= m.groupLen() {
			m.searchCursor = 0
		}
		if len(m.searchResults.Groups) > 0 {
			m.searchActive = true
			m.statusText = fmt.Sprintf("%d results", m.searchResults.Total)
		} else if m.searchQuery != "" {
			m.statusText = "no results"
		}
		return m, m.waitResults()

	case songEndedMsg:
		return m, tea.Batch(m.autoAdvance(), m.waitEnded())

	case tickMsg:
		return m, tickCmd()

	case playErrMsg:
		m.err = msg.err
		m.statusText = msg.err.Error()
		return m, nil

	case playOKMsg:
		m.err = nil
		if msg.song.Title == "" && msg.song.ID == "" {
			m.statusText = "end of queue"
			return m, nil
		}
		m.lastSong = msg.song
		if m.hist != nil {
			m.hist.Push(msg.song)
		}
		star := ""
		if m.favs != nil && m.favs.Contains(msg.song) {
			star = " ★"
		}
		m.statusText = fmt.Sprintf("playing · %s%s", msg.song.Title, star)
		m.searchQuery = ""
		m.searchResults = models.SearchResults{}
		m.searchActive = false
		m.searching = false
		return m, nil

	case error:
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showPanel != PanelNone {
		return m.handlePanelKeys(msg)
	}

	key := msg.String()

	// Always-on shortcuts (work while typing and while browsing).
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// Clear search.
		m.searchQuery = ""
		m.searchResults = models.SearchResults{}
		m.searchCursor = 0
		m.searchGroup = 0
		m.searchActive = false
		m.searching = false
		m.sm.Cancel()
		m.err = nil
		m.statusText = "type to search · t theme · f fav · h history · ? help · q quit"
		return m, nil
	case "enter":
		return m.handleControlKey("enter")
	case "tab", "shift+tab":
		return m.handleControlKey(key)
	case "up", "down", "left", "right", "+", "=", "-":
		return m.handleControlKey(key)
	case "ctrl+j":
		return m.handleControlKey("ctrl+j")
	case "ctrl+f":
		return m.openFavoritesPanel()
	case "ctrl+a":
		return m.favoriteSelectedOrCurrent()
	case "ctrl+h":
		if m.searchQuery == "" {
			return m.openHistoryPanel()
		}
		return m.handleControlKey("backspace")
	case "backspace":
		if len(m.searchQuery) > 0 {
			r := []rune(m.searchQuery)
			m.searchQuery = string(r[:len(r)-1])
			m.triggerSearch()
		}
		return m, nil
	case " ":
		if m.searchQuery == "" {
			_ = m.player.TogglePause()
			return m, nil
		}
		m.searchQuery += " "
		m.triggerSearch()
		return m, nil
	}

	// Single-key shortcuts — only when the search box is empty.
	// j/k are intentionally NOT here so they can be typed to start a search.
	if m.searchQuery == "" {
		switch key {
		case "q":
			return m, tea.Quit
		case "s":
			_ = m.player.Stop()
			m.statusText = "stopped"
			return m, nil
		case "n":
			return m, m.playNext()
		case "p":
			return m, m.playPrev()
		case "m":
			m.player.ToggleMute()
			m.statusText = fmt.Sprintf("vol %d%%", m.player.Volume())
			return m, nil
		case "t":
			m.themeIdx = (m.themeIdx + 1) % len(themes)
			m.styles = NewStyles(themes[m.themeIdx])
			m.statusText = fmt.Sprintf("theme: %s", themes[m.themeIdx].Name)
			return m, nil
		case "S":
			m.queue.SetShuffle(!m.queue.Shuffle())
			if m.queue.Shuffle() {
				m.statusText = "shuffle on"
			} else {
				m.statusText = "shuffle off"
			}
			return m, nil
		case "R":
			return m.cycleRepeat()
		case "f":
			return m.toggleFavoriteCurrent()
		case "h":
			return m.openHistoryPanel()
		case "?":
			m.showPanel = PanelHelp
			return m, nil
		}
	}

	// Anything printable is typed into the search (j/k included).
	if isPrintable(key) {
		m.searchQuery += key
		m.triggerSearch()
		return m, nil
	}

	return m.handleControlKey(key)
}

func (m Model) handleControlKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		if m.searchQuery != "" || m.searchActive {
			m.searchQuery = ""
			m.searchResults = models.SearchResults{}
			m.searchCursor = 0
			m.searchGroup = 0
			m.searchActive = false
			m.searching = false
			m.sm.Cancel()
			m.err = nil
			m.statusText = "type to search · f fav · ctrl+f/h panels · ? help · q quit"
		}
		return m, nil

	case "enter":
		if m.groupLen() > 0 && m.searchCursor < m.groupLen() {
			group := m.searchResults.Groups[m.searchGroup]
			if m.searchCursor < len(group.Items) {
				song := group.Items[m.searchCursor]
				m.queue.Add(song)
				m.queue.SetCursor(m.queue.Len() - 1)
				return m, m.resolveAndPlay(song)
			}
		}
		return m, nil

	case "up", "k":
		if m.searchCursor > 0 {
			m.searchCursor--
		}

	case "down", "j":
		if m.searchCursor < m.groupLen()-1 {
			m.searchCursor++
		}

	case "tab":
		if m.searchGroup < len(m.searchResults.Groups)-1 {
			m.searchGroup++
			m.searchCursor = 0
		} else if len(m.searchResults.Groups) > 0 {
			m.searchGroup = 0
			m.searchCursor = 0
		}

	case "shift+tab":
		if m.searchGroup > 0 {
			m.searchGroup--
			m.searchCursor = 0
		} else if len(m.searchResults.Groups) > 0 {
			m.searchGroup = len(m.searchResults.Groups) - 1
			m.searchCursor = 0
		}

	case "backspace":
		if len(m.searchQuery) > 0 {
			r := []rune(m.searchQuery)
			m.searchQuery = string(r[:len(r)-1])
			m.triggerSearch()
		}

	case "ctrl+j":
		if m.queue.Len() > 0 {
			m.showPanel = PanelQueue
			cur := m.queue.Cursor()
			if cur < 0 {
				cur = 0
			}
			m.panelCursor = cur
		} else {
			m.statusText = "queue is empty"
		}

	case "?":
		m.showPanel = PanelHelp

	case "q", "ctrl+c":
		return m, tea.Quit

	case "s":
		_ = m.player.Stop()
		m.statusText = "stopped"
	case "+", "=":
		m.player.SetVolume(m.player.Volume() + 5)
		m.statusText = fmt.Sprintf("vol %d%%", m.player.Volume())
	case "-":
		m.player.SetVolume(m.player.Volume() - 5)
		m.statusText = fmt.Sprintf("vol %d%%", m.player.Volume())
	case "right":
		m.player.Seek(5)
	case "left":
		m.player.Seek(-5)
	case "m":
		m.player.ToggleMute()
		m.statusText = fmt.Sprintf("vol %d%%", m.player.Volume())

	case "t":
		m.themeIdx = (m.themeIdx + 1) % len(themes)
		m.styles = NewStyles(themes[m.themeIdx])
		m.statusText = fmt.Sprintf("theme: %s", themes[m.themeIdx].Name)

	case "S":
		m.queue.SetShuffle(!m.queue.Shuffle())
		if m.queue.Shuffle() {
			m.statusText = "shuffle on"
		} else {
			m.statusText = "shuffle off"
		}

	case "R":
		return m.cycleRepeat()

	case "n":
		return m, m.playNext()
	case "p":
		return m, m.playPrev()
	}
	return m, nil
}

func (m Model) cycleRepeat() (tea.Model, tea.Cmd) {
	switch m.queue.Repeat() {
	case queue.RepeatOff:
		m.queue.SetRepeat(queue.RepeatAll)
		m.statusText = "repeat all"
	case queue.RepeatAll:
		m.queue.SetRepeat(queue.RepeatOne)
		m.statusText = "repeat one"
	case queue.RepeatOne:
		m.queue.SetRepeat(queue.RepeatOff)
		m.statusText = "repeat off"
	}
	return m, nil
}

func (m Model) openFavoritesPanel() (tea.Model, tea.Cmd) {
	if m.favs == nil || m.favs.Len() == 0 {
		m.statusText = "no favorites yet — press f on a playing track"
		return m, nil
	}
	m.showPanel = PanelFavorites
	m.panelCursor = 0
	m.statusText = fmt.Sprintf("favorites · %d", m.favs.Len())
	return m, nil
}

func (m Model) openHistoryPanel() (tea.Model, tea.Cmd) {
	if m.hist == nil || m.hist.Len() == 0 {
		m.statusText = "history empty — play something first"
		return m, nil
	}
	m.showPanel = PanelHistory
	m.panelCursor = 0
	m.statusText = fmt.Sprintf("history · %d", m.hist.Len())
	return m, nil
}

func (m Model) toggleFavoriteCurrent() (tea.Model, tea.Cmd) {
	song := m.lastSong
	if song.Title == "" {
		// Fall back to player metadata (weaker — may lack source ids).
		st := m.player.Status()
		if st.Song.Title == "" {
			m.statusText = "nothing to favorite"
			return m, nil
		}
		song = models.Song{
			Title:  st.Song.Title,
			Artist: st.Song.Artist,
			Album:  st.Song.Album,
			URL:    st.Song.URL,
			Source: models.SourceType(st.Song.Source),
			ID:     st.Song.Source + ":" + st.Song.URL,
		}
	}
	if m.favs.Toggle(song) {
		m.statusText = "★ added to favorites"
	} else {
		m.statusText = "☆ removed from favorites"
	}
	return m, nil
}

func (m Model) favoriteSelectedOrCurrent() (tea.Model, tea.Cmd) {
	// Prefer selected search result.
	if m.groupLen() > 0 && m.searchCursor < m.groupLen() {
		song := m.searchResults.Groups[m.searchGroup].Items[m.searchCursor]
		if m.favs.Toggle(song) {
			m.statusText = "★ favorited: " + song.DisplayTitle()
		} else {
			m.statusText = "☆ unfavorited: " + song.DisplayTitle()
		}
		return m, nil
	}
	return m.toggleFavoriteCurrent()
}

func (m Model) handlePanelKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch m.showPanel {
	case PanelQueue:
		switch key {
		case "esc", "ctrl+j", "q":
			m.showPanel = PanelNone
		case "up", "k":
			if m.panelCursor > 0 {
				m.panelCursor--
			}
		case "down", "j":
			if m.panelCursor < m.queue.Len()-1 {
				m.panelCursor++
			}
		case "enter":
			m.queue.SetCursor(m.panelCursor)
			item := m.queue.Current()
			if item != nil {
				m.showPanel = PanelNone
				return m, m.resolveAndPlay(item.Song)
			}
		case "d", "x", "delete":
			if m.queue.Remove(m.panelCursor) {
				if m.panelCursor >= m.queue.Len() && m.panelCursor > 0 {
					m.panelCursor--
				}
				if m.queue.Len() == 0 {
					m.showPanel = PanelNone
					m.statusText = "queue cleared"
				}
			}
		case "f":
			items := m.queue.Items()
			if m.panelCursor >= 0 && m.panelCursor < len(items) {
				song := items[m.panelCursor].Song
				if m.favs.Toggle(song) {
					m.statusText = "★ favorited"
				} else {
					m.statusText = "☆ unfavorited"
				}
			}
		case "ctrl+c":
			return m, tea.Quit
		}

	case PanelFavorites:
		switch key {
		case "esc", "ctrl+f", "q":
			m.showPanel = PanelNone
		case "up", "k":
			if m.panelCursor > 0 {
				m.panelCursor--
			}
		case "down", "j":
			if m.panelCursor < m.favs.Len()-1 {
				m.panelCursor++
			}
		case "enter":
			song := m.favs.Get(m.panelCursor)
			if song != nil {
				m.queue.Add(*song)
				m.queue.SetCursor(m.queue.Len() - 1)
				m.showPanel = PanelNone
				return m, m.resolveAndPlay(*song)
			}
		case "d", "x", "delete", "f":
			if m.favs.RemoveAt(m.panelCursor) {
				if m.panelCursor >= m.favs.Len() && m.panelCursor > 0 {
					m.panelCursor--
				}
				m.statusText = "removed from favorites"
				if m.favs.Len() == 0 {
					m.showPanel = PanelNone
				}
			}
		case "a":
			// enqueue all favorites
			for _, s := range m.favs.Songs() {
				m.queue.Add(s)
			}
			m.statusText = fmt.Sprintf("queued %d favorites", m.favs.Len())
		case "ctrl+c":
			return m, tea.Quit
		}

	case PanelHistory:
		switch key {
		case "esc", "h", "q":
			m.showPanel = PanelNone
		case "ctrl+h":
			m.showPanel = PanelNone
		case "up", "k":
			if m.panelCursor > 0 {
				m.panelCursor--
			}
		case "down", "j":
			if m.panelCursor < m.hist.Len()-1 {
				m.panelCursor++
			}
		case "enter":
			song := m.hist.Get(m.panelCursor)
			if song != nil {
				m.queue.Add(*song)
				m.queue.SetCursor(m.queue.Len() - 1)
				m.showPanel = PanelNone
				return m, m.resolveAndPlay(*song)
			}
		case "d", "x", "delete":
			if m.hist.RemoveAt(m.panelCursor) {
				if m.panelCursor >= m.hist.Len() && m.panelCursor > 0 {
					m.panelCursor--
				}
				if m.hist.Len() == 0 {
					m.showPanel = PanelNone
					m.statusText = "history empty"
				}
			}
		case "c":
			m.hist.Clear()
			m.showPanel = PanelNone
			m.statusText = "history cleared"
		case "f":
			song := m.hist.Get(m.panelCursor)
			if song != nil {
				if m.favs.Toggle(*song) {
					m.statusText = "★ favorited"
				} else {
					m.statusText = "☆ unfavorited"
				}
			}
		case "ctrl+c":
			return m, tea.Quit
		}

	case PanelHelp:
		if key == "esc" || key == "?" || key == "q" {
			m.showPanel = PanelNone
		}
		if key == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) triggerSearch() {
	m.searchCursor = 0
	m.searchGroup = 0
	m.searchActive = true
	m.err = nil
	if m.searchQuery == "" {
		m.searchResults = models.SearchResults{}
		m.searching = false
		m.sm.Cancel()
		return
	}
	m.searching = true
	m.statusText = "searching..."
	m.sm.Search(m.searchQuery)
}

func (m Model) resolveAndPlay(song models.Song) tea.Cmd {
	yt := m.youtubeClient
	nd := m.navidromeClient
	p := m.player

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var streamURL string
		var err error

		switch song.Source {
		case models.SourceYouTube:
			if yt == nil {
				return playErrMsg{err: fmt.Errorf("youtube client unavailable")}
			}
			streamURL, err = yt.ResolveSongContext(ctx, song)
			if err != nil {
				return playErrMsg{err: fmt.Errorf("youtube: %w", err)}
			}
		case models.SourceRadio:
			streamURL = song.URL
		case models.SourceNavidrome:
			if nd == nil {
				return playErrMsg{err: fmt.Errorf("navidrome not configured")}
			}
			streamURL = nd.StreamURL(song.SourceID)
		case models.SourceLocal:
			streamURL = song.FilePath
		default:
			streamURL = song.URL
			if streamURL == "" {
				streamURL = song.FilePath
			}
		}

		if streamURL == "" {
			return playErrMsg{err: fmt.Errorf("no stream URL for %s", song.Title)}
		}

		if err := p.Play(streamURL, song.Title, song.Artist, song.Album, string(song.Source)); err != nil {
			return playErrMsg{err: err}
		}
		return playOKMsg{song: song}
	}
}

func (m Model) autoAdvance() tea.Cmd {
	return m.playNext()
}

func (m Model) playNext() tea.Cmd {
	item := m.queue.Next()
	if item == nil {
		return func() tea.Msg {
			return playOKMsg{song: models.Song{}}
		}
	}
	return m.resolveAndPlay(item.Song)
}

func (m Model) playPrev() tea.Cmd {
	item := m.queue.Prev()
	if item == nil {
		return nil
	}
	return m.resolveAndPlay(item.Song)
}

func (m Model) groupLen() int {
	if m.searchGroup < 0 || m.searchGroup >= len(m.searchResults.Groups) {
		return 0
	}
	return len(m.searchResults.Groups[m.searchGroup].Items)
}

func isPrintable(key string) bool {
	if len(key) != 1 {
		r := []rune(key)
		if len(r) != 1 {
			return false
		}
		return unicode.IsPrint(r[0]) && !unicode.IsControl(r[0])
	}
	r := rune(key[0])
	return unicode.IsPrint(r) && !unicode.IsControl(r)
}
