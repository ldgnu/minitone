package main

import (
	"fmt"
	"os"

	"github.com/ldgnu/minitone/internal/app"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("minitone %s\n", app.Version)
			return
		case "-h", "--help", "help":
			fmt.Print(`minitone — TUI music player

Usage:
  minitone              start the player
  minitone --version    print version
  minitone --help       this help

Sources: YouTube, Radio Browser, Navidrome, local library, favorites.

Config: ~/.config/minitone/config.json
Data:   ~/.config/minitone/favorites.json
        ~/.config/minitone/history.json

Keys (search empty):
  type to search · enter play · f favorite · h history
  ctrl+f favorites · ctrl+j queue · space pause · q quit
  ? help

Requires: mpv, yt-dlp (for YouTube)
`)
			return
		}
	}

	a := app.New()
	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "minitone: %v\n", err)
		os.Exit(1)
	}
}
