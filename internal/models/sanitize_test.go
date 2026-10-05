package models

import (
	"strings"
	"testing"
)

// Regression: remote metadata reaches the terminal. Radio Browser in particular
// is a public, community-editable database, so a station name is attacker-
// controlled. Any OSC escape (title rewrite, clipboard write via OSC 52, DCS)
// must never reach the screen.
func TestSanitizeStripsTerminalEscapes(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		leaks []string
	}{
		{"OSC 2 sets the terminal title", "\x1b]2;PWNED\x07title", []string{"\x1b]2;", "PWNED"}},
		{"OSC 52 writes the clipboard", "\x1b]52;c;YWJjZGVmZ2hpamtsbW5vcA==\x07", []string{"\x1b]52;", "YWJj"}},
		{"OSC terminated by ST", "a\x1b]0;evil\x1b\\b", []string{"\x1b]", "evil"}},
		{"CSI cursor movement", "\x1b[2J\x1b[Hclean", []string{"\x1b["}},
		{"DCS", "\x1bPq#0;2;0;0;0\x1b\\x", []string{"\x1bP"}},
		{"stray BEL", "title\x07", []string{"\x07"}},
		{"newline injection", "line1\nline2", []string{"\n"}},
		{"carriage return", "overwrite\rX", []string{"\r"}},
		{"NUL and other C0", "a\x00b\x1fc", []string{"\x00", "\x1f"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Sanitize(c.in)
			for _, l := range c.leaks {
				if strings.Contains(got, l) {
					t.Fatalf("escape survived: input %q -> output %q (contains %q)", c.in, got, l)
				}
			}
			if strings.ContainsAny(got, "\x1b\x07\x00") {
				t.Fatalf("control character survived: %q", got)
			}
		})
	}
}

func TestSanitizeKeepsLegitimateText(t *testing.T) {
	// Sanitising must not mangle ordinary metadata, including non-ASCII.
	for _, in := range []string{
		"lofi hip hop radio 📚 - beats to relax/study to",
		"Björk — Jóga (remaster)",
		"Café Tacvba: Éramos Cuatro",
		"SomaFM Drone Zone (128k MP3)",
		"Спокойная ночь",
		"안녕하세요",
	} {
		if got := Sanitize(in); got != in {
			t.Errorf("Sanitize(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestSanitizeBoundsLength(t *testing.T) {
	huge := strings.Repeat("A", 5000)
	got := Sanitize(huge)
	if len([]rune(got)) > maxDisplayLen+1 {
		t.Fatalf("length %d, want <= %d", len([]rune(got)), maxDisplayLen+1)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatal("a truncated string should be marked")
	}
}

func TestSanitizeEmptyAndPlain(t *testing.T) {
	if got := Sanitize(""); got != "" {
		t.Fatalf("empty -> %q", got)
	}
	if got := Sanitize("plain title"); got != "plain title" {
		t.Fatalf("plain -> %q", got)
	}
}

func TestSongSanitizedCleansEveryField(t *testing.T) {
	payload := "\x1b]52;c;YWJj\x07"
	s := Song{
		Title:    "t" + payload,
		Artist:   "a" + payload,
		Album:    "al" + payload,
		Genre:    "g" + payload,
		URL:      "u" + payload,
		FilePath: "f" + payload,
	}.Sanitized()

	for name, v := range map[string]string{
		"title": s.Title, "artist": s.Artist, "album": s.Album,
		"genre": s.Genre, "url": s.URL, "file": s.FilePath,
	} {
		if strings.ContainsAny(v, "\x1b\x07") {
			t.Errorf("%s still carries an escape: %q", name, v)
		}
	}
}

func TestKeyUnaffectedBySanitize(t *testing.T) {
	// Sanitising must not collapse two different songs into one.
	a := Song{ID: "yt:a", Title: "A"}.Sanitized()
	b := Song{ID: "yt:b", Title: "B"}.Sanitized()
	if a.Key() == b.Key() {
		t.Fatal("distinct songs share a key after sanitising")
	}
}
