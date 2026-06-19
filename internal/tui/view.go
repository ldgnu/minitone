// minitone - TUI for Apple Music via Cider
// by ldgnu <ldgnu@users.noreply.github.com>

package tui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	switch m.state {
	case viewConnecting:
		return m.connectingView()
	case viewNowPlaying:
		return m.nowPlayingView()
	case viewSearch:
		return m.searchView()
	case viewSearchDetail:
		return m.detailView()
	case viewQueue:
		return m.queueView()
	case viewLibrary:
		return m.libraryView()
	case viewPlaylistTracks:
		return m.playlistTracksView()
	case viewLyrics:
		return m.lyricsView()
	default:
		return "unknown state"
	}
}

func (m Model) connectingView() string {
	var b strings.Builder
	b.WriteString(dragonLogo())
	b.WriteString("\n")
	b.WriteString(m.styles.Dimmed.Render("          ─── by ldgnu ───"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("✗ %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render("Make sure Cider is running with RPC enabled"))
		b.WriteString("\n")
		b.WriteString(m.styles.Help.Render("Settings → Connectivity → Websocket API → Enable"))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dimmed.Render("Press 'r' to retry, 'q' to quit"))
	} else {
		b.WriteString(m.styles.Dimmed.Render("Connecting to Cider..."))
	}

	return m.styles.App.Render(b.String())
}

func (m Model) nowPlayingView() string {
	var b strings.Builder

	b.WriteString(minitoneBlockLogo())
	theme := themes[m.themeIdx].Name
	b.WriteString(m.styles.Help.Render(fmt.Sprintf("  [%s]", theme)))
	b.WriteString("\n")

	np := m.nowPlaying

	if np.Track == "" {
		b.WriteString(m.styles.Info.Render("No track playing"))
	} else {
		b.WriteString(m.styles.Highlight.Render(np.Track))
		b.WriteString("\n")
		b.WriteString(m.styles.Info.Render(np.Artist))
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render(np.Album))

		eq := m.equalizer()
		if eq != "" {
			b.WriteString("\n\n")
			b.WriteString(m.styles.ProgressFull.Render(eq))
		}

		b.WriteString("\n\n")

		if np.DurationMS > 0 {
			current := int(np.CurrentSec)
			total := int(np.DurationMS / 1000)
			barWidth := 40
			if m.width > 60 {
				barWidth = m.width - 20
				if barWidth > 80 {
					barWidth = 80
				}
			}

			bar := progressBar(float64(current), float64(total), barWidth)
			timeStr := fmt.Sprintf("%d:%02d / %d:%02d",
				current/60, current%60, total/60, total%60)
			b.WriteString(m.styles.ProgressFull.Render(bar))
			b.WriteString("\n")
			b.WriteString(m.styles.Dimmed.Render(timeStr))
			b.WriteString("\n")
		}

		status := "▶"
		if !m.isPlaying {
			status = "⏸"
		}
		b.WriteString("\n")
		b.WriteString(m.styles.Info.Render(fmt.Sprintf("Vol: %d%%  %s", m.volume, status)))
	}

	if m.loading {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dimmed.Render("..."))
	}

	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render(
		"[space] play/pause  [n] next  [p] prev  [+] vol  [-] vol  " +
			"[s] search  [l] library  [w] queue  [y] lyrics  [t] theme  " +
			"[z] shuffle  [x] repeat  [ctrl+c] quit",
	))

	return m.styles.App.Render(b.String())
}

func (m Model) equalizer() string {
	if !m.isPlaying {
		return ""
	}
	blocks := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	var out strings.Builder
	for _, h := range m.eqBars {
		if h >= 0 && h < len(blocks) {
			out.WriteString(blocks[h])
		} else {
			out.WriteString("▁")
		}
		out.WriteString(" ")
	}
	return out.String()
}

func (m Model) searchView() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(" 🔍 Search "))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Info.Render("Search: "))
	b.WriteString(m.searchQuery)
	if !m.loading && m.searchQuery != "" && m.searchResults == nil {
		b.WriteString(m.styles.Dimmed.Render(" (press Enter to search)"))
	}
	b.WriteString("\n\n")

	if m.searchResults != nil {
		categories := []string{"songs", "albums", "artists", "playlists"}
		labels := map[string]string{
			"songs": "Songs", "albums": "Albums",
			"artists": "Artists", "playlists": "Playlists",
		}

		for _, cat := range categories {
			results := m.searchResults[cat]
			if len(results) == 0 {
				continue
			}

			header := fmt.Sprintf(" %s ", labels[cat])
			if cat == m.searchCategory {
				b.WriteString(m.styles.Active.Render(header))
			} else {
				b.WriteString(m.styles.Info.Render(header))
			}
			b.WriteString("\n")

			for i, r := range results {
				prefix := "  "
				if i == m.searchCursor && cat == m.searchCategory {
					prefix = "▸ "
				}
				line := fmt.Sprintf("%s%s", prefix, r.Title)
				if r.Artist != "" {
					line += fmt.Sprintf(" — %s", r.Artist)
				}
				if i == m.searchCursor && cat == m.searchCategory {
					b.WriteString(m.styles.Highlight.Render(line))
				} else {
					b.WriteString(line)
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(m.styles.Help.Render(
		"Type to search  [Enter] search  [Tab] category  " +
			"[↑↓] navigate  [Space] play  [→] detail  [q] back",
	))

	return m.styles.App.Render(b.String())
}

func (m Model) detailView() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(" 📋 " + m.detail.Title))
	b.WriteString("\n\n")

	if m.detail.Subtitle != "" {
		b.WriteString(m.styles.Info.Render(m.detail.Subtitle))
		b.WriteString("\n\n")
	}

	for i, t := range m.detail.Tracks {
		prefix := "  "
		if i == m.detailCursor() {
			prefix = "▸ "
		}
		line := fmt.Sprintf("%s%s — %s", prefix, t.Title, t.Artist)
		if t.DurationMS > 0 {
			line += fmt.Sprintf(" %s", fmtDuration(t.DurationMS))
		}
		if i == m.detailCursor() {
			b.WriteString(m.styles.Highlight.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [Space] play  [a] add to queue  [q] back"))

	return m.styles.App.Render(b.String())
}

func (m Model) queueView() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(" 📋 Queue "))
	b.WriteString("\n\n")

	if len(m.queue) == 0 {
		b.WriteString(m.styles.Info.Render("Queue is empty"))
	} else {
		for i, t := range m.queue {
			prefix := "  "
			if i == m.queueCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s — %s", prefix, t.Title, t.Artist)
			if t.DurationMS > 0 {
				line += fmt.Sprintf(" %s", fmtDuration(t.DurationMS))
			}
			if i == m.queueCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [Space] play  [q] back"))

	return m.styles.App.Render(b.String())
}

func (m Model) libraryView() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(" 📚 Library "))
	b.WriteString("\n\n")

	if len(m.playlists) == 0 {
		b.WriteString(m.styles.Info.Render("No playlists in your library"))
	} else {
		for i, p := range m.playlists {
			prefix := "  "
			if i == m.playlistCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s", prefix, p.Name)
			if i == m.playlistCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [→] view tracks  [q] back"))

	return m.styles.App.Render(b.String())
}

func (m Model) playlistTracksView() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(" 🎵 Playlist "))
	b.WriteString("\n\n")

	if len(m.playlistTracks) == 0 {
		b.WriteString(m.styles.Info.Render("No hay tracks en esta playlist"))
	} else {
		for i, t := range m.playlistTracks {
			prefix := "  "
			if i == m.ptCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s — %s", prefix, t.Title, t.Artist)
			if t.DurationMS > 0 {
				line += fmt.Sprintf(" %s", fmtDuration(t.DurationMS))
			}
			if i == m.ptCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [Space] play  [a] add to queue  [q] back"))

	return m.styles.App.Render(b.String())
}

func (m Model) lyricsView() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(" 🎤 Lyrics "))
	b.WriteString("\n\n")

	if m.lyrics == "" {
		if m.loading {
			b.WriteString(m.styles.Dimmed.Render("Loading..."))
		} else {
			b.WriteString(m.styles.Info.Render("No lyrics available"))
		}
	} else {
		lines := strings.Split(m.lyrics, "\n")
		maxLines := m.height - 6
		for _, line := range lines {
			if maxLines <= 0 {
				break
			}
			b.WriteString(line)
			b.WriteString("\n")
			maxLines--
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[q/esc] back"))

	return m.styles.App.Render(b.String())
}

func dragonLogo() string {
	return `                    ▄▄▄▄▄▄▄▄▄▄▄▄▄▄
                ▄▄▀▀▀▀░░░░░░░░░░░░▀▀▄▄
              ▄▀░░░░░░░░░░░░░░░░░░░░░░▀▄
            ▄▀░░░░░░░░░░░░░░░░░░░░░░░░░░▀▄
           █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
          █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
         █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
        █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
       █░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░█
       █░░░░░░░░░░░░░░░▄▄▄▄▄▄▄░░░░░░░░░░░░░░░█
      █░░░░░░░░░░░░▄▄▀▀░░░░░░░▀▀▄▄░░░░░░░░░░░░█
     █░░░░░░░░░░▄▀▀░░░░░░░░░░░░░░▀▄░░░░░░░░░░░█
     █░░░░░░░░░█░░░░░░░░░░░░░░░░░░░█░░░░░░░░░░█
    █░░░░░░░░░█░░░░░░░░░░░░░░░░░░░░░█░░░░░░░░░░█
    █░░░░░░░░█░░░░░░░░░░░░░░░░░░░░░░░█░░░░░░░░░█
    █░░░░░░░░█░░░░░░░░░░░░░░░░░░░░░░░█░░░░░░░░░█
    █░░░░░░░░█▄░░░░░░░░░░░░░░░░░░░░░▄█░░░░░░░░░█
    █░░░░░░░░░▀▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▀░░░░░░░░░░█
    █░░░░░░░░░█░░░░░░░░░░░░░░░░░░░░░█░░░░░░░░░░█
    █░░░░░░░░░█░░░░░░░░░░░░░░░░░░░░░█░░░░░░░░░░█
     █░░░░░░░░█░░░░░░░░░░░░░░░░░░░░░█░░░░░░░░░█
     █░░░░░░░░░█░░░░░░░░░░░░░░░░░░░█░░░░░░░░░░█
      █░░░░░░░░░▀▄░░░░░░░░░░░░░░░▄▀░░░░░░░░░░█
       █░░░░░░░░░░▀▀▄▄░░░░░░░▄▄▀▀░░░░░░░░░░░█
        ▀▄░░░░░░░░░░░░░▀▀▀▀▀░░░░░░░░░░░░░░░▄▀
          ▀▄░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▄▀
            ▀▀▄▄░░░░░░░░░░░░░░░░░░░░░░▄▄▀▀
                ▀▀▀▀▄▄▄▄▄▄▄▄▄▄▄▄▀▀▀▀`
}

func minitoneBlockLogo() string {
	return `  █▄▄▄ █▄▄▄ ▄▄▄ █   █▄▄▄ █▄▄▄ █▄▄▄
  █ █ █ █▄▄  █  █   █ █ █ █▄▄  █▄▄
  █   █ █▄▄▄ █▄▄ █▄▄▄ █▄▄▄ █▄▄▄ █▄▄▄
  ─────────── by ldgnu ────────────`
}
