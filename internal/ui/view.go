package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/search"
	"github.com/ldgnu/minitone/internal/utils"
)

// Layout heights for the main screen.
const (
	headerLines = 1
	sourceLines = 1
	searchLines = 1
	hintLines   = 1
	playerLines = 3 // 2 in compact mode
)

// compactWidth is below which secondary columns are dropped.
const (
	compactWidth  = 76
	compactHeight = 18
)

// compact reports whether the terminal is too small for the full layout.
func (m Model) compact() bool {
	return m.width < compactWidth || m.height < compactHeight
}

// View renders the whole screen: header, source selector, search box, results
// (or a panel), the persistent player and contextual hints.
func (m Model) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	playerH := playerLines
	if m.compact() {
		playerH = 2
	}

	resultsH := h - headerLines - sourceLines - searchLines - playerH - hintLines
	if resultsH < 1 {
		resultsH = 1
	}

	parts := []string{
		m.styles.Header.Width(w).Render(m.renderHeader()),
		m.styles.Dimmed.Width(w).Render(m.renderSources()),
		m.renderSearch(),
		m.renderBody(resultsH),
		m.renderPlayer(playerH),
		m.renderHints(),
	}

	// Final safety net: whatever the individual renderers produced, no line may
	// exceed the terminal width and no more lines than the terminal has may be
	// emitted. A narrow window must never wrap and destroy the layout.
	return clipBlock(lipgloss.JoinVertical(lipgloss.Left, parts...), w, h)
}

// clipBlock trims a rendered block to w columns and h rows.
func clipBlock(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, line := range lines {
		lines[i] = truncateWidth(line, w)
	}
	return strings.Join(lines, "\n")
}

// ── header ──────────────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	left := "♫ minitone"
	if m.favs != nil && m.favs.Len() > 0 {
		left += fmt.Sprintf(" ★%d", m.favs.Len())
	}

	right := themes[m.themeIdx].Name
	if !m.compact() {
		right = "minitone " + right
	}
	// In a very narrow terminal the name is dropped rather than squeezing the
	// title out of the line.
	if lipgloss.Width(left)+lipgloss.Width(right)+2 > m.width {
		right = ""
	}
	return fitTwo(left, right, m.width)
}

// ── source selector ─────────────────────────────────────────────────────────

// renderSources draws the source/filter row. The active filter is highlighted
// so the user always knows which sources the search will hit.
func (m Model) renderSources() string {
	type chip struct {
		label string
		style lipgloss.Style
		off   bool
		hint  string
	}

	// Compare on the canonical name so the /local prefix and the "local" chip
	// highlight the same thing.
	active := m.activeSource()
	if active == "" {
		active = "all"
	}

	chips := []chip{{label: "all"}}
	for _, s := range m.sources {
		c := chip{label: sourceAlias(s.Name)}
		if !s.Enabled {
			c.off = true
			c.hint = s.Hint
		}
		chips = append(chips, c)
	}

	var b strings.Builder
	width := 0
	room := m.width
	if !m.compact() {
		// Leave space for the right-hand status.
		room = m.width - 12
	}
	for _, c := range chips {
		selected := c.label == sourceAlias(active) || c.label == active
		style := m.styles.Chip
		if selected {
			style = m.styles.ChipActive
		}

		text := " " + c.label + " "
		if c.off {
			// A source that cannot work is dimmed and marked, never hidden:
			// the user needs to know *why* YouTube is empty.
			style = m.styles.ChipOff
			text = " " + c.label + "✕ "
		}
		rendered := style.Render(text)
		if width+lipgloss.Width(rendered) > room {
			// Keep the active chip visible even when space is tight.
			if selected && width > 0 {
				break
			}
			if width == 0 {
				rendered = style.Render(truncate(" "+c.label+" ", maxInt(room, 4)))
			} else {
				break
			}
		}
		b.WriteString(rendered)
		width += lipgloss.Width(rendered)
	}
	line := b.String()

	// The scanned/library indicator sits on the right when there is room.
	right := ""
	if m.libraryScanner != nil {
		if m.libraryScanner.Scanning() {
			right = "scanning…"
		} else if n := m.libraryScanner.Len(); n > 0 {
			right = fmt.Sprintf("%d local", n)
		}
	}
	if right == "" && m.search.running {
		right = m.spinner() + " searching"
	}
	if right != "" {
		line = fitTwo(line, m.styles.Dimmed.Render(right), m.width)
	}
	return line
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m Model) spinner() string {
	if len(spinnerFrames) == 0 {
		return ""
	}
	return spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
}

// ── search box ──────────────────────────────────────────────────────────────

func (m Model) renderSearch() string {
	q := m.search.query

	prefix := "› "
	if m.focus == FocusList {
		prefix = "  "
	}
	if src := m.search.source; src != "" {
		// Show the friendly alias, not a truncated source name: "/local" and
		// "/libr" are confusing side by side with "/library".
		prefix = "/" + sourceAlias(src) + " "
	}

	cursor := "▌"
	if m.focus == FocusSearch {
		cursor = "█"
	}
	line := prefix + q + cursor

	// Keep the tail of the query visible when it does not fit.
	if lipgloss.Width(line) > m.width {
		runes := []rune(line)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > m.width {
			runes = runes[1:]
		}
		line = string(runes)
	}

	style := m.styles.SearchBox
	if m.focus == FocusList {
		style = m.styles.SearchBoxDim
	}

	// Right side: live state of the search itself.
	right := ""
	switch {
	case m.search.running && len(m.search.pendingSources()) > 0:
		right = fmt.Sprintf("%s %d source%s", m.spinner(), len(m.search.pendingSources()), plural(len(m.search.pendingSources())))
	case m.search.total() > 0:
		right = fmt.Sprintf("%d result%s", m.search.total(), plural(m.search.total()))
	}
	if right != "" {
		left := style.Render(line)
		return fitTwo(left, m.styles.Dimmed.Render(right), m.width)
	}
	return style.Width(m.width).Render(line)
}

// ── body (results or panel) ─────────────────────────────────────────────────

func (m Model) renderBody(h int) string {
	switch m.panel {
	case PanelQueue:
		return m.renderQueuePanel(m.width, h)
	case PanelFavorites:
		return m.renderListPanel(m.width, h, "★ Favorites", m.favLines(), m.panelCursor)
	case PanelHistory:
		return m.renderListPanel(m.width, h, "◷ History", m.historyLines(), m.panelCursor)
	case PanelPlaylists:
		return m.renderPlaylistsPanel(m.width, h)
	case PanelLibrary:
		return m.renderLibraryPanel(m.width, h)
	case PanelDetails:
		return m.renderDetailsPanel(m.width, h)
	case PanelHelp:
		return m.renderHelpPanel(m.width, h)
	}
	return m.renderResults(m.width, h)
}

// renderResults draws the grouped result list, or the welcome/now-playing view
// when there is nothing to show.
//
// The active group scrolls with the cursor so a long result set never hides
// the highlighted track below the fold.
func (m Model) renderResults(w, h int) string {
	groups := m.search.groups()

	if len(groups) == 0 {
		if m.search.running {
			return m.center(m.spinner()+" searching "+quote(m.search.term)+"…", w)
		}
		if m.search.term != "" {
			return m.renderNoResults(w, h)
		}
		return m.renderIdle(w, h)
	}

	// Flatten into rows, then window them so the highlighted track is always
	// visible even when a group is longer than the body.
	rows := make([]resultRow, 0, 16)
	for gi, group := range groups {
		rows = append(rows, resultRow{header: group.Name, count: len(group.Items), isHeader: true, group: gi})
		for si, song := range group.Items {
			rows = append(rows, resultRow{
				song:     song,
				group:    gi,
				selected: gi == m.search.group && si == m.search.cursor,
			})
		}
	}

	// Index of the highlighted row (there always is one while results exist).
	cursorRow := 0
	for i, r := range rows {
		if r.selected {
			cursorRow = i
			break
		}
	}

	start, end := window(cursorRow, len(rows), h)
	var b strings.Builder
	for i := start; i < end; i++ {
		r := rows[i]
		if r.isHeader {
			text := fmt.Sprintf("▸ %s (%d)", r.header, r.count)
			if r.group == m.search.group {
				b.WriteString(m.styles.Group.Render(text))
			} else {
				b.WriteString(m.styles.Dimmed.Render(text))
			}
		} else {
			b.WriteString(m.renderResultRow(r.groupName(groups), r.song, r.selected, w))
		}
		b.WriteString("\n")
	}

	// Tell the user there is more to see when the list does not fit.
	if len(rows) > h {
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" %d/%d", cursorRow+1, len(rows))))
	}
	return b.String()
}

// resultRow is one line of the flattened results list.
type resultRow struct {
	song     models.Song
	header   string
	count    int
	group    int
	selected bool
	isHeader bool
}

// groupName returns the source label of a group index.
func (r resultRow) groupName(groups []models.SearchResultGroup) string {
	if r.group >= 0 && r.group < len(groups) {
		return groups[r.group].Name
	}
	return ""
}

func (m Model) renderResultRow(source string, song models.Song, selected bool, w int) string {
	mark := "  "
	style := m.styles.Item
	if selected {
		mark = m.styles.CursorMark.String() + " "
		style = m.styles.Selected
	}

	star := ""
	if m.favs != nil && m.favs.Contains(song) {
		star = "★ "
	}

	// The source badge only matters for the favorites group, where the rows
	// keep their original source.
	badge := ""
	if !m.compact() && source == search.SourceFavorites {
		badge = " [" + sourceLabel(song.Source) + "]"
	}

	title := star + song.DisplayTitle()
	meta := songMeta(song, m.compact())
	avail := lipgloss.Width(mark) + 2

	titleW := w - avail - lipgloss.Width(meta) - 2
	if titleW < 8 {
		titleW = 8
	}
	title = truncate(title, titleW)

	line := style.Render(mark + title)
	if meta != "" {
		line += m.styles.Source.Render(" " + meta + badge)
	}
	return truncateWidth(line, w)
}

// renderNoResults explains an empty result set instead of leaving a void.
func (m Model) renderNoResults(w, h int) string {
	var b strings.Builder
	b.WriteString(m.styles.Dimmed.Render(" no results for " + quote(m.search.term)))
	b.WriteString("\n")

	var fails []string
	for _, f := range m.search.failuresList() {
		fails = append(fails, f.Source+" unavailable")
	}
	if len(fails) > 0 {
		b.WriteString(m.styles.Error.Render(" " + strings.Join(fails, " · ")))
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render(" [r] retry  ·  esc clear  ·  tab change source"))
		return b.String()
	}

	b.WriteString(m.styles.Dimmed.Render(" try another spelling, or switch source with tab"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" /youtube term  ·  /radio term  ·  /local term"))
	return b.String()
}

// renderIdle shows the welcome / now-playing screen when nothing is searched.
func (m Model) renderIdle(w, h int) string {
	var b strings.Builder

	song := m.currentSong()
	if song.Title != "" {
		b.WriteString(m.styles.Title.Render(" now playing"))
		b.WriteString("\n")
		b.WriteString(" " + song.DisplayTitle())
		b.WriteString("\n")
		if song.Artist != "" {
			b.WriteString(m.styles.Item.Render(" " + song.Artist))
			b.WriteString("\n")
		}
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" from %s", sourceLabel(song.Source))))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dimmed.Render(" type to search · a to queue · f to favorite"))
		return b.String()
	}

	b.WriteString(m.styles.Title.Render(" minitone"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dimmed.Render(" type to search YouTube, Radio, Navidrome or your library"))
	b.WriteString("\n\n")

	if m.mpvMissing {
		b.WriteString(m.styles.Error.Render(" ! mpv is not installed — nothing can play"))
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render("   install it with your package manager, then restart"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(m.styles.Item.Render(" enter     play the highlighted result"))
		b.WriteString("\n")
		b.WriteString(m.styles.Item.Render(" a / A     add one / add all to the queue"))
		b.WriteString("\n")
		b.WriteString(m.styles.Item.Render(" space     play / pause    n / p  next / previous"))
		b.WriteString("\n")
	}

	b.WriteString(m.styles.Dimmed.Render(" ctrl+j queue · ctrl+l library · ctrl+f favorites · ctrl+p playlists"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" ? / ctrl+/   all shortcuts"))

	// First-run hints: only the things the user cannot guess.
	var warn []string
	if m.ytdlpMissing {
		warn = append(warn, "yt-dlp not installed → YouTube is off")
	}
	if m.libraryScanner != nil && m.libraryScanner.Len() == 0 && !m.libraryScanner.Scanning() {
		if len(m.libraryScanner.Dirs()) == 0 {
			warn = append(warn, "no library_paths configured → ctrl+s to set one")
		} else {
			warn = append(warn, "library is empty → ctrl+s to rescan")
		}
	}
	if len(warn) > 0 {
		b.WriteString("\n\n")
		for _, wmsg := range warn {
			b.WriteString(m.styles.Dimmed.Render(" · " + wmsg))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// ── player (always visible) ─────────────────────────────────────────────────

// renderPlayer draws the persistent bottom player: song, artist, state,
// position, duration, progress, volume, shuffle, repeat and queue size.
func (m Model) renderPlayer(height int) string {
	var status player.Status
	if m.player != nil {
		status = m.player.Status()
	}

	if m.player != nil && m.player.VideoPlaying() {
		title := m.player.VideoTitle()
		if title == "" {
			title = "video"
		}
		return m.styles.PlayerBox.Width(m.width).Render(
			fitTwo(" 🎬 "+title, "video mode", m.width))
	}

	if status.Song.Empty() {
		return m.styles.PlayerBox.Width(m.width).Render(
			m.styles.Dimmed.Render(" ■ nothing playing"))
	}

	song := m.currentSong()
	title := song.DisplayTitle()
	if title == "" {
		title = status.Song.Title
	}
	artist := song.Artist
	if artist == "" {
		artist = status.Song.Artist
	}

	star := ""
	if m.favs != nil && song.Title != "" && m.favs.Contains(song) {
		star = "★ "
	}

	// Line 1: state · title — artist ................ position / duration
	pos := ""
	switch {
	case status.Duration > 0:
		pos = fmt.Sprintf("%s / %s",
			utils.FormatDuration(int(status.Elapsed)),
			utils.FormatDuration(int(status.Duration)))
	default:
		pos = "live"
	}

	left := fmt.Sprintf(" %s %s%s", status.State.Icon(), star, title)
	if artist != "" {
		left += " — " + artist
	}
	right := pos

	line1 := fitTwo(m.styles.Player.Render(truncate(left, m.width-lipgloss.Width(right)-2)),
		m.styles.Dimmed.Render(right), m.width)

	if height <= 2 {
		// Compact: progress plus the essentials on one line.
		barW := m.width - lipgloss.Width(m.playerMeta()) - 3
		if barW < 8 {
			barW = 8
		}
		ratio := 0.0
		if status.Duration > 0 {
			ratio = status.Elapsed / status.Duration
		}
		bar := m.renderProgress(barW, ratio)
		return m.styles.PlayerBox.Width(m.width).Render(line1 + "\n" + fitTwo(" "+bar, m.playerMeta(), m.width))
	}

	// Line 2: the progress bar, full width.
	bar := m.renderProgress(m.width-4, progressRatio(status))
	line2 := " " + bar

	// Line 3: volume, modes and queue size.
	line3 := m.playerMeta()

	return m.styles.PlayerBox.Width(m.width).Render(line1 + "\n" + line2 + "\n" + line3)
}

// playerMeta is the compact status strip: volume, shuffle, repeat, queue.
func (m Model) playerMeta() string {
	vol := 70
	if m.player != nil {
		vol = m.player.Volume()
	}
	parts := []string{"vol " + m.renderVolumeBar(8) + " " + itoa(vol) + "%"}
	if m.queue.Shuffle() {
		parts = append(parts, "⇄ shuffle")
	}
	if m.queue.Repeat() != queue.RepeatOff {
		label := "↻ " + m.queue.Repeat().String()
		if m.queue.Repeat() == queue.RepeatOne {
			label = "↻1"
		}
		parts = append(parts, label)
	}
	parts = append(parts, "queue "+itoa(m.queue.Len()))
	if m.compact() {
		return truncateWidth(strings.Join(parts, " "), m.width)
	}
	if src := sourceLabel(m.currentSong().Source); src != "" {
		parts = append(parts, src)
	}
	return truncateWidth(" "+strings.Join(parts, "  "), m.width)
}

func progressRatio(status player.Status) float64 {
	if status.Duration <= 0 {
		return 0
	}
	return status.Elapsed / status.Duration
}

func (m Model) renderProgress(w int, ratio float64) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(w))
	if filled > w {
		filled = w
	}
	if filled < 0 {
		filled = 0
	}

	if filled > 0 && m.player != nil && m.player.Status().State == player.StatePlaying {
		// A knob makes the position obvious without any animation.
		return m.styles.ProgressFilled.Render(strings.Repeat("█", filled-1)) +
			m.styles.ProgressKnob.Render("█") +
			m.styles.ProgressEmpty.Render(strings.Repeat("░", w-filled))
	}
	return m.styles.ProgressFilled.Render(strings.Repeat("█", filled)) +
		m.styles.ProgressEmpty.Render(strings.Repeat("░", w-filled))
}

// renderVolumeBar is a tiny 8-cell volume meter.
func (m Model) renderVolumeBar(w int) string {
	if w < 4 {
		w = 4
	}
	vol := 70
	if m.player != nil {
		vol = m.player.Volume()
	}
	filled := vol * w / 100
	if filled > w {
		filled = w
	}
	if filled < 0 {
		filled = 0
	}
	return m.styles.VolumeOn.Render(strings.Repeat("▮", filled)) +
		m.styles.VolumeOff.Render(strings.Repeat("▯", w-filled))
}

// ── hints / notices ─────────────────────────────────────────────────────────

// renderHints shows either the active notice (with its action) or the most
// relevant shortcuts for the current view. Never the full key list.
func (m Model) renderHints() string {
	if m.notice.active() {
		return m.renderNotice()
	}
	hints := m.hintsFor()
	if len(hints) == 0 {
		return m.styles.Dimmed.Width(m.width).Render(" " + strings.Join(
			[]string{"/term  /youtube term  /radio term  /local term"}, "  "))
	}
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, m.styles.HintKey.Render(h.key)+" "+m.styles.HintText.Render(h.label))
	}
	line := " " + strings.Join(parts, m.styles.Dimmed.Render(" · "))
	return m.styles.Dimmed.Width(m.width).Render(truncateWidth(line, m.width))
}

func (m Model) renderNotice() string {
	style := m.styles.Notice
	icon := "·"
	switch m.notice.kind {
	case noticeWarn:
		style = m.styles.NoticeWarn
		icon = "!"
	case noticeError:
		style = m.styles.NoticeError
		icon = "✗"
	}

	detail := m.notice.detail
	if detail == "" {
		detail = m.notice.action
	}
	left := fmt.Sprintf(" %s %s", icon, m.notice.text)
	if detail != "" {
		left += " — " + detail
	}
	if m.notice.action != "" && detail != m.notice.action {
		left += "  " + m.notice.action
	}
	return style.Width(m.width).Render(truncateWidth(left, m.width))
}

// ── text helpers ────────────────────────────────────────────────────────────

// fitTwo places left and right on one line, padded to exactly w cells.
//
// The padding is emitted plain (unstyled) on purpose: a styled filler would
// paint the terminal background across the line and break the "clean terminal"
// look. Long left-hand text is clipped rather than allowed to push the right
// side off screen.
func fitTwo(left, right string, w int) string {
	if w <= 0 {
		return ""
	}
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if lw+rw+1 > w {
		avail := w - rw - 1
		if avail < 4 {
			return truncateWidth(right, w)
		}
		left = truncate(left, avail)
		lw = lipgloss.Width(left)
	}
	gap := w - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// truncate shortens a plain string to w display cells, adding an ellipsis.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > w {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return strings.Repeat(" ", w)
	}
	return string(runes) + "…"
}

// truncateWidth clips an already-styled string to w display cells.
func truncateWidth(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	// Cut conservatively on runes, keeping ANSI sequences intact.
	var b strings.Builder
	width := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
		}
		if inEscape {
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		cw := lipgloss.Width(string(r))
		if width+cw > w {
			break
		}
		b.WriteRune(r)
		width += cw
	}
	return b.String()
}

func (m Model) center(s string, w int) string {
	pad := (w - lipgloss.Width(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

func quote(s string) string {
	if s == "" {
		return ""
	}
	return "“" + s + "”"
}

// sourceAlias returns the short, user-facing name for a source ("local",
// "youtube"), used in the search box and the source selector.
func sourceAlias(name string) string {
	switch name {
	case search.SourceYouTube:
		return "youtube"
	case search.SourceRadio:
		return "radio"
	case search.SourceNavidrome:
		return "navidrome"
	case search.SourceLibrary:
		return "local"
	case search.SourceFavorites:
		return "fav"
	default:
		return strings.ToLower(name)
	}
}

func sourceLabel(s models.SourceType) string {
	switch s {
	case models.SourceYouTube:
		return "youtube"
	case models.SourceRadio:
		return "radio"
	case models.SourceNavidrome:
		return "navidrome"
	case models.SourceLocal:
		return "local"
	case "":
		return ""
	default:
		return string(s)
	}
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
