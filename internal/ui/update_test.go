package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/search"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	}
	if s == "shift+tab" {
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	}
	if strings.HasPrefix(s, "ctrl+") && len(s) == 6 {
		if t, ok := ctrlKeys[s[5:]]; ok {
			return tea.KeyMsg{Type: t}
		}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// ctrlKeys maps a ctrl+<letter> name to Bubble Tea's key type, so the tests
// exercise exactly what a real terminal produces.
var ctrlKeys = map[string]tea.KeyType{
	"a": tea.KeyCtrlA, "c": tea.KeyCtrlC, "f": tea.KeyCtrlF, "h": tea.KeyCtrlH,
	"j": tea.KeyCtrlJ, "k": tea.KeyCtrlK, "l": tea.KeyCtrlL, "n": tea.KeyCtrlN,
	"p": tea.KeyCtrlP, "r": tea.KeyCtrlR, "s": tea.KeyCtrlS, "t": tea.KeyCtrlT,
	"u": tea.KeyCtrlU, "v": tea.KeyCtrlV, "d": tea.KeyCtrlD, "b": tea.KeyCtrlB,
}

// press sends a key and returns the updated model.
func press(t *testing.T, m Model, k string) Model {
	t.Helper()
	next, _ := m.Update(key(k))
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	return got
}

func results(m Model, query string, items ...models.Song) Model {
	m.search.query = query
	m.search.term = query
	m.search.bySource = map[string][]models.Song{search.SourceYouTube: items}
	m.search.pending = map[string]bool{}
	m.search.failures = map[string]error{}
	m.search.running = false
	m.focus = FocusList
	m.clampCursors()
	return m
}

func song(title string) models.Song {
	return models.Song{ID: "yt:" + title, Source: models.SourceYouTube, Title: title}
}

// ── the acceptance criterion: q must always close the app ───────────────────

func TestQAlwaysQuits(t *testing.T) {
	cases := []struct {
		name  string
		setup func(Model) Model
		key   string
	}{
		{"empty search", func(m Model) Model { return m }, "q"},
		{"with results", func(m Model) Model {
			return results(m, "lofi", song("a"), song("b"), song("c"))
		}, "q"},
		{"queue panel", func(m Model) Model {
			m = results(m, "lofi", song("a"))
			m.queue.Add(song("a"))
			m.panel = PanelQueue
			return m
		}, "q"},
		{"history panel", func(m Model) Model {
			m.panel = PanelHistory
			return m
		}, "q"},
		{"library panel", func(m Model) Model {
			m.panel = PanelLibrary
			return m
		}, "q"},
		{"help panel", func(m Model) Model {
			m.panel = PanelHelp
			return m
		}, "q"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := c.setup(newTestModel(t, 100, 30))
			_, cmd := m.Update(key(c.key))
			if cmd == nil {
				t.Fatalf("%q did not quit", c.key)
			}
			if msg := cmd(); msg == nil {
				t.Fatal("expected tea.QuitMsg")
			} else if _, ok := msg.(tea.QuitMsg); !ok {
				t.Fatalf("expected QuitMsg, got %T", msg)
			}
		})
	}
}

// Regression: typing "q" into a search term must type a "q", not quit.
func TestQTypesInSearchWhenQueryIsPresent(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.query = "lofi"
	m.search.term = "lofi"

	next, cmd := m.Update(key("q"))
	got := next.(Model)
	if cmd != nil {
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Fatal("typing q inside a search query must not quit")
		}
	}
	if !strings.HasSuffix(got.search.query, "q") {
		t.Fatalf("query = %q, want it to end in q", got.search.query)
	}
}

func TestCtrlCAlwaysQuits(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.query = "lofi"
	_, cmd := m.Update(key("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c must quit even with a query")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected QuitMsg")
	}
}

// ── letters type in search focus ────────────────────────────────────────────

func TestEveryPrintableGoesToQueryInSearchFocus(t *testing.T) {
	// Every letter, including the ones used as actions in list focus.
	// "q" is excluded only when the box is empty, where it means "quit".
	// Excluded: q (quits on an empty box) and ? (opens help) — both have their
	// own tests below.
	for _, r := range "abcdefghijklmnpstuvwxyzABCPSTUVWXYZ0123456789 ./" {
		m := newTestModel(t, 100, 30)
		got := press(t, m, string(r))
		if got.search.query != string(r) {
			t.Fatalf("key %q produced query %q, want %q", r, got.search.query, r)
		}
	}

	// Once a query exists, "q" and "-" are ordinary characters again.
	m := newTestModel(t, 100, 30)
	m = press(t, m, "l")
	got := press(t, m, "q")
	if got.search.query != "lq" {
		t.Fatalf("q inside a query should type, got %q", got.search.query)
	}

	got = press(t, got, "-")
	if got.search.query != "lq-" {
		t.Fatalf("dash must be typeable inside a query, got %q", got.search.query)
	}
}

// "?" is the portable help key (ctrl+/ is not reported by every terminal), so
// it opens help instead of ending up in the query.
func TestQuestionMarkOpensHelp(t *testing.T) {
	for _, focus := range []Focus{FocusSearch, FocusList} {
		m := newTestModel(t, 100, 30)
		m.focus = focus
		m = press(t, m, "?")
		if m.panel != PanelHelp {
			t.Fatalf("focus %v: ? did not open help (panel=%v)", focus, m.panel)
		}
		if m.search.query != "" {
			t.Fatalf("focus %v: ? was typed into the query: %q", focus, m.search.query)
		}
		// esc closes it and restores the focus.
		m = press(t, m, "esc")
		if m.panel != PanelNone {
			t.Fatalf("esc did not close help")
		}
		if m.focus != focus {
			t.Fatalf("focus restored to %v, want %v", m.focus, focus)
		}
	}
}

func TestHelpToggles(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m = press(t, m, "?")
	if m.panel != PanelHelp {
		t.Fatal("? did not open help")
	}
	m = press(t, m, "?")
	if m.panel != PanelNone {
		t.Fatal("pressing ? again should close help")
	}
}

func TestSpaceIsTypedNotPlayed(t *testing.T) {
	// Otherwise "lofi girl" is impossible to search for.
	m := newTestModel(t, 100, 30)
	for _, r := range "lofi girl" {
		m = press(t, m, string(r))
	}
	if m.search.query != "lofi girl" {
		t.Fatalf("query %q", m.search.query)
	}
}

// Regression: j/k/t/f used to be swallowed by the search box in *both* modes,
// and f used to be documented as "favorite" while doing nothing.
func TestListFocusActions(t *testing.T) {
	items := []models.Song{song("one"), song("two"), song("three")}

	t.Run("j/k navigate", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m.search.cursor = 0
		m = press(t, m, "j")
		if m.search.cursor != 1 {
			t.Fatalf("j: cursor %d", m.search.cursor)
		}
		m = press(t, m, "j")
		if m.search.cursor != 2 {
			t.Fatalf("jj: cursor %d", m.search.cursor)
		}
		m = press(t, m, "k")
		if m.search.cursor != 1 {
			t.Fatalf("k: cursor %d", m.search.cursor)
		}
	})

	t.Run("a enqueues the selection", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m.search.cursor = 1
		m = press(t, m, "a")
		if m.queue.Len() != 1 {
			t.Fatalf("queue len %d", m.queue.Len())
		}
		if got := m.queue.Items()[0].Song.Title; got != "two" {
			t.Fatalf("queued %q", got)
		}
	})

	t.Run("A enqueues everything", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m = press(t, m, "A")
		if m.queue.Len() != 3 {
			t.Fatalf("queue len %d", m.queue.Len())
		}
		// A second press must not duplicate.
		m = press(t, m, "A")
		if m.queue.Len() != 3 {
			t.Fatalf("duplicates added: len %d", m.queue.Len())
		}
	})

	t.Run("f toggles favorite", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m = press(t, m, "f")
		if m.favs.Len() != 1 {
			t.Fatalf("favs %d", m.favs.Len())
		}
		m = press(t, m, "f")
		if m.favs.Len() != 0 {
			t.Fatalf("toggle failed: favs %d", m.favs.Len())
		}
	})

	t.Run("i opens details", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m = press(t, m, "i")
		if m.panel != PanelDetails {
			t.Fatalf("panel %v", m.panel)
		}
		if m.details.Title != "one" {
			t.Fatalf("details %q", m.details.Title)
		}
	})

	t.Run("n and p step the queue", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m.queue.Add(items[0])
		m.queue.Add(items[1])
		m.queue.Add(items[2])
		m.queue.SetCursor(0)

		if cmd := m.playNext(); cmd == nil {
			t.Fatal("playNext returned no command")
		}
		if cur := m.queue.Cursor(); cur != 1 {
			t.Fatalf("cursor %d", cur)
		}
		m.playPrev()
		if cur := m.queue.Cursor(); cur != 0 {
			t.Fatalf("prev cursor %d", cur)
		}
	})

	t.Run("c clears the queue", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "x", items...)
		m.queue.Add(items[0])
		m = press(t, m, "c")
		if m.queue.Len() != 0 {
			t.Fatalf("queue len %d", m.queue.Len())
		}
	})
}

func TestJKMovesTheQueueItem(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"))
	m.queue.Add(song("a"))
	m.queue.Add(song("b"))
	m.queue.Add(song("c"))
	m.queue.SetCursor(1)

	before := m.queue.Cursor()
	m = press(t, m, "J")
	if m.queue.Cursor() != before-1 {
		t.Fatalf("J should move the playing item up: %d -> %d", before, m.queue.Cursor())
	}
	m = press(t, m, "K")
	if m.queue.Cursor() != before {
		t.Fatalf("K should move it back: %d", m.queue.Cursor())
	}
}

// Typing a letter that is *also* an action must fall back to search mode,
// so search terms containing "a", "s" or "d" still work.
func TestTypingFallsBackToSearch(t *testing.T) {
	// A printable key that is not a shortcut returns to typing, so a search
	// term can contain characters that are also actions in list focus.
	m := results(newTestModel(t, 100, 30), "lofi", song("a"))
	m.queue.Add(song("a"))

	m = press(t, m, "z")
	if m.focus != FocusSearch {
		t.Fatalf("focus %v", m.focus)
	}
	if !strings.HasSuffix(m.search.query, "z") {
		t.Fatalf("query %q", m.search.query)
	}

	// A key that *is* a shortcut acts instead of typing.
	m2 := results(newTestModel(t, 100, 30), "lofi", song("a"))
	m2 = press(t, m2, "s")
	if m2.focus != FocusList {
		t.Fatal("s is the stop shortcut in list focus")
	}
	if !strings.Contains(m2.notice.text, "stopped") {
		t.Fatalf("notice %q", m2.notice.text)
	}
}

// ── esc cancels, in order ───────────────────────────────────────────────────

func TestEscapeOrder(t *testing.T) {
	t.Run("dismisses the error first", func(t *testing.T) {
		m := newTestModel(t, 100, 30)
		m.focus = FocusList
		m.notice = errorNotice("boom", "why", "[esc] dismiss")
		m = press(t, m, "esc")
		if m.notice.active() {
			t.Fatal("esc should dismiss the error")
		}
		if m.focus != FocusList {
			t.Fatal("esc must not change focus while an error is shown")
		}
	})

	t.Run("closes the panel", func(t *testing.T) {
		m := newTestModel(t, 100, 30)
		m.queue.Add(song("a"))
		m.panel = PanelQueue
		m = press(t, m, "esc")
		if m.panel != PanelNone {
			t.Fatalf("panel %v", m.panel)
		}
	})

	t.Run("leaves list focus for the search box", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "lofi", song("a"))
		m = press(t, m, "esc")
		if m.focus != FocusSearch {
			t.Fatalf("focus %v", m.focus)
		}
		if m.search.query != "lofi" {
			t.Fatalf("esc should not clear the query yet: %q", m.search.query)
		}
	})

	t.Run("clears the query last", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "lofi", song("a"))
		m = press(t, m, "esc") // to search
		m = press(t, m, "esc") // clear
		if m.search.query != "" {
			t.Fatalf("query %q", m.search.query)
		}
	})
}

// ── panels ──────────────────────────────────────────────────────────────────

func TestQueuePanelKeys(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"), song("b"), song("c"))
	for _, s := range []string{"a", "b", "c"} {
		m.queue.Add(song(s))
	}
	m.queue.SetCursor(1)
	m.panel = PanelQueue
	m.panelCursor = 0

	m = press(t, m, "j")
	if m.panelCursor != 1 {
		t.Fatalf("panelCursor %d", m.panelCursor)
	}

	m = press(t, m, "J")
	if m.queue.Cursor() != 0 {
		t.Fatalf("J in the queue should move the item: cursor %d", m.queue.Cursor())
	}

	m = press(t, m, "d")
	if m.queue.Len() != 2 {
		t.Fatalf("d should remove: len %d", m.queue.Len())
	}

	m = press(t, m, "c")
	if m.queue.Len() != 0 {
		t.Fatalf("c should clear: len %d", m.queue.Len())
	}
	if m.panel != PanelNone {
		t.Fatal("an empty queue should close the panel")
	}
}

func TestFavoritesPanelKeys(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"), song("b"))
	m.favs.Add(song("a"))
	m.favs.Add(song("b"))
	m.panel = PanelFavorites

	m = press(t, m, "a")
	if m.queue.Len() != 1 {
		t.Fatalf("a should enqueue the selection: %d", m.queue.Len())
	}

	m = press(t, m, "A")
	if m.queue.Len() != 2 {
		t.Fatalf("A should enqueue all: %d", m.queue.Len())
	}

	m = press(t, m, "d")
	if m.favs.Len() != 1 {
		t.Fatalf("d should remove a favorite: %d", m.favs.Len())
	}
}

func TestEmptyPanelsExplainThemselves(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m = press(t, m, "ctrl+j")
	if m.panel != PanelNone {
		t.Fatal("ctrl+j on an empty queue should not open the panel")
	}
	if !strings.Contains(m.notice.text, "empty") {
		t.Fatalf("notice %q should explain the empty queue", m.notice.text)
	}

	m = press(t, m, "ctrl+f")
	if m.panel != PanelNone {
		t.Fatal("ctrl+f with no favorites should not open the panel")
	}
	if !strings.Contains(m.notice.text, "favorite") {
		t.Fatalf("notice %q", m.notice.text)
	}

	m = press(t, m, "ctrl+h")
	if m.panel != PanelNone {
		t.Fatal("ctrl+h with empty history should not open the panel")
	}
	if !strings.Contains(m.notice.text, "history") {
		t.Fatalf("notice %q", m.notice.text)
	}
}

// ── playback controls ───────────────────────────────────────────────────────

func TestVolumeAndRepeatAndShuffle(t *testing.T) {
	p := player.New()
	m := New(Deps{Player: p, Theme: "terminal"})
	m.width, m.height = 100, 30

	m = press(t, m, "+")
	if got := p.Volume(); got != 75 {
		t.Fatalf("volume %d", got)
	}
	m = press(t, m, "-")
	m = press(t, m, "-")
	if got := p.Volume(); got != 65 {
		t.Fatalf("volume %d", got)
	}

	if m.queue.Repeat() != queue.RepeatOff {
		t.Fatal("repeat starts off")
	}
	m = press(t, m, "ctrl+r")
	if m.queue.Repeat() != queue.RepeatAll {
		t.Fatalf("repeat %v", m.queue.Repeat())
	}
	m = press(t, m, "ctrl+r")
	if m.queue.Repeat() != queue.RepeatOne {
		t.Fatalf("repeat %v", m.queue.Repeat())
	}
	m = press(t, m, "ctrl+r")
	if m.queue.Repeat() != queue.RepeatOff {
		t.Fatalf("repeat %v", m.queue.Repeat())
	}

	m = press(t, m, "ctrl+u")
	if !m.queue.Shuffle() {
		t.Fatal("ctrl+u should enable shuffle")
	}
}

func TestStopDoesNotAdvanceTheQueue(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"), song("b"))
	m.queue.Add(song("a"))
	m.queue.Add(song("b"))
	m.queue.SetCursor(0)
	before := m.queue.Cursor()

	m = press(t, m, "s")

	if m.queue.Cursor() != before {
		t.Fatalf("stop moved the queue cursor: %d -> %d", before, m.queue.Cursor())
	}
	if !strings.Contains(m.notice.text, "stopped") {
		t.Fatalf("notice %q", m.notice.text)
	}
}

func TestThemeCycle(t *testing.T) {
	m := newTestModel(t, 100, 30)
	first := m.themeIdx
	m = press(t, m, "ctrl+t")
	if m.themeIdx == first {
		t.Fatal("ctrl+t should change the theme")
	}
	if !strings.Contains(m.notice.text, "theme") {
		t.Fatalf("notice %q", m.notice.text)
	}
}

func TestVideoModeToggle(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m = press(t, m, "ctrl+v")
	if !m.videoMode {
		t.Fatal("ctrl+v should enable video mode")
	}
	m = press(t, m, "ctrl+v")
	if m.videoMode {
		t.Fatal("ctrl+v should disable video mode again")
	}
}

// ── search flow ─────────────────────────────────────────────────────────────

func TestSearchPrefixes(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m = press(t, m, "/")
	m = press(t, m, "y")
	m = press(t, m, "t")
	m = press(t, m, " ")
	m = press(t, m, "l")
	m = press(t, m, "o")
	m = press(t, m, "f")
	m = press(t, m, "i")

	if m.search.query != "/yt lofi" {
		t.Fatalf("query %q", m.search.query)
	}
	if m.search.source != search.SourceYouTube {
		t.Fatalf("source %q", m.search.source)
	}
	if m.search.term != "lofi" {
		t.Fatalf("term %q", m.search.term)
	}
}

func TestSearchBackspace(t *testing.T) {
	m := newTestModel(t, 100, 30)
	for _, r := range "lofi" {
		m = press(t, m, string(r))
	}
	m = press(t, m, "backspace")
	if m.search.query != "lof" {
		t.Fatalf("query %q", m.search.query)
	}
}

func TestSearchStreamsAndMergesSources(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.query = "lofi"
	m.search.term = "lofi"

	// Started event.
	m.applySearchEvent(search.Event{Query: "lofi", Started: true})
	if !m.search.running {
		t.Fatal("a started event must mark the search as running")
	}
	if len(m.search.pending) == 0 {
		t.Fatal("every source should start pending")
	}

	// YouTube answers.
	m.applySearchEvent(search.Event{
		Query: "lofi", Source: search.SourceYouTube, Done: true,
		Songs: []models.Song{song("yt one"), song("yt two")},
	})
	if len(m.search.bySource[search.SourceYouTube]) != 2 {
		t.Fatalf("youtube results %+v", m.search.bySource)
	}

	// Radio fails.
	m.applySearchEvent(search.Event{
		Query: "lofi", Source: search.SourceRadio, Done: true, Err: errOffline,
	})
	if _, ok := m.search.failures[search.SourceRadio]; !ok {
		t.Fatal("the failure must be recorded")
	}
	if len(m.search.bySource[search.SourceYouTube]) != 2 {
		t.Fatal("one source failing must not drop the other's results")
	}
	// The notice must be actionable.
	if !m.notice.active() || !m.notice.retryable {
		t.Fatalf("notice %+v", m.notice)
	}

	// Everything answered.
	for _, src := range []string{search.SourceNavidrome, search.SourceLibrary, search.SourceFavorites} {
		m.applySearchEvent(search.Event{Query: "lofi", Source: src, Done: true})
	}
	if m.search.running {
		t.Fatal("search should be finished once every source answered")
	}
}

func TestSearchDropsStaleEvents(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.query = "lofi"
	m.search.term = "lofi"
	m.applySearchEvent(search.Event{Query: "lofi", Started: true})
	m.applySearchEvent(search.Event{
		Query: "lofi", Source: search.SourceYouTube, Done: true,
		Songs: []models.Song{song("fresh")},
	})

	// An event for an older query must be ignored.
	m.applySearchEvent(search.Event{
		Query: "lof", Source: search.SourceYouTube, Done: true,
		Songs: []models.Song{song("stale")},
	})
	for _, s := range m.search.bySource[search.SourceYouTube] {
		if s.Title == "stale" {
			t.Fatal("a stale search overwrote fresh results")
		}
	}
}

func TestRetryResetsAndReruns(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.query = "jazz"
	m.search.term = "jazz"
	m.search.lastQuery = "jazz"
	m.notice = retryNotice("Radio unavailable", errOffline)
	m.search.bySource[search.SourceYouTube] = []models.Song{song("old")}

	// retrySearch without a manager must not panic and must keep the query
	// available for the next attempt.
	m.retrySearch()
	if m.search.lastQuery != "jazz" {
		t.Fatalf("lastQuery %q", m.search.lastQuery)
	}
}

func TestPendingSourcesListedWhileSearching(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.pending = map[string]bool{
		search.SourceYouTube: true,
		search.SourceRadio:   true,
	}
	got := m.search.pendingSources()
	if len(got) != 2 {
		t.Fatalf("pending %v", got)
	}
	// Canonical order, not map order.
	if got[0] != search.SourceYouTube || got[1] != search.SourceRadio {
		t.Fatalf("pending order %v", got)
	}
}

// ── narrow terminals ────────────────────────────────────────────────────────

func TestCursorClampedToWindowResize(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"), song("b"), song("c"))
	m.search.cursor = 2
	m.search.group = 0

	// Shrinking the window must not leave a cursor pointing nowhere.
	m.width, m.height = 40, 12
	m.clampCursors()
	if m.search.cursor > len(m.search.groups()[0].Items)-1 {
		t.Fatalf("cursor %d out of range", m.search.cursor)
	}
}

func TestMoveCursorCrossesGroups(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.search.bySource = map[string][]models.Song{
		search.SourceYouTube: {song("a"), song("b")},
		search.SourceRadio:   {song("r1")},
	}
	m.search.pending = map[string]bool{}
	m.clampCursors()

	m.moveCursor(-1) // above the first row stays put
	if m.search.cursor != 0 {
		t.Fatalf("cursor %d", m.search.cursor)
	}

	m.moveCursor(1)
	m.moveCursor(1) // past the end of YouTube → next group
	if m.search.group != 1 || m.search.cursor != 0 {
		t.Fatalf("group %d cursor %d", m.search.group, m.search.cursor)
	}
}

// ── text helpers ────────────────────────────────────────────────────────────

func TestTruncateKeepsWidth(t *testing.T) {
	long := "a very long title that keeps going and going and going"
	for w := 1; w < 40; w++ {
		got := truncate(long, w)
		if lipgloss.Width(got) > w {
			t.Fatalf("truncate(%d) = %q (%d cells)", w, got, lipgloss.Width(got))
		}
	}
	if truncate("short", 20) != "short" {
		t.Fatal("truncate must not touch short strings")
	}
}

func TestFitTwoFits(t *testing.T) {
	for _, w := range []int{10, 20, 40, 80} {
		got := fitTwo("left side", "right", w)
		if lipgloss.Width(got) != w {
			t.Fatalf("fitTwo width %d = %q (%d cells)", w, got, lipgloss.Width(got))
		}
	}
	// Right side always survives.
	got := fitTwo("a very very very long left hand side indeed", "R", 20)
	if !strings.Contains(got, "R") {
		t.Fatalf("right side was dropped: %q", got)
	}
	if lipgloss.Width(got) != 20 {
		t.Fatalf("width %d", lipgloss.Width(got))
	}
}

func TestFormatAgo(t *testing.T) {
	if got := formatAgo(time.Time{}); got != "" {
		t.Fatalf("zero time %q", got)
	}
	if got := formatAgo(time.Now().Add(-30 * time.Second)); got != "just now" {
		t.Fatalf("recent %q", got)
	}
	if got := formatAgo(time.Now().Add(-3 * time.Hour)); got != "3h ago" {
		t.Fatalf("hours %q", got)
	}
	if got := formatAgo(time.Now().Add(-50 * time.Hour)); got == "" {
		t.Fatal("old entries still get a date")
	}
}

func TestSongMeta(t *testing.T) {
	s := models.Song{Title: "T", Artist: "A", Album: "Al", Duration: 125, Bitrate: 320}
	full := songMeta(s, false)
	for _, want := range []string{"A", "Al", "2:05", "320k"} {
		if !strings.Contains(full, want) {
			t.Fatalf("meta %q missing %q", full, want)
		}
	}
	small := songMeta(s, true)
	if strings.Contains(small, "Al") || strings.Contains(small, "320k") {
		t.Fatalf("compact meta should drop album and bitrate: %q", small)
	}
	if !strings.Contains(small, "2:05") {
		t.Fatalf("compact meta must keep the duration: %q", small)
	}
}

// errOffline is the failure used by the search-error tests.

// Closing a panel with esc must return to the focus you came from. If it
// dropped you into the search box, "n"/"p"/space would silently type.
func TestPanelReturnsFocus(t *testing.T) {
	t.Run("from the results list", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "b", song("a"), song("b"))
		m.queue.Add(song("a"))
		m.queue.Add(song("b"))

		m = press(t, m, "ctrl+j")
		if m.panel != PanelQueue {
			t.Fatal("panel did not open")
		}
		if m.focusBeforePanel != FocusList {
			t.Fatal("the previous focus must be remembered")
		}
		m = press(t, m, "esc")
		if m.focus != FocusList {
			t.Fatalf("focus %v, want list", m.focus)
		}
		// And a single-key action still acts.
		m = press(t, m, "n")
		if m.focus != FocusList {
			t.Fatal("n must act, not type")
		}
	})

	t.Run("from the search box", func(t *testing.T) {
		m := newTestModel(t, 100, 30)
		m.queue.Add(song("a"))
		m.focus = FocusSearch

		m = press(t, m, "ctrl+j")
		m = press(t, m, "esc")
		if m.focus != FocusSearch {
			t.Fatalf("focus %v, want search", m.focus)
		}
	})

	t.Run("help", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "b", song("a"))
		m = press(t, m, "ctrl+/")
		if m.panel != PanelHelp {
			t.Fatal("help did not open")
		}
		m = press(t, m, "esc")
		if m.panel != PanelNone {
			t.Fatal("help did not close")
		}
		if m.focus != FocusList {
			t.Fatalf("focus %v", m.focus)
		}
	})
}

// After choosing something to play, the single-key actions must keep working.
func TestPlayFromPanelMovesFocusToList(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "b", song("a"), song("b"))
	m.queue.Add(song("a"))
	m.queue.Add(song("b"))
	m.panel = PanelQueue
	m.panelCursor = 1

	m = press(t, m, "enter")
	if m.panel != PanelNone {
		t.Fatal("the panel must close after playing")
	}
	if m.focus != FocusList {
		t.Fatalf("focus %v, want list", m.focus)
	}
}

// Regression found by driving the real TUI in a pty: after tab → a → ctrl+j →
// esc, the focus came back as "search", so the next `n` typed a letter instead
// of skipping to the next track.
func TestFocusSurvivesEnqueueOpenClose(t *testing.T) {
	m := results(newTestModel(t, 90, 26), "one", song("one"), song("two"))
	m = press(t, m, "tab") // browse the results
	if m.focus != FocusList {
		t.Fatalf("tab must enter the results list: %v", m.focus)
	}
	m = press(t, m, "a")
	if m.queue.Len() != 1 {
		t.Fatalf("queue %d", m.queue.Len())
	}
	if m.focus != FocusList {
		t.Fatalf("focus %v", m.focus)
	}

	m = press(t, m, "ctrl+j")
	if m.panel != PanelQueue || m.focusBeforePanel != FocusList {
		t.Fatalf("panel=%v before=%v", m.panel, m.focusBeforePanel)
	}
	m = press(t, m, "esc")
	if m.panel != PanelNone {
		t.Fatal("panel did not close")
	}
	if m.focus != FocusList {
		t.Fatalf("focus after close = %v, want list", m.focus)
	}

	// The decisive check: n must advance, not type.
	before := m.search.query
	m = press(t, m, "n")
	if m.search.query != before {
		t.Fatalf("n typed %q into the query", m.search.query)
	}
}

func TestEmptySelectionDoesNotChangeFocus(t *testing.T) {
	for _, key := range []string{"a", "A", "f", "i", "d", "c"} {
		m := results(newTestModel(t, 100, 30), "", nil...) // no results at all
		m.search.term = ""
		m = press(t, m, "tab")
		before := m.focus
		m = press(t, m, key)
		if m.focus != before {
			t.Errorf("%q moved the focus from %v to %v", key, before, m.focus)
		}
		// And it must explain itself rather than doing nothing silently.
		if m.notice.text == "" && m.panel == PanelNone {
			t.Errorf("%q with no selection gave no feedback", key)
		}
	}
}

// tab must always mean "move into the results", even with a single source
// group — otherwise it silently cycles the source filter instead.
func TestTabEntersResultsFirst(t *testing.T) {
	t.Run("single group", func(t *testing.T) {
		m := results(newTestModel(t, 100, 30), "one", song("a"), song("b"))
		if m.focus != FocusList {
			t.Fatal("precondition: browse focus")
		}
		m.focus = FocusSearch
		m.search.restrict = "all"
		m = press(t, m, "tab")
		if m.focus != FocusList {
			t.Fatalf("tab must enter the results, focus = %v", m.focus)
		}
		if m.search.restrict != "all" {
			t.Fatalf("tab must not change the source filter (%q)", m.search.restrict)
		}
	})

	t.Run("then cycles groups", func(t *testing.T) {
		m := newTestModel(t, 100, 30)
		m.search.term = "x"
		m.search.bySource = map[string][]models.Song{
			search.SourceYouTube: {song("a")},
			search.SourceRadio:   {song("r")},
		}
		m.search.pending = map[string]bool{}
		m.focus = FocusSearch
		m.clampCursors()

		m = press(t, m, "tab")
		if m.focus != FocusList || m.search.group != 0 {
			t.Fatalf("first tab: focus=%v group=%d", m.focus, m.search.group)
		}
		m = press(t, m, "tab")
		if m.search.group != 1 {
			t.Fatalf("second tab should move to the next group, got %d", m.search.group)
		}
		m = press(t, m, "shift+tab")
		if m.search.group != 0 {
			t.Fatalf("shift+tab should go back, got %d", m.search.group)
		}
	})
}

// With no results at all, tab cycles the source filter.
func TestTabCyclesFilterWhenNoResults(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.sources = []search.Source{
		{Name: search.SourceYouTube, Enabled: true},
		{Name: search.SourceLibrary, Enabled: true},
	}
	m = press(t, m, "tab")
	if m.search.restrict == "" || m.search.restrict == "all" {
		t.Fatalf("tab should have moved off 'all', got %q", m.search.restrict)
	}
}

// Regression from driving the real app: pressing n at the end of the queue did
// nothing at all, because the "end of queue" notice was assigned to a copy.
func TestNextAtEndExplainsItself(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"))
	m.queue.Add(song("a"))
	m.queue.SetCursor(0)

	m = press(t, m, "n") // the only track: we are already at the end
	if m.notice.text == "" {
		t.Fatal("n at the end of the queue must explain itself")
	}
	if !strings.Contains(m.notice.text, "end of queue") {
		t.Fatalf("notice %q", m.notice.text)
	}
}

func TestPrevAtStartReplaysAndRestarts(t *testing.T) {
	// p on the very first track used to do nothing visible.
	m := results(newTestModel(t, 100, 30), "x", song("a"), song("b"))
	m.queue.Add(song("a"))
	m.queue.Add(song("b"))
	m.queue.SetCursor(0)

	m = press(t, m, "p")
	if m.playToken == 0 {
		t.Fatal("p at the start of the queue should still act (replay / restart)")
	}

	// Deep into a track, p restarts it instead of skipping back.
	m2 := results(newTestModel(t, 100, 30), "x", song("a"), song("b"))
	m2.player.SetPreview("a", "", "", "youtube", 30, 300, 50)
	m2.queue.Add(song("a"))
	m2.queue.Add(song("b"))
	m2.queue.SetCursor(1)

	tokenBefore := m2.playToken
	m2 = press(t, m2, "p")
	if m2.playToken != tokenBefore {
		t.Fatal("p deep into a track must restart it, not change tracks")
	}
	if !strings.Contains(m2.notice.text, "restarted") {
		t.Fatalf("notice %q", m2.notice.text)
	}
}

func TestSpaceWithNothingLoadedStartsTheQueue(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"), song("b"))
	m.queue.Add(song("a"))
	m.queue.Add(song("b"))

	// focus=list so space is the pause/play key, not a typed space
	m = press(t, m, "space")
	if m.playToken == 0 {
		t.Fatal("space with nothing loaded should start the queue")
	}
}

func TestSpaceWithEmptyQueueExplainsItself(t *testing.T) {
	m := results(newTestModel(t, 100, 30), "x", song("a"))
	m.queue.Clear()
	m = press(t, m, "space")
	if !strings.Contains(m.notice.text, "nothing to play") {
		t.Fatalf("notice %q", m.notice.text)
	}
}

// A /source-restricted query must not wait forever: only the restricted
// sources are pending, so the spinner ends when that source answers.
func TestRestrictedQueryOnlyWaitsForItsSource(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.sources = []search.Source{
		{Name: search.SourceYouTube, Enabled: true},
		{Name: search.SourceRadio, Enabled: true},
		{Name: search.SourceLibrary, Enabled: true},
	}

	// Type "/local wav"
	for _, r := range "/local wav" {
		m = press(t, m, string(r))
	}

	if len(m.search.pending) != 1 {
		t.Fatalf("pending = %v, want only Library", m.search.pending)
	}
	if !m.search.pending[search.SourceLibrary] {
		t.Fatalf("pending = %v, want Library", m.search.pending)
	}

	// Library answers -> the search is done.
	m.applySearchEvent(search.Event{Query: "wav", Source: search.SourceLibrary, Done: true,
		Songs: []models.Song{{Source: models.SourceLocal, Title: "one.wav"}}})
	if m.search.running {
		t.Fatal("a single-source query must finish when that source answers")
	}
	if len(m.search.pendingSources()) != 0 {
		t.Fatalf("still pending: %v", m.search.pendingSources())
	}
}

// The selector restriction has the same effect as the /prefix.
func TestSelectorRestrictionWaitsForOneSource(t *testing.T) {
	m := newTestModel(t, 100, 30)
	m.sources = []search.Source{
		{Name: search.SourceYouTube, Enabled: true},
		{Name: search.SourceLibrary, Enabled: true},
	}
	m.search.restrict = "local"

	if got := m.activeSourceNames(); len(got) != 1 || got[0] != search.SourceLibrary {
		t.Fatalf("activeSourceNames = %v", got)
	}
}

// Regression: the query handed to the sources must be the TERM, not the raw
// text. Sending "/local wav" made every source search for that literal string,
// and the results were then dropped as a mismatched query — so a prefixed
// search always came back empty.
func TestPrefixedSearchSendsTheTerm(t *testing.T) {
	sm := newFakeSearch()
	m := New(Deps{Search: sm, Theme: "terminal"})
	m.width, m.height = 100, 30
	m.sources = []search.Source{
		{Name: search.SourceYouTube, Enabled: true},
		{Name: search.SourceLibrary, Enabled: true},
	}

	for _, r := range "/local wav" {
		m = press(t, m, string(r))
	}

	if got := sm.lastQuery(); got != "wav" {
		t.Fatalf("the source received %q, want %q", got, "wav")
	}
	if got := sm.restriction(); len(got) != 1 || got[0] != "Library" {
		t.Fatalf("restriction = %v, want [Library]", got)
	}
	if m.search.term != "wav" {
		t.Fatalf("term = %q", m.search.term)
	}
	if m.search.lastQuery != "wav" {
		t.Fatalf("lastQuery = %q (retry would search the wrong text)", m.search.lastQuery)
	}
}

func TestPlainSearchHasNoRestriction(t *testing.T) {
	sm := newFakeSearch()
	m := New(Deps{Search: sm, Theme: "terminal"})
	m.width, m.height = 100, 30

	for _, r := range "lofi" {
		m = press(t, m, string(r))
	}
	if got := sm.lastQuery(); got != "lofi" {
		t.Fatalf("query %q", got)
	}
	if got := sm.restriction(); len(got) != 0 {
		t.Fatalf("restriction = %v, want none", got)
	}
}
