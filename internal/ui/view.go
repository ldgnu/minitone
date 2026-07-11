package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/utils"
)

func (m Model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	headerH := 1
	searchH := 1
	playerH := 1
	statusH := 1
	footerH := 2
	resultsH := m.height - headerH - searchH - playerH - statusH - footerH
	if resultsH < 3 {
		resultsH = 3
	}

	header := m.renderHeader()
	searchLine := m.renderSearch()
	playerLine := m.renderPlayer()
	status := m.renderStatus()
	footer := m.renderFooter()

	hR := m.styles.Header.Width(m.width).Render(header)
	sR := m.styles.Search.Width(m.width).Render(searchLine)
	pR := m.styles.Player.Width(m.width).Render(playerLine)
	stR := m.styles.Status.Width(m.width).Render(status)
	fR := m.styles.Dimmed.Width(m.width).Render(footer)

	var results string
	switch m.showPanel {
	case PanelHelp:
		results = m.styles.Results.Width(m.width).Height(resultsH).Render(m.renderHelp(resultsH))
	case PanelFavorites:
		results = m.styles.Results.Width(m.width).Height(resultsH).Render(m.renderListPanel("★ Favorites", m.favListLines(), resultsH))
	case PanelHistory:
		results = m.styles.Results.Width(m.width).Height(resultsH).Render(m.renderListPanel("◷ History", m.histListLines(), resultsH))
	case PanelQueue:
		leftW := (m.width * 2) / 3
		rightW := m.width - leftW - 1
		if rightW < 20 {
			rightW = 20
			leftW = m.width - rightW - 1
		}
		base := m.renderResults(resultsH)
		if leftW < 10 {
			results = m.styles.Results.Width(m.width).Height(resultsH).Render(base)
		} else {
			resultsR := m.styles.Results.Width(leftW).Height(resultsH).Render(base)
			panelR := m.renderQueuePanel(rightW, resultsH)
			results = lipgloss.JoinHorizontal(lipgloss.Top, resultsR, panelR)
		}
	default:
		results = m.styles.Results.Width(m.width).Height(resultsH).Render(m.renderResults(resultsH))
	}

	return lipgloss.JoinVertical(lipgloss.Left, hR, sR, results, pR, stR, fR)
}

func (m Model) renderHeader() string {
	left := "♫ minitone"
	if m.favs != nil && m.favs.Len() > 0 {
		left += fmt.Sprintf(" · ★%d", m.favs.Len())
	}
	right := themes[m.themeIdx].Name
	sp := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if sp < 1 {
		sp = 1
	}
	return left + strings.Repeat(" ", sp) + right
}

func (m Model) renderSearch() string {
	prefix := " › "
	q := m.searchQuery
	cursor := "█"
	line := prefix + q + cursor

	if lipgloss.Width(line) > m.width {
		runes := []rune(prefix + q + cursor)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > m.width {
			runes = runes[1:]
		}
		line = string(runes)
	}
	return line
}

func (m Model) renderResults(h int) string {
	if m.searchQuery == "" && len(m.searchResults.Groups) == 0 {
		s := m.player.Status()
		if s.Song.Title != "" {
			return m.renderNowPlaying(h)
		}
		return m.renderWelcome(h)
	}

	if m.searching && len(m.searchResults.Groups) == 0 {
		return m.styles.Dimmed.Render(" searching...")
	}

	if m.searchQuery != "" && !m.searching && len(m.searchResults.Groups) == 0 {
		return m.styles.Dimmed.Render(" no results — try another query")
	}

	var b strings.Builder
	lines := 0

	for gi, group := range m.searchResults.Groups {
		if lines >= h {
			break
		}
		if len(group.Items) == 0 {
			continue
		}

		sourceStyle := m.styles.Group
		if gi == m.searchGroup {
			sourceStyle = m.styles.Cursor
		}
		groupName := fmt.Sprintf("▸ %s (%d)", group.Name, len(group.Items))
		b.WriteString(sourceStyle.Render(groupName))
		b.WriteString("\n")
		lines++

		for si, song := range group.Items {
			if lines >= h {
				break
			}
			prefix := "  "
			style := m.styles.Item
			if gi == m.searchGroup && si == m.searchCursor {
				prefix = "▸ "
				style = m.styles.Selected
			}

			star := ""
			if m.favs != nil && m.favs.Contains(song) {
				star = "★ "
			}

			title := star + song.DisplayTitle()
			meta := ""
			if song.Artist != "" && song.Artist != song.Title {
				meta += " · " + song.Artist
			}
			if song.Duration > 0 {
				meta += " " + utils.FormatDuration(song.Duration)
			}
			if song.Bitrate > 0 {
				meta += fmt.Sprintf(" %dk", song.Bitrate)
			}

			maxW := m.width - 6
			if maxW < 10 {
				maxW = 10
			}
			line := title
			if lipgloss.Width(line+meta) > maxW {
				for len([]rune(line)) > 3 && lipgloss.Width(line+meta) > maxW {
					r := []rune(line)
					line = string(r[:len(r)-1])
				}
				line += "…"
			}
			rendered := style.Render(prefix+line) + m.styles.Source.Render(meta)
			b.WriteString(rendered)
			b.WriteString("\n")
			lines++
		}
	}

	return b.String()
}

func (m Model) renderNowPlaying(h int) string {
	var b strings.Builder
	s := m.player.Status()

	star := ""
	if m.favs != nil && m.lastSong.Title != "" && m.favs.Contains(m.lastSong) {
		star = " ★"
	}

	b.WriteString(m.styles.Title.Render(" " + s.Song.Title + star))
	b.WriteString("\n")
	if s.Song.Artist != "" {
		b.WriteString(m.styles.Item.Render(fmt.Sprintf(" %s", s.Song.Artist)))
		b.WriteString("\n")
	}
	if s.Song.Album != "" {
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" %s", s.Song.Album)))
		b.WriteString("\n")
	}
	b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" source: %s", s.Song.Source)))
	b.WriteString("\n\n")

	if s.Duration > 0 {
		bw := m.width - 8
		if bw < 10 {
			bw = 10
		}
		ratio := s.Elapsed / s.Duration
		progress := m.styles.Progress.Render(renderProgressBar(bw, ratio))
		b.WriteString(" " + progress)
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" %s / %s",
			utils.FormatDuration(int(s.Elapsed)),
			utils.FormatDuration(int(s.Duration)),
		)))
		b.WriteString("\n")
	} else if s.State == player.StatePlaying {
		b.WriteString(m.styles.Dimmed.Render(" live stream"))
		b.WriteString("\n")
	}

	vol := m.player.Volume()
	b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf(" vol: %d%%", vol)))

	if m.queue.Len() > 0 {
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("  queue: %d", m.queue.Len())))
	}
	if m.favs != nil && m.favs.Len() > 0 {
		b.WriteString(m.styles.Dimmed.Render(fmt.Sprintf("  ★%d", m.favs.Len())))
	}
	if m.queue.Shuffle() {
		b.WriteString(m.styles.Dimmed.Render("  shuffle"))
	}
	if m.queue.Repeat() != queue.RepeatOff {
		b.WriteString(m.styles.Dimmed.Render("  repeat:" + m.queue.Repeat().String()))
	}

	return b.String()
}

func (m Model) renderWelcome(h int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" minitone"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dimmed.Render(" type to search · YouTube / Radio / Navidrome / Library"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dimmed.Render(" enter     play selected"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" ctrl+a    favorite selected / playing track"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" ctrl+f    favorites · ctrl+h history · ctrl+j queue"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" space     play/pause · ctrl+n/p next/prev"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" ctrl+t theme · ctrl+v video · ctrl+r repeat · ctrl+u shuffle · ctrl+/ help"))
	return b.String()
}

func (m Model) renderPlayer() string {
	if m.player.VideoPlaying() {
		left := " 🎬 " + m.player.VideoTitle()
		right := "video"
		sp := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 1
		if sp < 1 {
			sp = 1
		}
		return m.styles.Player.Render(left + strings.Repeat(" ", sp) + right)
	}

	status := m.player.Status()
	if status.Song.Title == "" {
		return m.styles.Dimmed.Render(" ♪ no track playing")
	}

	stateIcon := "▶"
	switch status.State {
	case player.StatePaused:
		stateIcon = "⏸"
	case player.StateStopped:
		stateIcon = "⏹"
	}

	star := ""
	if m.favs != nil && m.lastSong.Title != "" && m.favs.Contains(m.lastSong) {
		star = "★ "
	}

	left := fmt.Sprintf(" %s %s%s", stateIcon, star, status.Song.Title)
	if status.Song.Artist != "" {
		left += " · " + status.Song.Artist
	}

	right := ""
	if status.Duration > 0 {
		right = fmt.Sprintf("%s/%s",
			utils.FormatDuration(int(status.Elapsed)),
			utils.FormatDuration(int(status.Duration)),
		)
	} else {
		right = "live"
	}
	right = fmt.Sprintf("%s  %d%%", right, status.Volume)

	sp := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if sp < 1 {
		sp = 1
		for lipgloss.Width(left)+lipgloss.Width(right)+1 > m.width && len([]rune(left)) > 8 {
			r := []rune(left)
			left = string(r[:len(r)-1])
		}
		left += "…"
		sp = m.width - lipgloss.Width(left) - lipgloss.Width(right)
		if sp < 1 {
			sp = 1
		}
	}
	return left + strings.Repeat(" ", sp) + right
}

func (m Model) renderStatus() string {
	if m.err != nil {
		return m.styles.Error.Render(fmt.Sprintf(" ! %v", m.err))
	}
	if m.statusText != "" {
		return m.styles.Status.Render(" " + m.statusText)
	}
	return m.styles.Dimmed.Render(" enter play · f fav · h history · ctrl+f favorites · q quit")
}

func (m Model) renderQueuePanel(w, h int) string {
	items := m.queue.Items()
	cur := m.queue.Cursor()

	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" Queue"))
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString(m.styles.Dimmed.Render(" empty"))
		return m.styles.Panel.Width(w).Height(h).Render(b.String())
	}

	for i, item := range items {
		if i >= h-2 {
			break
		}
		prefix := "  "
		style := m.styles.Item
		if i == m.panelCursor {
			prefix = "▸ "
			style = m.styles.Selected
		}
		mark := " "
		if i == cur {
			mark = "♪"
		}
		line := mark + " " + item.Song.DisplayTitle()
		maxW := w - 6
		if maxW < 5 {
			maxW = 5
		}
		if lipgloss.Width(line) > maxW {
			r := []rune(line)
			for len(r) > 1 && lipgloss.Width(string(r)) > maxW-1 {
				r = r[:len(r)-1]
			}
			line = string(r) + "…"
		}
		b.WriteString(style.Render(prefix + line))
		b.WriteString("\n")
	}

	return m.styles.Panel.Width(w).Height(h).Render(b.String())
}

type listLine struct {
	title string
	meta  string
}

func (m Model) favListLines() []listLine {
	if m.favs == nil {
		return nil
	}
	var out []listLine
	for _, e := range m.favs.Items() {
		meta := string(e.Song.Source)
		if e.Song.Artist != "" {
			meta = e.Song.Artist + " · " + meta
		}
		out = append(out, listLine{title: e.Song.DisplayTitle(), meta: meta})
	}
	return out
}

func (m Model) histListLines() []listLine {
	if m.hist == nil {
		return nil
	}
	var out []listLine
	for _, e := range m.hist.Items() {
		ago := formatAgo(e.PlayedAt)
		meta := string(e.Song.Source) + " · " + ago
		if e.Song.Artist != "" {
			meta = e.Song.Artist + " · " + meta
		}
		out = append(out, listLine{title: e.Song.DisplayTitle(), meta: meta})
	}
	return out
}

func (m Model) renderListPanel(title string, lines []listLine, h int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" " + title))
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render(" enter play · f/d remove · esc close"))
	b.WriteString("\n\n")

	if len(lines) == 0 {
		b.WriteString(m.styles.Dimmed.Render(" empty"))
		return b.String()
	}

	maxLines := h - 4
	if maxLines < 1 {
		maxLines = 1
	}
	for i, line := range lines {
		if i >= maxLines {
			break
		}
		prefix := "  "
		style := m.styles.Item
		if i == m.panelCursor {
			prefix = "▸ "
			style = m.styles.Selected
		}
		text := line.title
		meta := ""
		if line.meta != "" {
			meta = " · " + line.meta
		}
		maxW := m.width - 4
		if lipgloss.Width(text+meta) > maxW {
			for len([]rune(text)) > 3 && lipgloss.Width(text+meta) > maxW {
				r := []rune(text)
				text = string(r[:len(r)-1])
			}
			text += "…"
		}
		b.WriteString(style.Render(prefix+text) + m.styles.Source.Render(meta))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderHelp(h int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" Help"))
	b.WriteString("\n\n")

	helpItems := []string{
		"type              search (YouTube / Radio / Navidrome / Library)",
		"enter             play selected (+ add to queue)",
		"esc               back: close panel / clear search",
		"tab / shift+tab   next / prev source group",
		"↑ ↓ ← →           navigate / seek / volume",
		"space             play / pause (when not searching)",
		"ctrl+t            cycle theme",
		"ctrl+f            favorites panel",
		"ctrl+a            favorite selected / playing track",
		"ctrl+h            history panel",
		"ctrl+j            queue panel (d delete, f favorite)",
		"ctrl+s            stop",
		"ctrl+n / ctrl+p   next / previous in queue",
		"ctrl+m            mute toggle",
		"ctrl+r            repeat: off → all → one",
		"ctrl+u            shuffle toggle",
		"ctrl+v            video mode (YouTube plays with video)",
		"ctrl+/            toggle this help",
		"ctrl+c            quit",
	}

	for _, line := range helpItems {
		b.WriteString(m.styles.Help.Render(" " + line))
		b.WriteString("\n")
	}

	return m.styles.Panel.Width(m.width).Height(h).Render(b.String())
}

func (m Model) renderFooter() string {
	line1 := "ctrl+t theme · ctrl+f fav · ctrl+j queue · ctrl+h hist · ctrl+s stop · space play · enter play"
	line2 := "ctrl+n/p next/prev · ctrl+r repeat · ctrl+u shuffle · ctrl+v video"
	if m.videoMode {
		line2 += " [ON]"
	}
	line2 += " · ctrl+/ help · esc back · ctrl+c quit"
	return line1 + "\n" + line2
}

func renderProgressBar(w int, ratio float64) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	if w < 1 {
		w = 1
	}
	filled := int(ratio * float64(w))
	if filled > w {
		filled = w
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", w-filled)
}

func formatAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}
