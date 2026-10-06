package discover

import (
	"context"
	"errors"
	"testing"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/store"
)

func fakeSearch(found map[string][]models.Song) SearchFunc {
	return func(_ context.Context, q string, _ int) ([]models.Song, error) {
		s, ok := found[q]
		if !ok {
			return nil, errors.New("no results")
		}
		return s, nil
	}
}

func TestDefaultSeeds(t *testing.T) {
	seeds := DefaultSeeds([]string{"hardcore", "uptempo"}, []string{"Angerfist"})
	if len(seeds) < 4 { // 2 genres + 1 artist + weekly
		t.Fatalf("expected >=4 seeds, got %d", len(seeds))
	}
	last := seeds[len(seeds)-1]
	if last.Kind != store.PlaylistWeekly || last.ID != "weekly-new" {
		t.Fatalf("last seed should be weekly, got %+v", last)
	}
	empty := DefaultSeeds(nil, nil)
	if len(empty) == 0 {
		t.Fatal("empty taste should still seed defaults")
	}
}

func TestRefreshDedups(t *testing.T) {
	pl := store.Playlist{ID: "x", Kind: store.PlaylistTaste, Query: "hardcore"}
	s := models.Song{ID: "yt:1", Source: models.SourceYouTube, Title: "a"}
	out, err := Refresh(context.Background(), pl, fakeSearch(map[string][]models.Song{
		"hardcore": {s, s, {ID: "yt:2", Source: models.SourceYouTube, Title: "b"}},
	}), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected dedup to 2, got %d", len(out))
	}
	if _, err := Refresh(context.Background(), pl, fakeSearch(nil), 10); err == nil {
		t.Fatal("expected error on empty results")
	}
}
