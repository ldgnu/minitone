package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(checkConnection(m.client), tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case connectionOKMsg:
		m.connected = true
		m.state = viewNowPlaying
		m.loading = false
		return m, tea.Batch(fetchNowPlaying(m.client), fetchIsPlaying(m.client))

	case errMsg:
		m.err = msg.err
		m.loading = false
		if !m.connected {
			m.state = viewConnecting
		}
		return m, nil

	case nowPlayingMsg:
		if msg.err == nil {
			m.nowPlaying = msg.np
		}
		m.loading = false
		return m, nil

	case isPlayingMsg:
		m.isPlaying = msg.playing
		return m, nil

	case searchResultsMsg:
		if msg.err == nil {
			m.searchResults = msg.results
			m.searchCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case searchDetailMsg:
		if msg.err == nil {
			m.detail = msg.detail
			m.state = viewSearchDetail
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case playlistsMsg:
		if msg.err == nil {
			m.playlists = msg.playlists
			m.playlistCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case playlistTracksMsg:
		if msg.err == nil {
			m.playlistTracks = msg.tracks
			m.ptCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case queueMsg:
		if msg.err == nil {
			m.queue = msg.tracks
			m.queueCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case lyricsMsg:
		if msg.err == nil {
			m.lyrics = msg.text
		} else {
			m.lyrics = ""
		}
		m.loading = false
		return m, nil

	case tickMsg:
		if m.connected && m.state == viewNowPlaying {
			return m, tea.Batch(
				fetchNowPlaying(m.client),
				fetchIsPlaying(m.client),
				tick(),
			)
		}
		return m, tick()
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case viewConnecting:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "r" {
			m.err = nil
			return m, checkConnection(m.client)
		}
		return m, nil

	case viewNowPlaying:
		return m.handleNowPlayingKeys(msg)

	case viewSearch:
		return m.handleSearchKeys(msg)

	case viewSearchDetail:
		return m.handleSearchDetailKeys(msg)

	case viewQueue:
		return m.handleQueueKeys(msg)

	case viewLibrary:
		return m.handleLibraryKeys(msg)

	case viewPlaylistTracks:
		return m.handlePlaylistTracksKeys(msg)

	case viewLyrics:
		if msg.String() == "q" || msg.String() == "esc" {
			m.state = viewNowPlaying
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleNowPlayingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case " ":
		m.loading = true
		return m, func() tea.Msg {
			err := m.client.TogglePlayPause()
			if err != nil {
				return errMsg{err}
			}
			return fetchIsPlaying(m.client)()
		}
	case "n":
		m.loading = true
		return m, func() tea.Msg {
			if err := m.client.Next(); err != nil {
				return errMsg{err}
			}
			return nil
		}
	case "p":
		m.loading = true
		return m, func() tea.Msg {
			if err := m.client.Previous(); err != nil {
				return errMsg{err}
			}
			return nil
		}
	case "s":
		m.state = viewSearch
		m.searchQuery = ""
		m.searchResults = nil
		return m, nil
	case "l":
		m.state = viewLibrary
		m.loading = true
		return m, fetchPlaylists(m.client)
	case "w":
		m.state = viewQueue
		m.loading = true
		return m, fetchQueue(m.client)
	case "+", "=":
		if m.volume < 100 {
			m.volume += 5
			return m, func() tea.Msg {
				return m.client.SetVolume(m.volume)
			}
		}
	case "-":
		if m.volume > 0 {
			m.volume -= 5
			return m, func() tea.Msg {
				return m.client.SetVolume(m.volume)
			}
		}
	case "z":
		return m, func() tea.Msg {
			return m.client.ToggleShuffle()
		}
	case "x":
		return m, func() tea.Msg {
			return m.client.ToggleRepeat()
		}
	case "y":
		if m.nowPlaying.TrackID != "" {
			m.state = viewLyrics
			m.loading = true
			return m, fetchLyrics(m.client, m.nowPlaying.TrackID)
		}
	case "t":
		m.themeIdx = (m.themeIdx + 1) % len(themes)
		m.styles = NewStyles(themes[m.themeIdx])
		return m, nil
	}
	return m, nil
}

func (m *Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewNowPlaying
		return m, nil
	case "enter":
		q := strings.TrimSpace(m.searchQuery)
		if q == "" {
			return m, nil
		}
		m.loading = true
		return m, fetchSearch(m.client, q)
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	case "tab":
		categories := []string{"songs", "albums", "artists", "playlists"}
		for i, c := range categories {
			if c == m.searchCategory && i+1 < len(categories) {
				m.searchCategory = categories[i+1]
				m.searchCursor = 0
				break
			}
		}
	case "shift+tab":
		categories := []string{"songs", "albums", "artists", "playlists"}
		for i, c := range categories {
			if c == m.searchCategory && i-1 >= 0 {
				m.searchCategory = categories[i-1]
				m.searchCursor = 0
				break
			}
		}
	case "up", "k":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
	case "down", "j":
		results := m.searchResults[m.searchCategory]
		if m.searchCursor < len(results)-1 {
			m.searchCursor++
		}
	case "right", "l":
		if results, ok := m.searchResults[m.searchCategory]; ok && len(results) > 0 {
			idx := m.searchCursor
			if idx >= 0 && idx < len(results) {
				r := results[idx]
				if r.Type != "songs" {
					m.loading = true
					return m, fetchDetail(m.client, r.Type, r.ID)
				}
			}
		}
	case " ":
		if results, ok := m.searchResults[m.searchCategory]; ok && len(results) > 0 {
			idx := m.searchCursor
			if idx >= 0 && idx < len(results) {
				r := results[idx]
				return m, func() tea.Msg {
					err := m.client.PlayItem(r.Type, r.ID)
					if err != nil {
						return errMsg{err}
					}
					return nil
				}
			}
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}
	return m, nil
}

func (m *Model) handleSearchDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewSearch
		return m, nil
	case "up", "k":
		if m.detailCursor() > 0 {
			m.searchCursor--
		}
	case "down", "j":
		if m.detailCursor() < len(m.detail.Tracks)-1 {
			m.searchCursor++
		}
	case " ":
		tracks := m.detail.Tracks
		if len(tracks) > 0 {
			idx := m.detailCursor()
			if idx >= 0 && idx < len(tracks) {
				t := tracks[idx]
				return m, func() tea.Msg {
					err := m.client.PlayItem("songs", t.ID)
					if err != nil {
						return errMsg{err}
					}
					return nil
				}
			}
		}
	case "a":
		tracks := m.detail.Tracks
		if len(tracks) > 0 {
			idx := m.detailCursor()
			if idx >= 0 && idx < len(tracks) {
				t := tracks[idx]
				return m, func() tea.Msg {
					err := m.client.PlayLater("songs", t.ID)
					if err != nil {
						return errMsg{err}
					}
					return nil
				}
			}
		}
	}
	return m, nil
}

func (m *Model) handleQueueKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewNowPlaying
		return m, nil
	case "up", "k":
		if m.queueCursor > 0 {
			m.queueCursor--
		}
	case "down", "j":
		if m.queueCursor < len(m.queue)-1 {
			m.queueCursor++
		}
	case " ":
		if len(m.queue) > 0 && m.queueCursor >= 0 {
			t := m.queue[m.queueCursor]
			return m, func() tea.Msg {
				err := m.client.PlayItem("songs", t.ID)
				if err != nil {
					return errMsg{err}
				}
				return nil
			}
		}
	}
	return m, nil
}

func (m *Model) handleLibraryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewNowPlaying
		return m, nil
	case "up", "k":
		if m.playlistCursor > 0 {
			m.playlistCursor--
		}
	case "down", "j":
		if m.playlistCursor < len(m.playlists)-1 {
			m.playlistCursor++
		}
	case "right", "l":
		if len(m.playlists) > 0 {
			id := m.playlists[m.playlistCursor].ID
			m.loading = true
			return m, fetchPlaylistTracks(m.client, id)
		}
	}
	return m, nil
}

func (m *Model) handlePlaylistTracksKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewLibrary
		return m, nil
	case "up", "k":
		if m.ptCursor > 0 {
			m.ptCursor--
		}
	case "down", "j":
		if m.ptCursor < len(m.playlistTracks)-1 {
			m.ptCursor++
		}
	case " ":
		if len(m.playlistTracks) > 0 {
			t := m.playlistTracks[m.ptCursor]
			return m, func() tea.Msg {
				err := m.client.PlayItem("songs", t.ID)
				if err != nil {
					return errMsg{err}
				}
				return nil
			}
		}
	case "a":
		if len(m.playlistTracks) > 0 {
			t := m.playlistTracks[m.ptCursor]
			return m, func() tea.Msg {
				err := m.client.PlayLater("songs", t.ID)
				if err != nil {
					return errMsg{err}
				}
				return nil
			}
		}
	}
	return m, nil
}

func (m *Model) detailCursor() int {
	return m.searchCursor
}

func fmtDuration(ms int64) string {
	sec := ms / 1000
	min := sec / 60
	sec = sec % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}

func progressBar(current, total float64, width int) string {
	if total <= 0 {
		return ""
	}
	ratio := current / total
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}
