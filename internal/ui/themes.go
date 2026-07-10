package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name       string
	Primary    lipgloss.Color
	Dimmed     lipgloss.Color
	Highlight  lipgloss.Color
	Active     lipgloss.Color
	Error      lipgloss.Color
	Progress   lipgloss.Color
	ProgressBg lipgloss.Color
}

var themes = []Theme{
	{Name: "terminal", Primary: lipgloss.Color(""), Dimmed: lipgloss.Color(""), Highlight: lipgloss.Color(""), Active: lipgloss.Color(""), Error: lipgloss.Color("1"), Progress: lipgloss.Color("2"), ProgressBg: lipgloss.Color("")},
	{Name: "fallout", Primary: lipgloss.Color("#00FF41"), Dimmed: lipgloss.Color("#005F00"), Highlight: lipgloss.Color("#00FF41"), Active: lipgloss.Color("#00FF41"), Error: lipgloss.Color("#FF0000"), Progress: lipgloss.Color("#00FF41"), ProgressBg: lipgloss.Color("#003300")},
	{Name: "tokyonight", Primary: lipgloss.Color("#7AA2F7"), Dimmed: lipgloss.Color("#565F89"), Highlight: lipgloss.Color("#7DCFFF"), Active: lipgloss.Color("#A9B1D6"), Error: lipgloss.Color("#DB4B4B"), Progress: lipgloss.Color("#7AA2F7"), ProgressBg: lipgloss.Color("#3B4261")},
	{Name: "everforest", Primary: lipgloss.Color("#A7C080"), Dimmed: lipgloss.Color("#859289"), Highlight: lipgloss.Color("#D3C6AA"), Active: lipgloss.Color("#E5E9C5"), Error: lipgloss.Color("#E67E80"), Progress: lipgloss.Color("#A7C080"), ProgressBg: lipgloss.Color("#4A5555")},
	{Name: "catppuccin", Primary: lipgloss.Color("#F5C2E7"), Dimmed: lipgloss.Color("#6C7086"), Highlight: lipgloss.Color("#CBA6F7"), Active: lipgloss.Color("#CDD6F4"), Error: lipgloss.Color("#F38BA8"), Progress: lipgloss.Color("#F5C2E7"), ProgressBg: lipgloss.Color("#45475A")},
	{Name: "gruvbox", Primary: lipgloss.Color("#FE8019"), Dimmed: lipgloss.Color("#7C6F64"), Highlight: lipgloss.Color("#FABD2F"), Active: lipgloss.Color("#EBDBB2"), Error: lipgloss.Color("#FB4934"), Progress: lipgloss.Color("#FE8019"), ProgressBg: lipgloss.Color("#504945")},
	{Name: "nord", Primary: lipgloss.Color("#88C0D0"), Dimmed: lipgloss.Color("#4C566A"), Highlight: lipgloss.Color("#8FBCBB"), Active: lipgloss.Color("#D8DEE9"), Error: lipgloss.Color("#BF616A"), Progress: lipgloss.Color("#88C0D0"), ProgressBg: lipgloss.Color("#434C5E")},
	{Name: "kanagawa", Primary: lipgloss.Color("#E6C384"), Dimmed: lipgloss.Color("#727169"), Highlight: lipgloss.Color("#DCD7BA"), Active: lipgloss.Color("#DCD7BA"), Error: lipgloss.Color("#C34043"), Progress: lipgloss.Color("#E6C384"), ProgressBg: lipgloss.Color("#363646")},
	{Name: "dracula", Primary: lipgloss.Color("#BD93F9"), Dimmed: lipgloss.Color("#6272A4"), Highlight: lipgloss.Color("#F1FA8C"), Active: lipgloss.Color("#F8F8F2"), Error: lipgloss.Color("#FF5555"), Progress: lipgloss.Color("#50FA7B"), ProgressBg: lipgloss.Color("#44475A")},
	{Name: "monochrome", Primary: lipgloss.Color("#00FF41"), Dimmed: lipgloss.Color("#005F00"), Highlight: lipgloss.Color("#00FF41"), Active: lipgloss.Color("#00FF41"), Error: lipgloss.Color("#FF0000"), Progress: lipgloss.Color("#00FF41"), ProgressBg: lipgloss.Color("#003300")},
	{Name: "amber", Primary: lipgloss.Color("#FFB000"), Dimmed: lipgloss.Color("#6A4E00"), Highlight: lipgloss.Color("#FFD700"), Active: lipgloss.Color("#FFC000"), Error: lipgloss.Color("#FF4500"), Progress: lipgloss.Color("#FFB000"), ProgressBg: lipgloss.Color("#3A2A00")},
}

// ThemeIndex returns the theme index for a name (case-insensitive). Defaults to terminal (system).
func ThemeIndex(name string) int {
	if name == "" {
		return 0 // terminal (system)
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	for i, t := range themes {
		if strings.ToLower(t.Name) == lower {
			return i
		}
	}
	return 0
}

