package ui

import (
	"strings"
	"testing"

	"github.com/ldgnu/minitone/internal/models"
)

// End-to-end: a hostile remote title must not reach the terminal.
//
// This is the view-level regression for the OSC-injection finding: the source
// clients sanitise on ingest, and this asserts nothing re-introduces an escape
// on the way to the screen.
func TestRemoteDataNeverReachesTerminalRaw(t *testing.T) {
	payloads := map[string]string{
		"osc2 title":      "\x1b]2;PWNED\x07",
		"osc52 clipboard": "\x1b]52;c;YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=\x07",
		"osc st":          "\x1b]0;evil\x1b\\",
		"csi clear":       "\x1b[2J\x1b[H",
		"dcs":             "\x1bPq#0;2;0;0;0\x1b\\",
		"stray bel":       "\a",
	}

	for name, payload := range payloads {
		t.Run(name, func(t *testing.T) {
			dirty := models.Song{
				Source: models.SourceRadio,
				Title:  "song" + payload + "tail",
				Artist: "artist" + payload,
				Album:  "album" + payload,
				Genre:  "genre" + payload,
			}
			// What the source layer would hand to the UI.
			clean := dirty.Sanitized()

			m := New(Deps{Theme: "terminal"})
			m.width, m.height = 100, 30
			m = results(m, "x", clean)
			m.focus = FocusList

			out := m.View()
			for _, bad := range []string{"\x1b]", "\x1bP", "\x1b[2J", "\a"} {
				if strings.Contains(out, bad) {
					t.Fatalf("escape %q survived into the view:\n%q", bad, out)
				}
			}
			// The visible text must survive.
			if !strings.Contains(out, "tail") {
				t.Fatalf("sanitising destroyed the visible text:\n%q", out)
			}
		})
	}
}

// A poisoned favourites.json must be cleaned on load, before it is rendered.
func TestPoisonedLocalStateIsCleanedOnLoad(t *testing.T) {
	song := models.Song{
		ID:     "radio:x",
		Source: models.SourceRadio,
		Title:  "nice\x1b]52;c;YWJjZGVmZ2hpamtsbW5vcA==\x07name",
		URL:    "http://x/\x1b]0;evil\x1b\\",
	}
	clean := song.Sanitized()

	if strings.ContainsAny(clean.Title, "\x1b\a") {
		t.Fatalf("title still carries an escape: %q", clean.Title)
	}
	if !strings.Contains(clean.Title, "name") {
		t.Fatalf("title text lost: %q", clean.Title)
	}
	if strings.ContainsAny(clean.URL, "\x1b") {
		t.Fatalf("URL still carries an escape: %q", clean.URL)
	}
}
