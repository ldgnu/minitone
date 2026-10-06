package discover

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/store"
)

// SearchFunc resolves one query to YouTube songs (ytsearch).
type SearchFunc func(ctx context.Context, query string, limit int) ([]models.Song, error)

// DefaultSeeds builds the initial playlists from a taste profile.
// This is "lo más fácil": no login, no Apple API — just ytsearch queries
// derived from what you like on Apple Music (hardcore, uptempo, artists).
func DefaultSeeds(genres, artists []string) []store.Playlist {
	now := time.Now()
	var out []store.Playlist
	if len(genres) == 0 && len(artists) == 0 {
		genres = []string{"hardcore", "uptempo"}
	}
	for _, g := range genres {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		id := "taste-" + slug(g)
		out = append(out, store.Playlist{
			ID:          id,
			Name:        "❤ " + g,
			Kind:        store.PlaylistTaste,
			Query:       g + " hardcore uptempo mix",
			Tracks:      nil,
			UpdatedAt:   time.Time{},
			AutoRefresh: true,
		})
		_ = now
	}
	for _, a := range artists {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		out = append(out, store.Playlist{
			ID:          "taste-" + slug(a),
			Name:        "★ " + a,
			Kind:        store.PlaylistTaste,
			Query:       a,
			AutoRefresh: true,
		})
	}
	out = append(out, store.Playlist{
		ID:          "weekly-new",
		Name:        "✨ Novedades semanales",
		Kind:        store.PlaylistWeekly,
		Query:       WeeklyQuery(genres, artists),
		AutoRefresh: true,
	})
	return out
}

// WeeklyQuery combines taste into one discovery query.
func WeeklyQuery(genres, artists []string) string {
	parts := append(append([]string{}, genres...), artists...)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) == 0 {
		return "hardcore uptempo new music"
	}
	return strings.Join(parts, " ") + " new music"
}

// Refresh fills one playlist via SearchFunc, deduping by song key.
func Refresh(ctx context.Context, pl store.Playlist, search SearchFunc, perQuery int) ([]models.Song, error) {
	if perQuery <= 0 {
		perQuery = 10
	}
	queries := []string{pl.Query}
	if pl.Kind == store.PlaylistWeekly {
		// Weekly mix: split the query so one dead term can't empty the list.
		queries = splitQuery(pl.Query)
	}
	seen := map[string]bool{}
	var out []models.Song
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		songs, err := search(ctx, q, perQuery)
		if err != nil {
			continue
		}
		for _, s := range songs {
			k := s.Key()
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no results for %q", pl.Query)
	}
	return out, nil
}

func splitQuery(q string) []string {
	// "hardcore uptempo Angerfist new music" -> per-term searches + full query.
	words := strings.Fields(q)
	if len(words) <= 2 {
		return []string{q}
	}
	seen := map[string]bool{}
	var out []string
	for _, w := range words {
		if w == "new" || w == "music" || w == "mix" {
			continue
		}
		if !seen[w] {
			seen[w] = true
			out = append(out, w+" new music")
		}
		if len(out) >= 4 {
			break
		}
	}
	out = append(out, q)
	return out
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if out == "" {
		out = "mix"
	}
	return out
}
