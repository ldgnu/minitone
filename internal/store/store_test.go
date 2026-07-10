package store

import (
	"path/filepath"
	"testing"

	"github.com/ldgnu/minitone/internal/models"
)

func song(id, title string) models.Song {
	return models.Song{ID: id, Title: title, Source: models.SourceYouTube, SourceID: id}
}

func TestFavoritesTogglePersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "favorites.json")

	f := NewFavorites(path)
	s := song("yt:1", "Track One")
	if !f.Add(s) {
		t.Fatal("add")
	}
	if f.Add(s) {
		t.Fatal("duplicate add should be false")
	}
	if !f.Contains(s) {
		t.Fatal("contains")
	}
	if f.Len() != 1 {
		t.Fatal("len")
	}

	// Reload from disk
	f2 := NewFavorites(path)
	if f2.Len() != 1 {
		t.Fatalf("reload len %d", f2.Len())
	}
	if !f2.Contains(s) {
		t.Fatal("reload contains")
	}

	if f2.Toggle(s) {
		t.Fatal("toggle should remove")
	}
	if f2.Contains(s) {
		t.Fatal("removed")
	}

	f3 := NewFavorites(path)
	if f3.Len() != 0 {
		t.Fatal("persist remove")
	}
}

func TestFavoritesSearch(t *testing.T) {
	f := NewFavorites("")
	f.Add(song("1", "Paranoid Android"))
	f.Add(models.Song{ID: "2", Title: "Karma Police", Artist: "Radiohead"})
	res := f.Search("paranoid", 10)
	if len(res) != 1 || res[0].Title != "Paranoid Android" {
		t.Fatalf("%+v", res)
	}
	res = f.Search("radiohead", 10)
	if len(res) != 1 {
		t.Fatalf("%+v", res)
	}
}

func TestFavoritesRemoveAt(t *testing.T) {
	f := NewFavorites("")
	f.Add(song("a", "A"))
	f.Add(song("b", "B"))
	f.Add(song("c", "C"))
	if !f.RemoveAt(1) {
		t.Fatal()
	}
	if f.Len() != 2 {
		t.Fatal()
	}
	if f.Get(0).Title != "A" || f.Get(1).Title != "C" {
		t.Fatal()
	}
}

func TestHistoryPushDedupeAndMax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	h := NewHistory(path, 3)

	h.Push(song("1", "One"))
	h.Push(song("2", "Two"))
	h.Push(song("3", "Three"))
	h.Push(song("4", "Four"))
	if h.Len() != 3 {
		t.Fatalf("max len %d", h.Len())
	}
	// Newest first
	if h.Get(0).Title != "Four" {
		t.Fatalf("newest %s", h.Get(0).Title)
	}

	// Replay older moves to front
	h.Push(song("2", "Two"))
	if h.Get(0).Title != "Two" {
		t.Fatal("move front")
	}
	// Still unique
	count := 0
	for _, s := range h.Songs() {
		if s.ID == "2" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("dedupe count %d", count)
	}

	h2 := NewHistory(path, 3)
	if h2.Len() != 3 {
		t.Fatalf("reload %d", h2.Len())
	}
}

func TestHistoryClear(t *testing.T) {
	h := NewHistory("", 10)
	h.Push(song("1", "One"))
	h.Clear()
	if h.Len() != 0 {
		t.Fatal()
	}
}

func TestSongKey(t *testing.T) {
	s := models.Song{ID: "yt:abc"}
	if s.Key() != "yt:abc" {
		t.Fatal()
	}
	s = models.Song{Source: models.SourceRadio, SourceID: "uuid"}
	if s.Key() != "radio:uuid" {
		t.Fatal(s.Key())
	}
}
