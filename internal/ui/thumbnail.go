package ui

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"time"

	"github.com/ldgnu/minitone/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

type thumbMsg struct {
	url string
	img image.Image
	err error
}

func fetchThumb(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return thumbMsg{url: url, err: err}
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (thumbnail)")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return thumbMsg{url: url, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return thumbMsg{url: url, err: fmt.Errorf("thumbnail status %d", resp.StatusCode)}
		}
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return thumbMsg{url: url, err: err}
		}
		return thumbMsg{url: url, img: img}
	}
}

// selectedSong returns the currently highlighted search result, or nil.
func (m Model) selectedSong() *models.Song {
	if m.searchGroup < 0 || m.searchGroup >= len(m.searchResults.Groups) {
		return nil
	}
	items := m.searchResults.Groups[m.searchGroup].Items
	if m.searchCursor < 0 || m.searchCursor >= len(items) {
		return nil
	}
	return &items[m.searchCursor]
}

// maybeFetchThumb triggers an async thumbnail load for the selected YouTube
// result when it is not already cached.
func (m Model) maybeFetchThumb() (Model, tea.Cmd) {
	if m.thumbs == nil {
		m.thumbs = map[string]image.Image{}
	}
	s := m.selectedSong()
	if s == nil || s.Source != models.SourceYouTube || s.Thumbnail == "" {
		m.thumbURL = ""
		return m, nil
	}
	m.thumbURL = s.Thumbnail
	if _, ok := m.thumbs[s.Thumbnail]; ok {
		return m, nil
	}
	return m, fetchThumb(s.Thumbnail)
}

func (m Model) renderThumbPanel(w, h int) string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(" Preview"))
	b.WriteString("\n")

	if m.thumbURL == "" {
		return m.styles.Panel.Width(w).Height(h).Render(b.String())
	}
	img, ok := m.thumbs[m.thumbURL]
	if !ok {
		b.WriteString(m.styles.Dimmed.Render(" loading…"))
		return m.styles.Panel.Width(w).Height(h).Render(b.String())
	}

	cols := w - 2
	rows := h - 3
	if cols < 4 {
		cols = 4
	}
	if rows < 2 {
		rows = 2
	}
	b.WriteString(brailleImage(img, cols, rows))
	return m.styles.Panel.Width(w).Height(h).Render(b.String())
}

// brailleImage renders an image as braille/ANSI truecolor art.
// Each cell is 2 dots wide x 4 dots tall (Unicode braille block).
func brailleImage(src image.Image, cols, rows int) string {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return ""
	}

	out := make([]byte, 0, cols*(rows+1)*6)
	var sb strings.Builder

	for ry := 0; ry < rows; ry++ {
		lastColor := ""
		for rx := 0; rx < cols; rx++ {
			// map cell -> 2x4 source pixels
			var dots [4][2]struct {
				on   bool
				r, g, b uint8
			}
			var ar, ag, ab, al, an uint32
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					sx := b.Min.X + (rx*2+dx)*sw/(cols*2)
					sy := b.Min.Y + (ry*4+dy)*sh/(rows*4)
					if sx >= b.Max.X {
						sx = b.Max.X - 1
					}
					if sy >= b.Max.Y {
						sy = b.Max.Y - 1
					}
					c := src.At(sx, sy)
					r, g, bl, _ := c.RGBA()
					r8, g8, bl8 := uint8(r>>8), uint8(g>>8), uint8(bl>>8)
					dots[dy][dx].r, dots[dy][dx].g, dots[dy][dx].b = r8, g8, bl8
					ar += uint32(r8)
					ag += uint32(g8)
					ab += uint32(bl8)
					al += uint32(luminance(r8, g8, bl8))
					an++
				}
			}

			// adaptive threshold: a dot is "on" when brighter than the cell average
			avg := uint8(al / an)
			cellOn := false
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					if luminance(dots[dy][dx].r, dots[dy][dx].g, dots[dy][dx].b) >= int(avg) {
						dots[dy][dx].on = true
						cellOn = true
					}
				}
			}

			if !cellOn {
				sb.WriteString(" ")
				lastColor = ""
				continue
			}

			// average color of the cell for the foreground
			cr, cg, cb := uint8(ar/an), uint8(ag/an), uint8(ab/an)
			color := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", cr, cg, cb)
			if color != lastColor {
				sb.WriteString(color)
				lastColor = color
			}
			sb.WriteRune(brailleCell(dots))
		}
		sb.WriteString("\x1b[0m\n")
		out = append(out, []byte(sb.String())...)
		sb.Reset()
	}
	return string(out)
}

func luminance(r, g, b uint8) int {
	return int(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
}

// brailleCell maps the 2x4 dot grid to a Unicode braille code point.
func brailleCell(dots [4][2]struct {
	on          bool
	r, g, b uint8
}) rune {
	var v rune
	if dots[0][0].on {
		v |= 0x01
	}
	if dots[0][1].on {
		v |= 0x02
	}
	if dots[1][0].on {
		v |= 0x04
	}
	if dots[1][1].on {
		v |= 0x08
	}
	if dots[2][0].on {
		v |= 0x10
	}
	if dots[2][1].on {
		v |= 0x20
	}
	if dots[3][0].on {
		v |= 0x40
	}
	if dots[3][1].on {
		v |= 0x80
	}
	return 0x2800 + v
}
