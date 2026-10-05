package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds every visual element of the TUI.
//
// Contrast rules: the progress bar and the selection use high-contrast
// colours; metadata uses the dimmed colour; nothing else is decorated.
type Styles struct {
	Header lipgloss.Style

	// source selector
	Chip       lipgloss.Style
	ChipActive lipgloss.Style
	ChipOff    lipgloss.Style

	// search box
	SearchBox    lipgloss.Style
	SearchBoxDim lipgloss.Style

	// results
	Group      lipgloss.Style
	Item       lipgloss.Style
	Selected   lipgloss.Style
	CursorMark lipgloss.Style
	Source     lipgloss.Style
	Title      lipgloss.Style

	// player
	PlayerBox      lipgloss.Style
	Player         lipgloss.Style
	ProgressFilled lipgloss.Style
	ProgressEmpty  lipgloss.Style
	ProgressKnob   lipgloss.Style
	VolumeOn       lipgloss.Style
	VolumeOff      lipgloss.Style

	// status / notices
	HintKey     lipgloss.Style
	HintText    lipgloss.Style
	Notice      lipgloss.Style
	NoticeWarn  lipgloss.Style
	NoticeError lipgloss.Style

	// panels
	PanelTitle lipgloss.Style
	Panel      lipgloss.Style

	Help   lipgloss.Style
	Error  lipgloss.Style
	Dimmed lipgloss.Style
}

func NewStyles(t Theme) Styles {
	return Styles{
		Header: lipgloss.NewStyle().Foreground(t.Primary).Bold(true),

		Chip:       lipgloss.NewStyle().Foreground(t.Dimmed),
		ChipActive: lipgloss.NewStyle().Foreground(t.Highlight).Bold(true),
		// Unavailable sources are dimmed but still readable.
		ChipOff: lipgloss.NewStyle().Foreground(t.Dimmed).Faint(true),

		SearchBox:    lipgloss.NewStyle().Foreground(t.Active).Bold(true),
		SearchBoxDim: lipgloss.NewStyle().Foreground(t.Dimmed),

		Group:      lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		Item:       lipgloss.NewStyle().Foreground(t.Active),
		Selected:   lipgloss.NewStyle().Foreground(t.Highlight).Bold(true),
		CursorMark: lipgloss.NewStyle().Foreground(t.Highlight).Bold(true),
		Source:     lipgloss.NewStyle().Foreground(t.Dimmed),
		Title:      lipgloss.NewStyle().Foreground(t.Primary).Bold(true),

		PlayerBox:      lipgloss.NewStyle(),
		Player:         lipgloss.NewStyle().Foreground(t.Active).Bold(true),
		ProgressFilled: lipgloss.NewStyle().Foreground(t.Progress),
		ProgressEmpty:  lipgloss.NewStyle().Foreground(t.ProgressBg),
		ProgressKnob:   lipgloss.NewStyle().Foreground(t.Highlight),
		VolumeOn:       lipgloss.NewStyle().Foreground(t.Progress),
		VolumeOff:      lipgloss.NewStyle().Foreground(t.ProgressBg),

		HintKey:     lipgloss.NewStyle().Foreground(t.Highlight),
		HintText:    lipgloss.NewStyle().Foreground(t.Dimmed),
		Notice:      lipgloss.NewStyle().Foreground(t.Active),
		NoticeWarn:  lipgloss.NewStyle().Foreground(t.Primary),
		NoticeError: lipgloss.NewStyle().Foreground(t.Error).Bold(true),

		PanelTitle: lipgloss.NewStyle().Foreground(t.Highlight).Bold(true),
		Panel:      lipgloss.NewStyle().Foreground(t.Active),

		Help:   lipgloss.NewStyle().Foreground(t.Dimmed),
		Error:  lipgloss.NewStyle().Foreground(t.Error).Bold(true),
		Dimmed: lipgloss.NewStyle().Foreground(t.Dimmed),
	}
}
