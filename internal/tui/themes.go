// minitone - TUI pa' controlar Apple Music desde Cider
// by ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

package tui

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Dimmed    lipgloss.Color
	Highlight lipgloss.Color
	Active    lipgloss.Color
	Error     lipgloss.Color
	Progress  lipgloss.Color
	ProgressBg lipgloss.Color
}

var themes = []Theme{
	{
		Name:       "system",
		Primary:    lipgloss.Color("#FA5860"),
		Dimmed:     lipgloss.Color("#505050"),
		Highlight:  lipgloss.Color("#FA5860"),
		Active:     lipgloss.Color("#FFF"),
		Error:      lipgloss.Color("#FF0000"),
		Progress:   lipgloss.Color("#FA5860"),
		ProgressBg: lipgloss.Color("#333"),
	},
	{
		Name:       "tokyonight",
		Primary:    lipgloss.Color("#7AA2F7"),
		Dimmed:     lipgloss.Color("#565F89"),
		Highlight:  lipgloss.Color("#7DCFFF"),
		Active:     lipgloss.Color("#A9B1D6"),
		Error:      lipgloss.Color("#DB4B4B"),
		Progress:   lipgloss.Color("#7AA2F7"),
		ProgressBg: lipgloss.Color("#3B4261"),
	},
	{
		Name:       "everforest",
		Primary:    lipgloss.Color("#A7C080"),
		Dimmed:     lipgloss.Color("#859289"),
		Highlight:  lipgloss.Color("#D3C6AA"),
		Active:     lipgloss.Color("#E5E9C5"),
		Error:      lipgloss.Color("#E67E80"),
		Progress:   lipgloss.Color("#A7C080"),
		ProgressBg: lipgloss.Color("#4A5555"),
	},
	{
		Name:       "ayu",
		Primary:    lipgloss.Color("#FF9940"),
		Dimmed:     lipgloss.Color("#5C6773"),
		Highlight:  lipgloss.Color("#E6B450"),
		Active:     lipgloss.Color("#B3B1AD"),
		Error:      lipgloss.Color("#F07171"),
		Progress:   lipgloss.Color("#FF9940"),
		ProgressBg: lipgloss.Color("#3D424D"),
	},
	{
		Name:       "catppuccin",
		Primary:    lipgloss.Color("#F5C2E7"),
		Dimmed:     lipgloss.Color("#6C7086"),
		Highlight:  lipgloss.Color("#CBA6F7"),
		Active:     lipgloss.Color("#CDD6F4"),
		Error:      lipgloss.Color("#F38BA8"),
		Progress:   lipgloss.Color("#F5C2E7"),
		ProgressBg: lipgloss.Color("#45475A"),
	},
	{
		Name:       "catppuccin-macchiato",
		Primary:    lipgloss.Color("#F4DBD6"),
		Dimmed:     lipgloss.Color("#6E738D"),
		Highlight:  lipgloss.Color("#C6A0F6"),
		Active:     lipgloss.Color("#CAD3F5"),
		Error:      lipgloss.Color("#ED8796"),
		Progress:   lipgloss.Color("#F4DBD6"),
		ProgressBg: lipgloss.Color("#494D64"),
	},
	{
		Name:       "gruvbox",
		Primary:    lipgloss.Color("#FE8019"),
		Dimmed:     lipgloss.Color("#7C6F64"),
		Highlight:  lipgloss.Color("#FABD2F"),
		Active:     lipgloss.Color("#EBDBB2"),
		Error:      lipgloss.Color("#FB4934"),
		Progress:   lipgloss.Color("#FE8019"),
		ProgressBg: lipgloss.Color("#504945"),
	},
	{
		Name:       "kanagawa",
		Primary:    lipgloss.Color("#E6C384"),
		Dimmed:     lipgloss.Color("#727169"),
		Highlight:  lipgloss.Color("#DCD7BA"),
		Active:     lipgloss.Color("#DCD7BA"),
		Error:      lipgloss.Color("#C34043"),
		Progress:   lipgloss.Color("#E6C384"),
		ProgressBg: lipgloss.Color("#363646"),
	},
	{
		Name:       "nord",
		Primary:    lipgloss.Color("#88C0D0"),
		Dimmed:     lipgloss.Color("#4C566A"),
		Highlight:  lipgloss.Color("#8FBCBB"),
		Active:     lipgloss.Color("#D8DEE9"),
		Error:      lipgloss.Color("#BF616A"),
		Progress:   lipgloss.Color("#88C0D0"),
		ProgressBg: lipgloss.Color("#434C5E"),
	},
	{
		Name:       "matrix",
		Primary:    lipgloss.Color("#00FF41"),
		Dimmed:     lipgloss.Color("#005F00"),
		Highlight:  lipgloss.Color("#00FF41"),
		Active:     lipgloss.Color("#00FF41"),
		Error:      lipgloss.Color("#FF0000"),
		Progress:   lipgloss.Color("#00FF41"),
		ProgressBg: lipgloss.Color("#003300"),
	},
	{
		Name:       "one-dark",
		Primary:    lipgloss.Color("#61AFEF"),
		Dimmed:     lipgloss.Color("#5C6370"),
		Highlight:  lipgloss.Color("#E5C07B"),
		Active:     lipgloss.Color("#ABB2BF"),
		Error:      lipgloss.Color("#E06C75"),
		Progress:   lipgloss.Color("#61AFEF"),
		ProgressBg: lipgloss.Color("#3E4452"),
	},
}

func themeByName(name string) (Theme, int) {
	for i, t := range themes {
		if t.Name == name {
			return t, i
		}
	}
	return themes[0], 0
}
