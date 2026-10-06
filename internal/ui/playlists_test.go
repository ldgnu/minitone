package ui

import (
	"testing"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/store"
)

func ytSong(title, artist string) models.Song {
	return models.Song{
		ID: "yt:" + title, Source: models.SourceYouTube, SourceID: "abc",
		Title: title, Artist: artist, URL: "https://www.youtube.com/watch?v=abc",
	}
}

func playlistsModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t, 100, 30)
	m.playls.Upsert(store.Playlist{
		ID: "taste-hardcore", Name: "hardcore", Kind: store.PlaylistTaste,
		Query: "hardcore", AutoRefresh: true,
		Tracks: []models.Song{ytSong("t1", "Angerfist"), ytSong("t2", "Miss K8")},
	})
	m.playls.Upsert(store.Playlist{
		ID: "weekly-new", Name: "weekly", Kind: store.PlaylistWeekly,
		Query: "new music", AutoRefresh: true,
	})
	return m
}

// Opening the panel with playlists must work; empty store must refuse.
func TestPlaylistsPanelOpen(t *testing.T) {
	m := playlistsModel(t)
	m.openPanelCmd(PanelPlaylists)
	if m.panel != PanelPlaylists {
		t.Fatal("panel should open with playlists")
	}
	empty := newTestModel(t, 100, 30)
	empty.openPanelCmd(PanelPlaylists)
	if empty.panel == PanelPlaylists {
		t.Fatal("panel should not open when empty")
	}
}

// Enter drills into tracks, esc goes back to the list, panelLen follows.
func TestPlaylistsDrill(t *testing.T) {
	m := playlistsModel(t)
	m.openPanelCmd(PanelPlaylists)
	if got := m.panelLen(); got != 2 {
		t.Fatalf("level 0 len = %d, want 2", got)
	}
	mm, _ := m.handlePlaylistsPanelKey(m.keys.Enter.key)
	m = mm.(Model)
	if m.plLevel != 1 || m.plIndex != 0 {
		t.Fatalf("should drill into first playlist, got level=%d idx=%d", m.plLevel, m.plIndex)
	}
	if got := m.panelLen(); got != 2 {
		t.Fatalf("level 1 len = %d, want 2 tracks", got)
	}
	if s, ok := m.panelSong(0); !ok || s.Title != "t1" {
		t.Fatalf("track 0 = %+v", s)
	}
	mm, _ = m.handlePanelKey("esc")
	m = mm.(Model)
	if m.plLevel != 0 || m.panel != PanelPlaylists {
		t.Fatal("esc should go back to playlist list")
	}
}

// Refresh messages land tracks and clear staleness.
func TestPlaylistRefreshMsg(t *testing.T) {
	m := playlistsModel(t)
	m.onPlaylistRefresh(playlistRefreshMsg{
		id:     "weekly-new",
		tracks: []models.Song{ytSong("fresh", "Rebelion")},
	})
	pl := m.playls.Get("weekly-new")
	if pl == nil || len(pl.Tracks) != 1 || pl.Tracks[0].Title != "fresh" {
		t.Fatalf("refresh did not land: %+v", pl)
	}
}

// Enqueue-all from a track level queues only that playlist.
func TestPlaylistsEnqueueAll(t *testing.T) {
	m := playlistsModel(t)
	m.openPanelCmd(PanelPlaylists)
	mm, _ := m.handlePlaylistsPanelKey(m.keys.Enter.key)
	m = mm.(Model)
	m.enqueuePanelAll()
	if m.queue.Len() != 2 {
		t.Fatalf("queue = %d, want 2", m.queue.Len())
	}
}

// Deleting a track keeps the playlist; deleting a list removes it.
func TestPlaylistsDelete(t *testing.T) {
	m := playlistsModel(t)
	m.openPanelCmd(PanelPlaylists)
	mm, _ := m.handlePlaylistsPanelKey(m.keys.Enter.key)
	m = mm.(Model)
	m.deletePanelItem()
	if got := len(m.playls.Get("taste-hardcore").Tracks); got != 1 {
		t.Fatalf("tracks = %d, want 1", got)
	}
	mm, _ = m.handlePanelKey("esc")
	m = mm.(Model)
	m.panelCursor = 1
	m.deletePanelItem()
	if m.playls.Len() != 1 {
		t.Fatalf("playlists = %d, want 1", m.playls.Len())
	}
}
