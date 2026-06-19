// minitone - TUI for Apple Music via Cider
// by ldgnu <ldgnu@users.noreply.github.com>


package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/cider"
	"github.com/ldgnu/minitone/internal/tui"
)

func main() {
	client := cider.NewFromEnv()

	p := tea.NewProgram(
		tui.NewModel(client),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
