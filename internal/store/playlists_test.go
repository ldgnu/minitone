package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

func TestPlaylistsUpsertSetTracksStale(t *testing.T) {
	dir := t.TempDir()
	p := NewPlaylists(filepath.Join(dir, "playlists.json"))
	p.Upsert(Playlist{ID: "weekly-new", Name: "weekly", Kind: PlaylistWeekly, AutoRefresh: true, UpdatedAt: time.Now().Add(-8 * 24 * time.Hour)})
	if got := p.StaleIDs(7 * 24 * time.Hour); len(got) != 1 {
		t.Fatalf("expected 1 stale, got %v", got)
	}
	ok := p.SetTracks("weekly-new", []models.Song{{ID: "yt:abc", Source: models.SourceYouTube, Title: "t"}})
	if !ok {
		t.Fatal("SetTracks failed")
	}
	if got := p.StaleIDs(7 * 24 * time.Hour); len(got) != 0 {
		t.Fatalf("expected fresh after SetTracks, got %v", got)
	}
	// Reload persists.
	p2 := NewPlaylists(filepath.Join(dir, "playlists.json"))
	if p2.Len() != 1 || len(p2.Get("weekly-new").Tracks) != 1 {
		t.Fatal("playlists did not persist")
	}
}
