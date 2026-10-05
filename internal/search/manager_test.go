package search

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

// collector gathers streaming events for assertions.
type collector struct {
	mu     sync.Mutex
	events []Event
}

func (c *collector) fn(ev Event) {
	c.mu.Lock()
	c.events = append(c.events, ev)
	c.mu.Unlock()
}

func (c *collector) snapshot() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Event{}, c.events...)
}

// waitFor polls until cond is satisfied or the timeout expires.
func (c *collector) waitFor(t *testing.T, timeout time.Duration, what string, cond func([]Event) bool) []Event {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		evs := c.snapshot()
		if cond(evs) {
			return evs
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s; got %d events: %+v", what, len(c.snapshot()), c.snapshot())
	return nil
}

func countSources(evs []Event) map[string]int {
	out := map[string]int{}
	for _, e := range evs {
		if e.Done && e.Source != "" {
			out[e.Source]++
		}
	}
	return out
}

func TestManagerStreamsPerSource(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)

	slow := make(chan struct{})
	m.AddSearcher(NewSearcher(SourceYouTube, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "yt-" + q, Source: models.SourceYouTube}}, nil
	}))
	m.AddSearcher(NewSearcher(SourceRadio, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		<-slow // a slow source must not hold back the fast one
		return []models.Song{{Title: "radio-" + q, Source: models.SourceRadio}}, nil
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.SearchNow("test")

	// The fast source arrives before the slow one finishes: that is the whole
	// point of streaming instead of wg.Wait().
	c.waitFor(t, 2*time.Second, "fast source result", func(evs []Event) bool {
		return countSources(evs)[SourceYouTube] == 1
	})
	evs := c.snapshot()
	if countSources(evs)[SourceRadio] != 0 {
		t.Fatal("slow source result arrived early")
	}

	close(slow)
	evs = c.waitFor(t, 2*time.Second, "both sources", func(evs []Event) bool {
		return countSources(evs)[SourceRadio] == 1
	})

	if evs[0].Started != true {
		t.Fatal("first event must be the Started marker")
	}
	groups := BuildGroups(map[string][]models.Song{
		SourceYouTube: {{Title: "yt-test"}},
		SourceRadio:   {{Title: "radio-test"}},
	})
	if len(groups) != 2 || CountTotal(groups) != 2 {
		t.Fatalf("groups %+v", groups)
	}
	if groups[0].Name != SourceYouTube {
		t.Fatalf("canonical source order broken: %+v", groups)
	}
}

func TestManagerReportsPerSourceError(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)
	m.AddSearcher(NewSearcher(SourceRadio, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return nil, errors.New("all mirrors unreachable")
	}))
	m.AddSearcher(NewSearcher(SourceYouTube, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "ok", Source: models.SourceYouTube}}, nil
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.SearchNow("boom")

	evs := c.waitFor(t, 2*time.Second, "error event", func(evs []Event) bool {
		for _, e := range evs {
			if e.Err != nil {
				return true
			}
		}
		return false
	})

	var found Event
	for _, e := range evs {
		if e.Err != nil {
			found = e
		}
	}
	if found.Source != SourceRadio {
		t.Fatalf("error attributed to %q", found.Source)
	}
	if !found.Done {
		t.Fatal("error event must be marked Done")
	}
	// A failing source must not remove the healthy one.
	if countSources(evs)[SourceYouTube] != 1 {
		t.Fatal("healthy source lost because another failed")
	}
	if (SourceError{Source: found.Source, Err: found.Err}).Error() == "" {
		t.Fatal("SourceError should render a message")
	}
}

func TestManagerStaleResultsAreDropped(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)

	release := make(chan struct{})
	m.AddSearcher(NewSearcher(SourceYouTube, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		if q == "old" {
			<-release
			// Ignore the cancellation on purpose: a searcher that answers late
			// must still be ignored by the manager.
			return []models.Song{{Title: "stale"}}, nil
		}
		return []models.Song{{Title: "fresh-" + q}}, nil
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.SearchNow("old")
	time.Sleep(50 * time.Millisecond)
	m.SearchNow("new") // supersedes "old"

	c.waitFor(t, 2*time.Second, "fresh result", func(evs []Event) bool {
		for _, e := range evs {
			for _, s := range e.Songs {
				if s.Title == "fresh-new" {
					return true
				}
			}
		}
		return false
	})
	close(release)
	time.Sleep(150 * time.Millisecond)

	for _, e := range c.snapshot() {
		for _, s := range e.Songs {
			if s.Title == "stale" {
				t.Fatal("a superseded search leaked stale results")
			}
		}
	}
}

func TestManagerEmptyQueryEmitsDone(t *testing.T) {
	m := NewManager()
	c := &collector{}
	m.OnEvent(c.fn)
	m.Search("")

	evs := c.snapshot()
	if len(evs) != 1 || !evs[0].Done {
		t.Fatalf("empty query should emit a single Done event, got %+v", evs)
	}
}

func TestManagerCancelDropsPending(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)
	started := make(chan struct{})
	m.AddSearcher(NewSearcher(SourceYouTube, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return []models.Song{{Title: "late"}}, nil
		}
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.SearchNow("slow")
	<-started
	m.Cancel()
	time.Sleep(200 * time.Millisecond)

	for _, e := range c.snapshot() {
		for _, s := range e.Songs {
			if s.Title == "late" {
				t.Fatal("cancelled search should not emit results")
			}
		}
	}
}

func TestManagerRestrictTo(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)
	m.AddSearcher(NewSearcher(SourceYouTube, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "yt"}}, nil
	}))
	m.AddSearcher(NewSearcher(SourceRadio, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "radio"}}, nil
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.RestrictTo([]string{"radio"})
	m.SearchNow("x")
	c.waitFor(t, 2*time.Second, "radio only", func(evs []Event) bool {
		return countSources(evs)[SourceRadio] == 1
	})
	time.Sleep(100 * time.Millisecond)

	for _, e := range c.snapshot() {
		if e.Source == SourceYouTube {
			t.Fatal("restricted search still queried YouTube")
		}
	}

	// Back to all sources.
	c2 := &collector{}
	m.OnEvent(c2.fn)
	m.RestrictTo(nil)
	m.SearchNow("x")
	c2.waitFor(t, 2*time.Second, "both sources", func(evs []Event) bool {
		n := countSources(evs)
		return n[SourceYouTube] == 1 && n[SourceRadio] == 1
	})
}

func TestManagerDescribe(t *testing.T) {
	m := NewManager()
	m.AddSearcher(NewSearcher(SourceRadio, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return nil, nil
	}))
	m.AddSearcher(unavailableSearcher{})

	got := m.Describe()
	if len(got) != 2 {
		t.Fatalf("describe %+v", got)
	}
	// Canonical order: YouTube, Radio, ... then unknown ones last.
	if got[0].Name != SourceRadio || !got[0].Enabled {
		t.Fatalf("unexpected first source: %+v", got)
	}
	if got[1].Enabled || got[1].Hint == "" {
		t.Fatalf("unavailable source should be reported as such: %+v", got[1])
	}
}

type unavailableSearcher struct{}

func (unavailableSearcher) Search(ctx context.Context, q string, limit int) ([]models.Song, error) {
	return nil, nil
}
func (unavailableSearcher) Name() string              { return SourceNavidrome }
func (unavailableSearcher) Available() bool           { return false }
func (unavailableSearcher) UnavailableReason() string { return "not configured" }

func TestParseQuery(t *testing.T) {
	cases := []struct {
		in         string
		wantSource string
		wantTerm   string
	}{
		{"lofi", "", "lofi"},
		{"/youtube lofi girl", SourceYouTube, "lofi girl"},
		{"/yt lofi", SourceYouTube, "lofi"},
		{"/radio jazz", SourceRadio, "jazz"},
		{"/navidrome beatles", SourceNavidrome, "beatles"},
		{"/nav x", SourceNavidrome, "x"},
		{"/local miles", SourceLibrary, "miles"},
		{"/lib x", SourceLibrary, "x"},
		{"/search all kinds", "", "all kinds"},
		{"/fav x", SourceFavorites, "x"},
		{"//weird", "", "//weird"},
		{"/nonsense x", "", "/nonsense x"},
		{"/", "", "/"},
		{"/youtube", "", "/youtube"},
	}
	for _, c := range cases {
		src, term := ParseQuery(c.in)
		if src != c.wantSource || term != c.wantTerm {
			t.Errorf("ParseQuery(%q) = (%q,%q) want (%q,%q)", c.in, src, term, c.wantSource, c.wantTerm)
		}
	}
}

func TestBuildGroupsSkipsEmptyAndKeepsOrder(t *testing.T) {
	groups := BuildGroups(map[string][]models.Song{
		SourceLibrary: {{Title: "l"}},
		SourceYouTube: {},
		SourceRadio:   {{Title: "r1"}, {Title: "r2"}},
	})
	if len(groups) != 2 {
		t.Fatalf("groups %+v", groups)
	}
	if groups[0].Name != SourceRadio || groups[1].Name != SourceLibrary {
		t.Fatalf("order %+v", groups)
	}
	if groups[0].Index != 0 || groups[1].Index != 1 {
		t.Fatal("index must be sequential")
	}
	if CountTotal(groups) != 3 {
		t.Fatalf("total %d", CountTotal(groups))
	}
}

func TestRankPutsExactTitleFirst(t *testing.T) {
	songs := []models.Song{
		{Title: "Something Else Entirely", Artist: "Nobody"},
		{Title: "lofi", Artist: "Nobody"},
		{Title: "Lofi Girl Radio", Artist: "Lofi Girl"},
	}
	got := rank("lofi", songs)
	if got[0].Title != "lofi" {
		t.Fatalf("exact title should rank first, got %q", got[0].Title)
	}
}

func TestSetDebounceClamped(t *testing.T) {
	m := NewManager()
	m.SetDebounce(5)
	if d := m.debouncer.Delay(); d != 100*time.Millisecond {
		t.Fatalf("debounce %v", d)
	}
	m.SetDebounce(5000)
	if d := m.debouncer.Delay(); d != 800*time.Millisecond {
		t.Fatalf("debounce %v", d)
	}
	m.SetDebounce(250)
	if d := m.debouncer.Delay(); d != 250*time.Millisecond {
		t.Fatalf("debounce %v", d)
	}
}

func TestCanonicalSource(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		found bool
	}{
		{"youtube", SourceYouTube, true},
		{"YouTube", SourceYouTube, true},
		{"yt", SourceYouTube, true},
		{"radio", SourceRadio, true},
		{"radio-browser", "", false},
		{"local", SourceLibrary, true},
		{"library", SourceLibrary, true},
		{"navidrome", SourceNavidrome, true},
		{"subsonic", SourceNavidrome, true},
		{"fav", SourceFavorites, true},
		{"all", "", true},
		{"", "", true},
		{"nonsense", "", false},
	}
	for _, c := range cases {
		got, ok := CanonicalSource(c.in)
		if got != c.want || ok != c.found {
			t.Errorf("CanonicalSource(%q) = (%q,%v) want (%q,%v)", c.in, got, ok, c.want, c.found)
		}
	}
}

// Regression: "local" (the /prefix and the obvious config value) must resolve
// to the Library source. A name that matches nothing must mean "all sources"
// rather than a silently empty search.
func TestRestrictToUnknownNameFallsBackToAll(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)
	m.AddSearcher(NewSearcher(SourceLibrary, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "lib"}}, nil
	}))
	m.AddSearcher(NewSearcher(SourceRadio, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "radio"}}, nil
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.RestrictTo([]string{"nonsense"})
	m.SearchNow("x")

	c.waitFor(t, 2*time.Second, "both sources", func(evs []Event) bool {
		n := countSources(evs)
		return n[SourceLibrary] == 1 && n[SourceRadio] == 1
	})
}

func TestRestrictToLocalAlias(t *testing.T) {
	m := NewManager()
	m.SetDebounce(10)
	m.AddSearcher(NewSearcher(SourceLibrary, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "lib"}}, nil
	}))
	m.AddSearcher(NewSearcher(SourceRadio, func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "radio"}}, nil
	}))

	c := &collector{}
	m.OnEvent(c.fn)
	m.RestrictTo([]string{"local"})
	m.SearchNow("x")

	c.waitFor(t, 2*time.Second, "library result", func(evs []Event) bool {
		return countSources(evs)[SourceLibrary] == 1
	})
	time.Sleep(100 * time.Millisecond)
	for _, e := range c.snapshot() {
		if e.Source == SourceRadio {
			t.Fatal("local restriction still queried Radio")
		}
	}
}
