package search

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]models.Song, error)
	Name() string
}

type SearcherFunc struct {
	Fn   func(ctx context.Context, query string, limit int) ([]models.Song, error)
	name string
}

func (s SearcherFunc) Search(ctx context.Context, query string, limit int) ([]models.Song, error) {
	return s.Fn(ctx, query, limit)
}

func (s SearcherFunc) Name() string {
	return s.name
}

func NewSearcher(name string, fn func(ctx context.Context, query string, limit int) ([]models.Song, error)) Searcher {
	return SearcherFunc{Fn: fn, name: name}
}

// Source describes a configured source, used to build the source selector.
type Source struct {
	Name    string
	Enabled bool
	Hint    string // short note shown in the selector, e.g. "no yt-dlp"
}

// Describable is implemented by searchers that can explain why they are
// unavailable, so the UI can show it instead of silently returning nothing.
type Describable interface {
	Available() bool
	UnavailableReason() string
}

// Describe returns the display metadata of every registered searcher.
func (m *Manager) Describe() []Source {
	m.mu.Lock()
	searchers := append([]Searcher{}, m.searchers...)
	m.mu.Unlock()

	out := make([]Source, 0, len(searchers))
	for _, s := range searchers {
		src := Source{Name: s.Name(), Enabled: true}
		if d, ok := s.(Describable); ok && !d.Available() {
			src.Enabled = false
			src.Hint = d.UnavailableReason()
		}
		out = append(out, src)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return sourceRank(out[i].Name) < sourceRank(out[j].Name)
	})
	return out
}

func sourceRank(name string) int {
	for i, n := range sourceOrder {
		if n == name {
			return i
		}
	}
	return len(sourceOrder)
}

// ParseQuery understands the optional /source prefix:
//
//	/search term  /youtube term  /radio term  /navidrome term  /local term
//
// A query without a prefix is returned unchanged with an empty source.
func ParseQuery(q string) (source, term string) {
	q = strings.TrimSpace(q)
	if !strings.HasPrefix(q, "/") {
		return "", q
	}
	rest := q[1:]
	sp := strings.IndexAny(rest, " \t")
	if sp < 0 {
		return "", q // a lone "/" is just a search for a slash
	}
	head := strings.ToLower(rest[:sp])
	tail := strings.TrimSpace(rest[sp+1:])

	// One alias table drives both the /prefix and default_source, so the two
	// can never disagree.
	if canonical, ok := CanonicalSource(head); ok {
		return canonical, tail
	}
	return "", q
}

// FormatElapsed is a tiny helper for logging/debug of search timings.
func FormatElapsed(d time.Duration) string {
	return d.Round(time.Millisecond).String()
}
