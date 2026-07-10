package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanAndSearch(t *testing.T) {
	dir := t.TempDir()
	// Artist/Album/track.mp3 layout
	trackDir := filepath.Join(dir, "Radiohead", "OK Computer")
	if err := os.MkdirAll(trackDir, 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(trackDir, "Paranoid Android.mp3")
	if err := os.WriteFile(track, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	// non-audio ignored
	_ = os.WriteFile(filepath.Join(trackDir, "cover.jpg"), []byte("x"), 0o644)

	s := NewWithDirs([]string{dir})
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 1 {
		t.Fatalf("len=%d", s.Len())
	}

	res, err := s.Search("paranoid", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("search got %d", len(res))
	}
	if res[0].Title != "Paranoid Android" {
		t.Fatalf("title %q", res[0].Title)
	}

	res, _ = s.Search("zzzz-not-found", 10)
	if len(res) != 0 {
		t.Fatal("expected empty")
	}
}

func TestAddDirIdempotent(t *testing.T) {
	s := New()
	s.AddDir("/tmp/a")
	s.AddDir("/tmp/a")
	if len(s.Dirs()) != 1 {
		t.Fatal()
	}
}
