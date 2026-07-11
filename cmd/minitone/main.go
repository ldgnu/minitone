package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ldgnu/minitone/internal/app"
	"github.com/ldgnu/minitone/internal/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("minitone %s\n", app.Version)
			return
		case "--screenshot":
			// --screenshot <scenario> [w] [h] [theme]
			scenario := "welcome"
			w, h := 100, 30
			theme := "tokyonight"
			if len(os.Args) > 2 {
				scenario = os.Args[2]
			}
			if len(os.Args) > 3 {
				if v, err := strconv.Atoi(os.Args[3]); err == nil {
					w = v
				}
			}
			if len(os.Args) > 4 {
				if v, err := strconv.Atoi(os.Args[4]); err == nil {
					h = v
				}
			}
			if len(os.Args) > 5 {
				theme = os.Args[5]
			}
			fmt.Print(ui.Screenshot(scenario, w, h, theme))
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
  type to search · enter play · f favorite · ctrl+h history
  ctrl+f favorites · ctrl+j queue · ctrl+v video · space pause
  ctrl+t theme · ctrl+s stop · ? help · q quit
  esc back · ctrl+c quit

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
