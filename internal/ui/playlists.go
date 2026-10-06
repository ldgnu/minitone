package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/models"
)

// playlistRefreshMsg carries refreshed tracks for one playlist.
type playlistRefreshMsg struct {
	id     string
	tracks []models.Song
	err    error
}

// handlePlaylistsPanelKey routes keys while the playlists overlay is open.
// Level 0 = playlists, Level 1 = tracks of the selected playlist.
func (m Model) handlePlaylistsPanelKey(key string) (tea.Model, tea.Cmd) {
	k := m.keys
	n := m.panelLen()

	switch key {
	case "up", "k":
		m.movePanelCursor(-1)
	case "down", "j":
		m.movePanelCursor(1)
	case "home", "g":
		m.panelCursor = 0
	case "end", "G":
		m.panelCursor = maxInt(n-1, 0)
	case k.Enter.key:
		if m.plLevel == 0 {
			// Drill into the playlist.
			if m.playls != nil && m.panelCursor >= 0 && m.panelCursor < m.playls.Len() {
				m.plIndex = m.panelCursor
				m.plLevel = 1
				m.panelCursor = 0
			}
			return m, nil
		}
		return m.playPanelItem()
	case k.Enqueue.key:
		if m.plLevel == 0 {
			// Enqueue the whole highlighted playlist.
			if pl := m.playls.GetAt(m.panelCursor); pl != nil {
				nq := 0
				for _, s := range pl.Tracks {
					if m.queue.AddUnique(s) {
						nq++
					}
				}
				m.notice = okNotice(itoa(nq)+" tracks queued from "+pl.Name, "", "")
			}
			return m, nil
		}
		if song, ok := m.panelSong(m.panelCursor); ok {
			m.queue.Add(song)
			m.queue.SetCursor(m.queue.Len() - 1)
			m.notice = okNotice("queued "+song.DisplayTitle(), "", "")
		}
	case k.EnqueueAll.key:
		m.enqueuePanelAll()
	case k.Favorite.key:
		m.favoritePanelItem()
	case k.Delete.key:
		m.deletePanelItem()
	case "r":
		return m, m.refreshCurrentPlaylistCmd()
	case "R":
		return m, m.refreshAllPlaylistsCmd()
	case "q":
		m.panel = PanelNone
		m.saveSession()
		return m, tea.Quit
	}
	return m, nil
}

// refreshCurrentPlaylistCmd refreshes the selected playlist (or the open one)
// from YouTube in the background.
func (m Model) refreshCurrentPlaylistCmd() tea.Cmd {
	if m.playls == nil {
		return nil
	}
	var id string
	if m.plLevel == 1 {
		if pl := m.currentPlaylist(); pl != nil {
			id = pl.ID
		}
	} else if pl := m.playls.GetAt(m.panelCursor); pl != nil {
		id = pl.ID
	}
	if id == "" {
		return nil
	}
	return m.refreshPlaylistCmd(id)
}

// refreshAllPlaylistsCmd refreshes every auto playlist.
func (m Model) refreshAllPlaylistsCmd() tea.Cmd {
	if m.playls == nil {
		return nil
	}
	lists := m.playls.List()
	cmds := make([]tea.Cmd, 0, len(lists))
	for _, pl := range lists {
		if pl.AutoRefresh {
			cmds = append(cmds, m.refreshPlaylistCmd(pl.ID))
		}
	}
	if len(cmds) == 0 {
		m.notice = infoNotice("nothing to refresh", "no auto playlists", "")
		return nil
	}
	m.notice = infoNotice("refreshing playlists…", "tracks arrive as YouTube answers", "")
	return tea.Batch(cmds...)
}

// refreshPlaylistCmd resolves one playlist query to YouTube songs.
func (m Model) refreshPlaylistCmd(id string) tea.Cmd {
	pl := m.playls.Get(id)
	if pl == nil {
		return nil
	}
	query := pl.Query
	yt := m.youtubeClient
	if yt == nil || !yt.Available() {
		m.notice = infoNotice("yt-dlp missing — cannot refresh", "install yt-dlp for YouTube playlists", "")
		return nil
	}
	m.notice = infoNotice("refreshing "+pl.Name+"…", "searching YouTube for "+query, "")
	return func() tea.Msg {
		ctx := teaContext()
		tracks, err := refreshPlaylistTracks(ctx, *pl, yt)
		return playlistRefreshMsg{id: id, tracks: tracks, err: err}
	}
}

// onPlaylistRefresh stores refreshed tracks and notifies.
func (m *Model) onPlaylistRefresh(msg playlistRefreshMsg) {
	if msg.err != nil || len(msg.tracks) == 0 {
		m.notice = infoNotice("refresh found nothing", "try again later (r)", "")
		return
	}
	if m.playls == nil {
		return
	}
	m.playls.SetTracks(msg.id, msg.tracks)
	if pl := m.playls.Get(msg.id); pl != nil {
		m.notice = okNotice(fmt.Sprintf("updated %s · %d tracks", pl.Name, len(msg.tracks)), "", "")
	}
	m.clampCursors()
}
