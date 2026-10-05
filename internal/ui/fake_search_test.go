package ui

import (
	"sync"

	"github.com/ldgnu/minitone/internal/search"
)

// fakeSearch records what the UI asks of the search manager, so tests can
// assert on the query and the restriction without any network.
type fakeSearch struct {
	mu         sync.Mutex
	queries    []string
	restrictTo []string
	now        int
	cancelled  int
	events     []search.Event
	handlers   []func(search.Event)
}

func newFakeSearch() *fakeSearch { return &fakeSearch{} }

func (f *fakeSearch) Search(query string) {
	f.mu.Lock()
	f.queries = append(f.queries, query)
	f.mu.Unlock()
}

func (f *fakeSearch) SearchNow(query string) { f.Search(query) }

func (f *fakeSearch) RestrictTo(names []string) {
	f.mu.Lock()
	f.restrictTo = append([]string(nil), names...)
	f.mu.Unlock()
}

func (f *fakeSearch) SetDebounce(int) {}

func (f *fakeSearch) Describe() []search.Source {
	return []search.Source{
		{Name: search.SourceYouTube, Enabled: true},
		{Name: search.SourceRadio, Enabled: true},
		{Name: search.SourceLibrary, Enabled: true},
	}
}

func (f *fakeSearch) OnEvent(fn func(search.Event)) {
	f.mu.Lock()
	f.handlers = append(f.handlers, fn)
	f.mu.Unlock()
}

func (f *fakeSearch) Cancel() {
	f.mu.Lock()
	f.cancelled++
	f.mu.Unlock()
}

// ── assertions ──────────────────────────────────────────────────────────────

func (f *fakeSearch) lastQuery() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queries) == 0 {
		return ""
	}
	return f.queries[len(f.queries)-1]
}

func (f *fakeSearch) allQueries() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.queries...)
}

func (f *fakeSearch) restriction() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.restrictTo...)
}

func (f *fakeSearch) cancelCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cancelled
}

// emit pushes an event as if a source had answered.
func (f *fakeSearch) emit(ev search.Event) {
	f.mu.Lock()
	handlers := append([]func(search.Event){}, f.handlers...)
	f.events = append(f.events, ev)
	f.mu.Unlock()
	for _, h := range handlers {
		h(ev)
	}
}
