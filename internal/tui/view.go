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
	case viewArtists:
		return m.artistsView()
	case viewArtistAlbums:
		return m.albumsView()
	case viewAlbumSongs:
		return m.albumSongsView()
	case viewSearch:
		return m.searchView()
	case viewPlaylists:
		return m.playlistsView()
	case viewPlaylistSongs:
		return m.playlistSongsView()
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
		b.WriteString(m.styles.Help.Render("Set NAVIDROME_URL, NAVIDROME_USER and NAVIDROME_PASS"))
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render("Press 'q' to quit"))
	} else {
		b.WriteString(m.styles.Dimmed.Render("Connecting to Navidrome..."))
	}

	return m.styles.App.Render(b.String())
}

func (m Model) nowPlayingView() string {
	var b strings.Builder

	b.WriteString(minitoneBlockLogo())
	theme := themes[m.themeIdx].Name
	b.WriteString(m.styles.Help.Render(fmt.Sprintf("  [%s]", theme)))
	b.WriteString("\n")

	song := m.currentSong
	if song == nil {
		b.WriteString(m.styles.Info.Render("No track playing"))
		b.WriteString("\n")
		b.WriteString(m.styles.Help.Render("Browse artists [a] or search [s] to start"))
	} else {
		b.WriteString(m.styles.Highlight.Render(song.Title))
		b.WriteString("\n")
		b.WriteString(m.styles.Info.Render(song.Artist))
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render(song.Album))
		b.WriteString("\n\n")

		eq := m.equalizer()
		if eq != "" {
			b.WriteString(m.styles.ProgressFull.Render(eq))
			b.WriteString("\n\n")
		}

		barWidth := m.barWidth()
		bar := m.player.ProgressBar(barWidth)
		b.WriteString(m.styles.ProgressFull.Render(bar))
		b.WriteString("\n")
		b.WriteString(m.styles.Dimmed.Render(m.player.FormatProgress()))
		b.WriteString("\n\n")

		status := "▶"
		if !m.isPlaying {
			status = "⏸"
		}
		b.WriteString(m.styles.Info.Render(fmt.Sprintf("Vol: %d%%  %s", m.volume, status)))
	}

	if m.loading {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dimmed.Render("..."))
	}

	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render(
		"[space] play/pause  [n] next  [p] prev  [+] vol  [-] vol  " +
			"[a] artists  [s] search  [l] playlists  [t] theme  [ctrl+c] quit",
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

func (m Model) barWidth() int {
	w := 40
	if m.width > 60 {
		w = m.width - 20
		if w > 80 {
			w = 80
		}
	}
	return w
}

func (m Model) artistsView() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" 🎤 Artists "))
	b.WriteString("\n\n")

	if len(m.artists) == 0 {
		b.WriteString(m.styles.Info.Render("No artists found"))
	} else {
		for i, a := range m.artists {
			prefix := "  "
			if i == m.artistCursor {
				prefix = "▸ "
			}
			line := prefix + a.Name
			if i == m.artistCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [→/enter] albums  [q] back"))
	return m.styles.App.Render(b.String())
}

func (m Model) albumsView() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" 💿 Albums "))
	b.WriteString("\n\n")

	if len(m.artistAlbums) == 0 {
		b.WriteString(m.styles.Info.Render("No albums"))
	} else {
		for i, a := range m.artistAlbums {
			prefix := "  "
			if i == m.aaCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s (%d)", prefix, a.Name, a.Year)
			if i == m.aaCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [→/enter] tracks  [space] play all  [q] back"))
	return m.styles.App.Render(b.String())
}

func (m Model) albumSongsView() string {
	var b strings.Builder
	title := m.albumTitle
	if title == "" {
		title = "Album"
	}
	b.WriteString(m.styles.Title.Render(" 🎵 " + title))
	b.WriteString("\n")

	if m.albumSubtitle != "" {
		b.WriteString(m.styles.Info.Render(m.albumSubtitle))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.albumSongs) == 0 {
		b.WriteString(m.styles.Info.Render("No tracks"))
	} else {
		for i, s := range m.albumSongs {
			prefix := "  "
			if i == m.albumCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s  %s", prefix, s.Title, fmtDuration(s.Duration))
			if i == m.albumCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [space] play  [q] back"))
	return m.styles.App.Render(b.String())
}

func (m Model) searchView() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" 🔍 Search "))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Info.Render("Search: "))
	b.WriteString(m.searchQuery)
	if !m.loading && m.searchQuery != "" && m.searchResults == nil {
		b.WriteString(m.styles.Dimmed.Render(" (press Enter)"))
	}
	b.WriteString("\n\n")

	if m.searchResults != nil {
		for i, s := range m.searchResults {
			prefix := "  "
			if i == m.searchCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s — %s", prefix, s.Title, s.Artist)
			if s.Duration > 0 {
				line += " " + fmtDuration(s.Duration)
			}
			if i == m.searchCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render(
		"Type to search  [Enter] go  [↑↓] navigate  [space] play  [q] back",
	))
	return m.styles.App.Render(b.String())
}

func (m Model) playlistsView() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" 📋 Playlists "))
	b.WriteString("\n\n")

	if len(m.playlists) == 0 {
		b.WriteString(m.styles.Info.Render("No playlists"))
	} else {
		for i, p := range m.playlists {
			prefix := "  "
			if i == m.playlistCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s (%d songs)", prefix, p.Name, p.SongCount)
			if i == m.playlistCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [→/enter] view  [q] back"))
	return m.styles.App.Render(b.String())
}

func (m Model) playlistSongsView() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" 🎵 Playlist "))
	b.WriteString("\n\n")

	if len(m.playlistSongs) == 0 {
		b.WriteString(m.styles.Info.Render("No tracks"))
	} else {
		for i, s := range m.playlistSongs {
			prefix := "  "
			if i == m.plsongCursor {
				prefix = "▸ "
			}
			line := fmt.Sprintf("%s%s — %s", prefix, s.Title, s.Artist)
			if s.Duration > 0 {
				line += " " + fmtDuration(s.Duration)
			}
			if i == m.plsongCursor {
				b.WriteString(m.styles.Highlight.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("[↑↓] navigate  [space] play  [q] back"))
	return m.styles.App.Render(b.String())
}

func (m Model) lyricsView() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" 🎤 Lyrics "))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Info.Render("No lyrics available"))
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
