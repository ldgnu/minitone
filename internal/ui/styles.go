package ui

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Header   lipgloss.Style
	Search   lipgloss.Style
	Results  lipgloss.Style
	Group    lipgloss.Style
	Item     lipgloss.Style
	Cursor   lipgloss.Style
	Selected lipgloss.Style
	Dimmed   lipgloss.Style
	Player   lipgloss.Style
	Progress lipgloss.Style
	Bar      lipgloss.Style
	Status   lipgloss.Style
	Error    lipgloss.Style
	Spinner  lipgloss.Style
	Panel    lipgloss.Style
	Help     lipgloss.Style
	Title    lipgloss.Style
	Source   lipgloss.Style
}

func NewStyles(t Theme) Styles {
	return Styles{
		Header:   lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		Search:   lipgloss.NewStyle().Foreground(t.Active),
		Results:  lipgloss.NewStyle(),
		Group:    lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		Item:     lipgloss.NewStyle().Foreground(t.Active),
		Cursor:   lipgloss.NewStyle().Foreground(t.Highlight).Bold(true),
		Selected: lipgloss.NewStyle().Foreground(t.Highlight).Bold(true),
		Dimmed:   lipgloss.NewStyle().Foreground(t.Dimmed),
		Player:   lipgloss.NewStyle().Foreground(t.Active),
		Progress: lipgloss.NewStyle().Foreground(t.Progress),
		Bar:      lipgloss.NewStyle().Foreground(t.Progress).Background(t.ProgressBg),
		Status:   lipgloss.NewStyle().Foreground(t.Dimmed),
		Error:    lipgloss.NewStyle().Foreground(t.Error).Bold(true),
		Spinner:  lipgloss.NewStyle().Foreground(t.Primary),
		Panel:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Foreground(t.Active),
		Help:     lipgloss.NewStyle().Foreground(t.Dimmed),
		Title:    lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		Source:   lipgloss.NewStyle().Foreground(t.Dimmed).Italic(true),
	}
}
