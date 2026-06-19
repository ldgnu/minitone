package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/subsonic"
	"github.com/ldgnu/minitone/internal/tui"
)

func main() {
	server := os.Getenv("NAVIDROME_URL")
	user := os.Getenv("NAVIDROME_USER")
	pass := os.Getenv("NAVIDROME_PASS")

	if server == "" || user == "" || pass == "" {
		fmt.Fprintln(os.Stderr, "Set NAVIDROME_URL, NAVIDROME_USER and NAVIDROME_PASS")
		os.Exit(1)
	}

	client := subsonic.NewClient(server, user, pass)
	player := subsonic.NewPlayer(client)

	p := tea.NewProgram(
		tui.NewModel(client, player),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	player.Close()
}
