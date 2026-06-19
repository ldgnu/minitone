package tui

import (
	"fmt"
	"math/rand/v2"
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
		m.state = viewArtists
		m.loading = true
		return m, fetchArtists(m.client)

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case artistsMsg:
		if msg.err == nil {
			m.artists = msg.artists
			m.artistCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case albumsMsg:
		if msg.err == nil {
			m.artistAlbums = msg.albums
			m.aaCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case albumSongsMsg:
		if msg.err == nil {
			m.albumSongs = msg.songs
			m.albumCursor = 0
			m.albumTitle = msg.title
			m.albumSubtitle = msg.artist
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case searchMsg:
		if msg.err == nil {
			m.searchResults = msg.songs
			m.searchCursor = 0
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

	case playlistSongsMsg:
		if msg.err == nil {
			m.playlistSongs = msg.songs
			m.plsongCursor = 0
		} else {
			m.err = msg.err
		}
		m.loading = false
		return m, nil

	case nowPlayingMsg:
		m.loading = false
		return m, nil

	case eqTickMsg:
		if m.player.IsPlaying() {
			for i := range m.eqBars {
				m.eqBars[i] = rand.IntN(7) + 1
			}
		} else {
			m.eqBars = [8]int{}
		}
		if m.state == viewNowPlaying {
			return m, eqTick()
		}
		return m, nil

	case tickMsg:
		if m.connected {
			m.currentSong = m.player.NowPlaying()
			m.isPlaying = m.player.IsPlaying()
			m.volume = m.player.GetVolume()
			if m.state == viewNowPlaying {
				return m, tea.Batch(tick(), eqTick())
			}
		}
		return m, tick()
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case viewConnecting:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.player.Close()
			return m, tea.Quit
		}
		return m, nil

	case viewNowPlaying:
		return m.handleNowPlayingKeys(msg)

	case viewArtists:
		return m.handleArtistKeys(msg)

	case viewArtistAlbums:
		return m.handleAlbumsKeys(msg)

	case viewAlbumSongs:
		return m.handleAlbumSongsKeys(msg)

	case viewSearch:
		return m.handleSearchKeys(msg)

	case viewPlaylists:
		return m.handlePlaylistKeys(msg)

	case viewPlaylistSongs:
		return m.handlePlaylistSongsKeys(msg)

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
		m.player.Close()
		return m, tea.Quit
	case " ":
		return m, func() tea.Msg {
			m.player.TogglePause()
			m.isPlaying = m.player.IsPlaying()
			return nil
		}
	case "n":
		return m, func() tea.Msg {
			m.player.Next()
			return nil
		}
	case "p":
		return m, func() tea.Msg {
			m.player.Previous()
			return nil
		}
	case "s":
		m.state = viewSearch
		m.searchQuery = ""
		m.searchResults = nil
		return m, nil
	case "a":
		m.state = viewArtists
		m.loading = true
		return m, fetchArtists(m.client)
	case "l":
		m.state = viewPlaylists
		m.loading = true
		return m, fetchPlaylists(m.client)
	case "+", "=":
		m.volume += 5
		if m.volume > 100 {
			m.volume = 100
		}
		m.player.SetVolume(m.volume)
		return m, nil
	case "-":
		m.volume -= 5
		if m.volume < 0 {
			m.volume = 0
		}
		m.player.SetVolume(m.volume)
		return m, nil
	case "t":
		m.themeIdx = (m.themeIdx + 1) % len(themes)
		m.styles = NewStyles(themes[m.themeIdx])
		return m, nil
	}
	return m, nil
}

func (m *Model) handleArtistKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewNowPlaying
		return m, nil
	case "up", "k":
		if m.artistCursor > 0 {
			m.artistCursor--
		}
	case "down", "j":
		if m.artistCursor < len(m.artists)-1 {
			m.artistCursor++
		}
	case "right", "l", "enter":
		if len(m.artists) > 0 {
			id := m.artists[m.artistCursor].ID
			m.state = viewArtistAlbums
			m.loading = true
			return m, fetchAlbums(m.client, id)
		}
	}
	return m, nil
}

func (m *Model) handleAlbumsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewArtists
		return m, nil
	case "up", "k":
		if m.aaCursor > 0 {
			m.aaCursor--
		}
	case "down", "j":
		if m.aaCursor < len(m.artistAlbums)-1 {
			m.aaCursor++
		}
	case "right", "l", "enter":
		if len(m.artistAlbums) > 0 {
			album := m.artistAlbums[m.aaCursor]
			m.state = viewAlbumSongs
			m.loading = true
			return m, fetchAlbumSongs(m.client, album.ID, album.Name, album.Artist)
		}
	case " ":
		if len(m.artistAlbums) > 0 {
			album := m.artistAlbums[m.aaCursor]
			m.loading = true
			return m, func() tea.Msg {
				songs, err := m.client.GetAlbum(album.ID)
				if err != nil {
					return errMsg{err}
				}
				m.player.PlayPlaylist(songs, 0)
				m.state = viewNowPlaying
				m.loading = false
				return nil
			}
		}
	}
	return m, nil
}

func (m *Model) handleAlbumSongsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewArtistAlbums
		return m, nil
	case "up", "k":
		if m.albumCursor > 0 {
			m.albumCursor--
		}
	case "down", "j":
		if m.albumCursor < len(m.albumSongs)-1 {
			m.albumCursor++
		}
	case " ":
		if len(m.albumSongs) > 0 {
			idx := m.albumCursor
			m.player.PlayPlaylist(m.albumSongs, idx)
			m.state = viewNowPlaying
			return m, nil
		}
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
	case "up", "k":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
	case "down", "j":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
		}
	case " ":
		if len(m.searchResults) > 0 {
			s := m.searchResults[m.searchCursor]
			m.player.PlaySong(s)
			m.player.SetPlaylist(m.searchResults, m.searchCursor)
			m.state = viewNowPlaying
			return m, nil
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}
	return m, nil
}

func (m *Model) handlePlaylistKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "right", "l", "enter":
		if len(m.playlists) > 0 {
			id := m.playlists[m.playlistCursor].ID
			m.state = viewPlaylistSongs
			m.loading = true
			return m, fetchPlaylistSongs(m.client, id)
		}
	}
	return m, nil
}

func (m *Model) handlePlaylistSongsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = viewPlaylists
		return m, nil
	case "up", "k":
		if m.plsongCursor > 0 {
			m.plsongCursor--
		}
	case "down", "j":
		if m.plsongCursor < len(m.playlistSongs)-1 {
			m.plsongCursor++
		}
	case " ":
		if len(m.playlistSongs) > 0 {
			idx := m.plsongCursor
			m.player.PlayPlaylist(m.playlistSongs, idx)
			m.state = viewNowPlaying
			return m, nil
		}
	}
	return m, nil
}

func fmtDuration(sec int) string {
	if sec <= 0 {
		return "--:--"
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func progressBarFilled(current, total int, width int) string {
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	ratio := float64(current) / float64(total)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
