package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/search"
)

// The acceptance criterion: the app must work in narrow terminals, so every
// rendered frame has to fit the reported size exactly.
func TestViewNeverOverflows(t *testing.T) {
	sizes := []struct{ w, h int }{
		{40, 12}, {50, 16}, {64, 16}, {76, 18}, {80, 24}, {100, 30}, {120, 40}, {200, 50},
	}
	for _, sc := range Scenarios {
		for _, size := range sizes {
			out := Screenshot(sc, size.w, size.h, "terminal")
			lines := strings.Split(out, "\n")
			for i, line := range lines {
				if got := lipgloss.Width(line); got > size.w {
					t.Fatalf("%s %dx%d: line %d is %d cells wide:\n%q",
						sc, size.w, size.h, i+1, got, line)
				}
			}
			if len(lines) > size.h {
				t.Fatalf("%s %dx%d: rendered %d lines", sc, size.w, size.h, len(lines))
			}
		}
	}
}

// The player bar must be visible in every state — that is the whole point of
// making it persistent.
func TestPlayerBarAlwaysPresent(t *testing.T) {
	for _, sc := range []string{"welcome", "search", "queue", "favorites", "history", "library", "help", "details"} {
		out := Screenshot(sc, 100, 26, "terminal")
		if !strings.Contains(out, "nothing playing") && !strings.Contains(out, "▶") {
			t.Fatalf("%s: no player state line in:\n%s", sc, out)
		}
	}
}

func TestPlayerBarShowsEverything(t *testing.T) {
	out := Screenshot("playing", 100, 26, "terminal")
	for _, want := range []string{"Midnight City", "M83", "1:42", "4:04", "%", "queue"} {
		if !strings.Contains(out, want) {
			t.Fatalf("player bar missing %q:\n%s", want, out)
		}
	}
}

func TestPlayerBarCompactKeepsEssentials(t *testing.T) {
	out := Screenshot("playing", 60, 16, "terminal")
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Fatal("no output")
	}
	// Compact mode drops the third line but keeps song, position and volume.
	if !strings.Contains(out, "Midnight City") {
		t.Fatalf("compact lost the title:\n%s", out)
	}
	if !strings.Contains(out, "%") {
		t.Fatalf("compact lost the volume:\n%s", out)
	}
	if strings.Contains(out, "youtube\n") {
		t.Fatalf("compact should drop the source column:\n%s", out)
	}
}

func TestCompactThreshold(t *testing.T) {
	m := newTestModel(t, 100, 30)
	if m.compact() {
		t.Fatal("100x30 should not be compact")
	}
	m.width, m.height = 70, 30
	if !m.compact() {
		t.Fatal("width < 76 must be compact")
	}
	m.width, m.height = 100, 12
	if !m.compact() {
		t.Fatal("height < 18 must be compact")
	}
	m.width, m.height = 80, 24
	if m.compact() {
		t.Fatal("80x24 is a normal terminal")
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

// newTestPlayer returns a player that is never started, so a unit test can
// never spawn an mpv process.
func newTestPlayer() *player.Player { return player.New() }

func newTestModel(t *testing.T, w, h int) Model {
	t.Helper()
	m := New(Deps{
		Player:   newTestPlayer(),
		Theme:    "terminal",
		Debounce: 10,
	})
	m.width, m.height = w, h
	return m
}

func withResults(m Model, query string, items ...models.Song) Model {
	m.search.query = query
	m.search.term = query
	m.search.bySource = map[string][]models.Song{search.SourceYouTube: items}
	m.clampCursors()
	return m
}

var _ = strings.TrimSpace

// A highlighted track deep in a long group must stay visible: the list
// scrolls instead of pushing the selection off screen.
func TestResultsScrollToKeepSelectionVisible(t *testing.T) {
	var items []models.Song
	for i := 0; i < 40; i++ {
		items = append(items, models.Song{
			Source: models.SourceYouTube,
			Title:  "track " + itoa(i),
		})
	}

	for cursor := 0; cursor < len(items); cursor++ {
		m := results(newTestModel(t, 100, 18), "x", items...)
		m.search.cursor = cursor

		out := ScreenshotWith(m, 100, 18, "terminal")
		want := "track " + itoa(cursor)
		if !strings.Contains(out, want) {
			t.Fatalf("cursor %d: the highlighted track %q is not visible:\n%s", cursor, want, out)
		}
	}
}

// ScreenshotWith renders an existing model (used for cursor-dependent checks).
func ScreenshotWith(m Model, w, h int, theme string) string {
	m.width, m.height = w, h
	if m.styles.Header.GetForeground() == nil && theme != "" {
		m.themeIdx = ThemeIndex(theme)
		m.styles = NewStyles(themes[m.themeIdx])
	}
	m.clampCursors()
	return m.View()
}
