package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ldgnu/minitone/internal/app"
	"github.com/ldgnu/minitone/internal/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("minitone %s\n", app.Version)
			return
		case "--import-playlist":
			// minitone --import-playlist <youtube-playlist-URL> [name]
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: minitone --import-playlist <youtube-playlist-URL> [name]")
				os.Exit(2)
			}
			app.ImportPlaylist(os.Args[2], joinArgs(os.Args[3:]))
			return
		case "--screenshot":
			// --screenshot <scenario> [w] [h] [theme]
			scenario := "welcome"
			w, h := 100, 28
			theme := "tokyonight"
			if len(os.Args) > 2 {
				scenario = os.Args[2]
			}
			if !slices.Contains(ui.Scenarios, scenario) {
				fmt.Fprintf(os.Stderr, "minitone: unknown scenario %q\nknown: %s\n",
					scenario, strings.Join(ui.Scenarios, ", "))
				os.Exit(2)
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
		case "--list-screenshots":
			for _, s := range ui.Scenarios {
				fmt.Println(s)
			}
			return
		case "-h", "--help", "help":
			fmt.Print(`minitone — TUI music player

Usage:
  minitone              start the player
  minitone --version    print version
  minitone --import-playlist <youtube-URL> [name]
                        import a public YouTube playlist into Playlists
  minitone --help       this help

Sources: YouTube, Radio Browser, Navidrome, local library, favorites, playlists.

Config: ~/.config/minitone/config.json
          taste_genres: ["hardcore","uptempo"]  (your Apple Music reference)
          taste_artists: ["Angerfist"]
Data:   ~/.config/minitone/favorites.json
        ~/.config/minitone/history.json
        ~/.config/minitone/playlists.json

Two focus modes:
  search  type to search · tab/enter browse results · esc clear
  browse  j k move · enter play · a queue · A queue all · f favorite
          i details · d remove · c clear · J K move in queue
          n/p next/prev · s stop · space pause · esc back to search

Anywhere:
  ctrl+j queue · ctrl+f favorites · ctrl+h history · ctrl+l library
  ctrl+p playlists · ctrl+s rescan library · ctrl+t theme · ctrl+v video · ctrl+r repeat
  ctrl+u shuffle · ctrl+/ help · q quit · ctrl+c always quits

Search prefixes:  /search /youtube /radio /navidrome /local /fav  + term

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

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}
