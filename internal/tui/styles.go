// minitone - TUI pa' controlar Apple Music desde Cider
// Creado por ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

package tui

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	App      lipgloss.Style
	Title    lipgloss.Style
	Info     lipgloss.Style
	Help     lipgloss.Style
	Error    lipgloss.Style
	Highlight lipgloss.Style
	Active   lipgloss.Style
	Dimmed   lipgloss.Style
	ProgressFull lipgloss.Style
	ProgressEmpty lipgloss.Style
}

func NewStyles(t Theme) Styles {
	return Styles{
		App:      lipgloss.NewStyle().Padding(1, 2),
		Title:    lipgloss.NewStyle().Bold(true).Foreground(t.Primary),
		Info:     lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")),
		Help:     lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")),
		Error:    lipgloss.NewStyle().Foreground(t.Error),
		Highlight: lipgloss.NewStyle().Foreground(t.Highlight),
		Active: lipgloss.NewStyle().Foreground(t.Active).Background(t.Primary).Padding(0, 1),
		Dimmed:   lipgloss.NewStyle().Foreground(t.Dimmed),
		ProgressFull: lipgloss.NewStyle().Foreground(t.Progress),
		ProgressEmpty: lipgloss.NewStyle().Foreground(t.ProgressBg),
	}
}
