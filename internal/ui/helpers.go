package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/utils"
)

// songMeta renders the dimmed secondary line of a result row.
func songMeta(song models.Song, compact bool) string {
	var parts []string
	if song.Artist != "" && song.Artist != song.Title {
		parts = append(parts, song.Artist)
	}
	if !compact && song.Album != "" {
		parts = append(parts, song.Album)
	}
	if song.Duration > 0 {
		parts = append(parts, utils.FormatDuration(song.Duration))
	}
	if !compact && song.Bitrate > 0 {
		parts = append(parts, fmt.Sprintf("%dk", song.Bitrate))
	}
	return strings.Join(parts, " · ")
}

func formatSeconds(sec int) string { return utils.FormatDuration(sec) }

// formatAgo renders a short relative timestamp ("3h ago").
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

// renderProgressBar renders a fixed-width bar (used by the tests).
func renderProgressBar(w int, ratio float64) string {
	if w < 1 {
		w = 1
	}
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
	return strings.Repeat("█", filled) + strings.Repeat("░", w-filled)
}

// overflows reports whether a rendered line is wider than the terminal.
func overflows(line string, w int) bool {
	return lipgloss.Width(line) > w
}
