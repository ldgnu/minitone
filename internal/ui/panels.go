package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/store"
	"github.com/ldgnu/minitone/internal/utils"
)

// Panels are overlays drawn inside the body area. The header, search box,
// player bar and hints stay put, so opening a panel never loses context.

type listRow struct {
	title string
	meta  string
	badge string // e.g. the source, or "♪" for the playing item
}

// ── queue ───────────────────────────────────────────────────────────────────

func (m Model) queueRows() []listRow {
	items := m.queue.Items()
	cur := m.queue.Cursor()
	out := make([]listRow, 0, len(items))
	for i, item := range items {
		badge := "  "
		if i == cur {
			badge = m.styles.Player.Render("♪") + " "
		}
		out = append(out, listRow{
			title: item.Song.DisplayTitle(),
			meta:  songMeta(item.Song, m.compact()),
			badge: badge,
		})
	}
	return out
}

func (m Model) renderQueuePanel(w, h int) string {
	rows := m.queueRows()
	title := fmt.Sprintf(" Queue · %d track%s", len(rows), plural(len(rows)))
	if m.queue.Shuffle() {
		title += " · shuffle"
	}
	if m.queue.Repeat() != queue.RepeatOff {
		title += " · repeat " + m.queue.Repeat().String()
	}

	if len(rows) == 0 {
		return m.renderEmptyPanel(w, h, "Queue", "the queue is empty — press a on a result")
	}
	return m.renderListPanel(w, h, title, rows, m.panelCursor)
}

// ── favorites / history ─────────────────────────────────────────────────────

func (m Model) favLines() []listRow {
	items := m.favs.Items()
	out := make([]listRow, 0, len(items))
	for _, e := range items {
		out = append(out, listRow{
			title: e.Song.DisplayTitle(),
			meta:  sourceLabel(e.Song.Source),
			badge: "  ",
		})
	}
	return out
}

func (m Model) historyLines() []listRow {
	items := m.hist.Items()
	out := make([]listRow, 0, len(items))
	for _, e := range items {
		out = append(out, listRow{
			title: e.Song.DisplayTitle(),
			meta:  sourceLabel(e.Song.Source) + " · " + formatAgo(e.PlayedAt),
			badge: "  ",
		})
	}
	return out
}

// renderListPanel draws a scrollable list with a title bar.
func (m Model) renderListPanel(w, h int, title string, rows []listRow, cursor int) string {
	var b strings.Builder
	b.WriteString(m.styles.PanelTitle.Render(" " + title))
	b.WriteString("\n")

	if len(rows) == 0 {
		b.WriteString(m.styles.Dimmed.Render(" empty"))
		return b.String()
	}

	// Reserve one line for the title and keep the cursor visible.
	visible := h - 1
	if visible < 1 {
		visible = 1
	}
	start := 0
	if cursor >= visible {
		start = cursor - visible + 1
	}
	if start > len(rows)-1 {
		start = len(rows) - 1
	}

	end := start + visible
	if end > len(rows) {
		end = len(rows)
	}

	for i := start; i < end; i++ {
		row := rows[i]
		mark := "  "
		style := m.styles.Item
		if i == cursor {
			mark = m.styles.CursorMark.String() + " "
			style = m.styles.Selected
		}

		meta := row.meta
		titleW := w - lipgloss.Width(mark) - lipgloss.Width(row.badge) - lipgloss.Width(meta) - 3
		if titleW < 8 {
			titleW = 8
		}
		line := style.Render(mark+truncate(row.title, titleW)) + row.badge
		if meta != "" {
			line += m.styles.Source.Render(" " + meta)
		}
		b.WriteString(truncateWidth(line, w))
		b.WriteString("\n")
	}

	// Scroll indicator only when there is more to see.
	if len(rows) > visible {
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" %d/%d", cursor+1, len(rows))))
	}
	return b.String()
}

func (m Model) renderEmptyPanel(w, h int, title, hint string) string {
	var b strings.Builder
	b.WriteString(m.styles.PanelTitle.Render(" " + title))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" " + hint))
	return b.String()
}

// ── local library ───────────────────────────────────────────────────────────

func (m Model) renderLibraryPanel(w, h int) string {
	var b strings.Builder
	b.WriteString(m.styles.PanelTitle.Render(" " + m.libraryTitle()))
	b.WriteString("\n")
	if m.lib.status != "" {
		b.WriteString(m.styles.Dimmed.Render(" " + m.lib.status))
		b.WriteString("\n")
	}

	switch m.lib.level {
	case libSections:
		if len(m.lib.sections) == 0 {
			b.WriteString(m.styles.Dimmed.Render(" nothing indexed yet — press ctrl+s to scan"))
			return b.String()
		}
		for i, s := range m.lib.sections {
			mark := "  "
			style := m.styles.Item
			if i == m.lib.cursor {
				mark = m.styles.CursorMark.String() + " "
				style = m.styles.Selected
			}
			b.WriteString(style.Render(mark + s.Name))
			b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("  %d", s.Count)))
			b.WriteString("\n")
		}
		return b.String()

	case libNames:
		if len(m.lib.names) == 0 {
			b.WriteString(m.styles.Dimmed.Render(" nothing here"))
			return b.String()
		}
		start, end := window(m.lib.cursor, len(m.lib.names), h-2)
		for i := start; i < end; i++ {
			mark := "  "
			style := m.styles.Item
			if i == m.lib.cursor {
				mark = m.styles.CursorMark.String() + " "
				style = m.styles.Selected
			}
			name := m.lib.names[i]
			b.WriteString(style.Render(mark + truncate(name, w-4)))
			b.WriteString("\n")
		}
		return b.String()
	}

	// tracks
	if len(m.lib.items) == 0 {
		b.WriteString(m.styles.Dimmed.Render(" nothing here"))
		return b.String()
	}
	start, end := window(m.lib.cursor, len(m.lib.items), h-2)
	for i := start; i < end; i++ {
		song := m.lib.items[i]
		mark := "  "
		style := m.styles.Item
		if i == m.lib.cursor {
			mark = m.styles.CursorMark.String() + " "
			style = m.styles.Selected
		}
		meta := songMeta(song, m.compact())
		titleW := w - lipgloss.Width(mark) - lipgloss.Width(meta) - 3
		if titleW < 8 {
			titleW = 8
		}
		line := style.Render(mark + truncate(song.DisplayTitle(), titleW))
		if meta != "" {
			line += m.styles.Source.Render(" " + meta)
		}
		b.WriteString(truncateWidth(line, w))
		b.WriteString("\n")
	}
	return b.String()
}

// window returns the slice bounds that keep cursor visible in a h-line window.
func window(cursor, total, h int) (int, int) {
	if h < 1 {
		h = 1
	}
	start := 0
	if cursor >= h {
		start = cursor - h + 1
	}
	if start > total-1 {
		start = total - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + h
	if end > total {
		end = total
	}
	return start, end
}

// ── details ─────────────────────────────────────────────────────────────────

func (m Model) renderDetailsPanel(w, h int) string {
	s := m.details
	var b strings.Builder
	b.WriteString(m.styles.PanelTitle.Render(" Details"))
	b.WriteString("\n\n")

	add := func(k, v string) {
		if v == "" {
			return
		}
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" %-10s", k)))
		b.WriteString(truncate(v, maxInt(w-13, 10)))
		b.WriteString("\n")
	}

	add("title", s.Title)
	add("artist", s.Artist)
	add("album", s.Album)
	add("source", sourceLabel(s.Source))
	add("duration", utils.FormatDurationFull(s.Duration))
	if s.Bitrate > 0 {
		add("bitrate", fmt.Sprintf("%d kbps", s.Bitrate))
	}
	if s.Format != "" {
		add("format", s.Format)
	}
	if s.Genre != "" {
		add("genre", s.Genre)
	}
	if s.Year > 0 {
		add("year", fmt.Sprintf("%d", s.Year))
	}
	if s.FilePath != "" {
		add("file", s.FilePath)
	}
	if s.URL != "" && s.Source != models.SourceLocal {
		add("url", s.URL)
	}

	if m.favs.Contains(s) {
		b.WriteString("\n")
		b.WriteString(m.styles.Item.Render(" ★ in favorites"))
		b.WriteString("\n")
	}
	return b.String()
}

// ── help ────────────────────────────────────────────────────────────────────

func (m Model) renderHelpPanel(w, h int) string {
	var b strings.Builder
	b.WriteString(m.styles.PanelTitle.Render(" Shortcuts"))
	b.WriteString("\n")

	rows := m.helpRows()
	start, end := window(m.panelCursor, len(rows), h-1)
	for i := start; i < end; i++ {
		row := rows[i]
		if row.desc == "" {
			b.WriteString(m.styles.Group.Render(" " + row.key))
			b.WriteString("\n")
			continue
		}
		b.WriteString(m.styles.HintKey.Render(fmt.Sprintf(" %-16s", row.key)))
		b.WriteString(m.styles.HintText.Render(row.desc))
		b.WriteString("\n")
	}
	return b.String()
}

// ── playlists ───────────────────────────────────────────────────────────────

func (m Model) playlistRows() []listRow {
	if m.playls == nil {
		return nil
	}
	lists := m.playls.List()
	out := make([]listRow, 0, len(lists))
	for _, pl := range lists {
		meta := fmt.Sprintf("%d tracks", len(pl.Tracks))
		if pl.AutoRefresh {
			meta += " · auto"
		}
		out = append(out, listRow{title: pl.Name, meta: meta, badge: "  "})
	}
	return out
}

func (m Model) playlistTrackRows() ([]listRow, string) {
	pl := m.currentPlaylist()
	if pl == nil {
		return nil, "Playlists"
	}
	out := make([]listRow, 0, len(pl.Tracks))
	for _, s := range pl.Tracks {
		out = append(out, listRow{title: s.DisplayTitle(), meta: songMeta(s, m.compact()), badge: "  "})
	}
	return out, pl.Name
}

func (m Model) currentPlaylist() *store.Playlist {
	if m.playls == nil {
		return nil
	}
	return m.playls.GetAt(m.plIndex)
}

func (m Model) renderPlaylistsPanel(w, h int) string {
	if m.plLevel == 1 {
		rows, name := m.playlistTrackRows()
		if len(rows) == 0 {
			return m.renderEmptyPanel(w, h, name, "empty — press r to refresh from YouTube")
		}
		return m.renderListPanel(w, h, " "+name+fmt.Sprintf(" · %d", len(rows)), rows, m.panelCursor)
	}
	rows := m.playlistRows()
	if len(rows) == 0 {
		return m.renderEmptyPanel(w, h, "Playlists", "no playlists — set taste_genres/artists in config.json")
	}
	return m.renderListPanel(w, h, fmt.Sprintf(" Playlists · %d", len(rows)), rows, m.panelCursor)
}

// ── panel key handling ──────────────────────────────────────────────────────

// handlePanelKey routes keys while an overlay is open.
func (m Model) handlePanelKey(key string) (tea.Model, tea.Cmd) {
	k := m.keys

	switch key {
	case k.Quit.key:
		m.saveSession()
		return m, tea.Quit
	case "esc":
		if m.panel == PanelLibrary && m.lib.level != libSections {
			m.lib.reset()
			return m, nil
		}
		if m.panel == PanelPlaylists && m.plLevel == 1 {
			m.plLevel = 0
			m.panelCursor = m.plIndex
			return m, nil
		}
		m.lib.open = false
		m.panel = PanelNone
		// Back exactly where we came from.
		m.focus = m.focusBeforePanel
		return m, nil
	}

	switch m.panel {
	case PanelHelp:
		switch key {
		case k.Help.key, "?":
			m.panel = PanelNone
			m.focus = m.focusBeforePanel
		case "up", "k":
			m.movePanelCursor(-1)
		case "down", "j":
			m.movePanelCursor(1)
		case "q":
			m.panel = PanelNone
			m.saveSession()
			return m, tea.Quit
		}
		return m, nil

	case PanelDetails:
		return m, nil

	case PanelLibrary:
		return m.handleLibraryPanelKey(key)

	case PanelPlaylists:
		return m.handlePlaylistsPanelKey(key)
	}

	// Shared list behaviour for queue / favorites / history.
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
		return m.playPanelItem()
	case k.Enqueue.key:
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
	case k.ClearQueue.key:
		if m.panel == PanelQueue {
			m.clearQueue()
		}
	case k.MoveUp.key:
		m.movePanelItem(-1)
	case k.MoveDown.key:
		m.movePanelItem(1)
	case "q":
		m.panel = PanelNone
		m.saveSession()
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleLibraryPanelKey(key string) (tea.Model, tea.Cmd) {
	k := m.keys
	n := m.lib.len()

	switch key {
	case "up", "k":
		m.lib.cursor = clampIndex(m.lib.cursor-1, n)
	case "down", "j":
		m.lib.cursor = clampIndex(m.lib.cursor+1, n)
	case k.Enter.key:
		cmd := m.enterLibrary()
		return m, cmd
	case k.EnqueueAll.key:
		m.enqueueLibraryItems()
	case k.Delete.key:
		m.forgetLibraryItem()
	case k.Rescan.key:
		m.startScanCmd()
	case k.QuitKey.key:
		m.panel = PanelNone
		m.lib.open = false
		m.saveSession()
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) movePanelCursor(delta int) {
	n := m.panelLen()
	m.panelCursor = clampIndex(m.panelCursor+delta, n)
}

func clampIndex(i, n int) int {
	if n <= 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// panelSong returns the song at an index of the open panel.
func (m Model) panelSong(i int) (models.Song, bool) {
	switch m.panel {
	case PanelQueue:
		items := m.queue.Items()
		if i >= 0 && i < len(items) {
			return items[i].Song, true
		}
	case PanelFavorites:
		if s := m.favs.Get(i); s != nil {
			return *s, true
		}
	case PanelHistory:
		if s := m.hist.Get(i); s != nil {
			return *s, true
		}
	case PanelPlaylists:
		if m.plLevel == 1 {
			if pl := m.currentPlaylist(); pl != nil && i >= 0 && i < len(pl.Tracks) {
				return pl.Tracks[i], true
			}
		}
	}
	return models.Song{}, false
}

// playPanelItem plays the highlighted item of the open panel.
func (m Model) playPanelItem() (tea.Model, tea.Cmd) {
	song, ok := m.panelSong(m.panelCursor)
	if !ok {
		return m, nil
	}
	m.lib.open = false
	m.panel = PanelNone
	cmd := m.playFromPanel(song)
	return m, cmd
}

func (m *Model) favoritePanelItem() {
	song, ok := m.panelSong(m.panelCursor)
	if !ok {
		return
	}
	if m.favs.Toggle(song) {
		m.notice = okNotice("★ "+song.DisplayTitle()+" added", "", "")
	} else {
		m.notice = okNotice("☆ removed from favorites", "", "")
	}
	m.clampCursors()
}

func (m *Model) deletePanelItem() {
	switch m.panel {
	case PanelQueue:
		if m.queue.Remove(m.panelCursor) {
			if m.queue.Len() == 0 {
				m.panel = PanelNone
				m.notice = okNotice("queue cleared", "", "")
				return
			}
			m.notice = okNotice("removed from queue", "", "")
		}
	case PanelFavorites:
		if m.favs.RemoveAt(m.panelCursor) {
			if m.favs.Len() == 0 {
				m.panel = PanelNone
			}
			m.notice = okNotice("removed from favorites", "", "")
		}
	case PanelHistory:
		if m.hist.RemoveAt(m.panelCursor) {
			if m.hist.Len() == 0 {
				m.panel = PanelNone
			}
			m.notice = okNotice("removed from history", "", "")
		}
	case PanelPlaylists:
		if m.plLevel == 1 {
			// Remove one track from the playlist (keeps the list itself).
			if pl := m.currentPlaylist(); pl != nil {
				i := m.panelCursor
				if i >= 0 && i < len(pl.Tracks) {
					tracks := append(append([]models.Song{}, pl.Tracks[:i]...), pl.Tracks[i+1:]...)
					m.playls.SetTracks(pl.ID, tracks)
					m.notice = okNotice("removed from playlist", "", "")
				}
			}
		} else if m.playls.RemoveAt(m.panelCursor) {
			if m.playls.Len() == 0 {
				m.panel = PanelNone
			}
			m.notice = okNotice("playlist removed", "", "")
		}
	}
	m.clampCursors()
}

func (m *Model) movePanelItem(delta int) {
	if m.panel != PanelQueue {
		return
	}
	from := m.panelCursor
	to := from + delta
	if to < 0 || to >= m.queue.Len() {
		return
	}
	if m.queue.Move(from, to) {
		m.panelCursor = to
		m.notice = okNotice("moved in queue", "", "")
	}
}

func (m *Model) enqueuePanelAll() {
	var songs []models.Song
	switch m.panel {
	case PanelQueue:
		return
	case PanelFavorites:
		songs = m.favs.Songs()
	case PanelHistory:
		songs = m.hist.Songs()
	case PanelPlaylists:
		if m.plLevel == 1 {
			if pl := m.currentPlaylist(); pl != nil {
				songs = pl.Tracks
			}
		} else {
			// Enqueue every track of every playlist.
			for _, pl := range m.playls.List() {
				songs = append(songs, pl.Tracks...)
			}
		}
	default:
		return
	}
	n := 0
	for _, s := range songs {
		if m.queue.AddUnique(s) {
			n++
		}
	}
	if n == 0 {
		m.notice = okNotice("everything is already queued", "", "")
		return
	}
	m.notice = okNotice(fmt.Sprintf("queued %d tracks (%d total)", n, m.queue.Len()), "", "")
}
