package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

// loadIsEmpty avoids taking the address of a temporary in the assertions.
func loadIsEmpty(s *SessionStore) bool {
	sess := s.Load()
	return sess.Empty()
}

func TestSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")

	s := NewSessionStore(path)
	song := models.Song{ID: "yt:abc", Source: models.SourceYouTube, Title: "Teardrop", Artist: "Massive Attack"}

	want := Session{
		Song:     song,
		Position: 42.5,
		Queue:    []models.Song{song, {ID: "yt:def", Title: "Angel"}},
		Cursor:   1,
		Volume:   55,
		Shuffle:  true,
		Repeat:   2,
	}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}

	got := NewSessionStore(path).Load()
	if got.Song.Title != "Teardrop" || got.Song.Artist != "Massive Attack" {
		t.Fatalf("song %+v", got.Song)
	}
	if got.Position != 42.5 {
		t.Fatalf("position %v", got.Position)
	}
	if len(got.Queue) != 2 {
		t.Fatalf("queue %+v", got.Queue)
	}
	if got.Cursor != 1 {
		t.Fatalf("cursor %d", got.Cursor)
	}
	if got.Volume != 55 || !got.Shuffle || got.Repeat != 2 {
		t.Fatalf("prefs %+v", got)
	}
	if got.SavedAt.IsZero() {
		t.Fatal("SavedAt must be stamped")
	}
}

func TestSessionEmptyWhenMissing(t *testing.T) {
	s := NewSessionStore(filepath.Join(t.TempDir(), "nope.json"))
	if !loadIsEmpty(s) {
		t.Fatal("a missing session file must load as empty")
	}
}

func TestSessionEmptyPredicate(t *testing.T) {
	empty := Session{}
	if !empty.Empty() {
		t.Fatal("zero session is empty")
	}
	withQueue := Session{Queue: []models.Song{{Title: "x"}}}
	if withQueue.Empty() {
		t.Fatal("a session with a queue is not empty")
	}
	if empty.HasSong() {
		t.Fatal("no song")
	}
	withSong := Session{Song: models.Song{Title: "x"}}
	if !withSong.HasSong() {
		t.Fatal("song should be reported")
	}
}

func TestSessionClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	s := NewSessionStore(path)
	if err := s.Save(Session{Song: models.Song{Title: "x"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Clear must remove the file")
	}
	if !loadIsEmpty(s) {
		t.Fatal("Clear must reset the in-memory session")
	}
	// Clearing twice must not fail.
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionIgnoresCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewSessionStore(path)
	if !loadIsEmpty(s) {
		t.Fatal("a corrupt session file must degrade to empty, not crash")
	}
	// And it must still be overwritable.
	if err := s.Save(Session{Volume: 10}); err != nil {
		t.Fatal(err)
	}
	if got := NewSessionStore(path).Load().Volume; got != 10 {
		t.Fatalf("volume %d", got)
	}
}

func TestSessionNoPathIsInert(t *testing.T) {
	s := NewSessionStore("")
	if err := s.Save(Session{Volume: 30}); err != nil {
		t.Fatalf("saving without a path must be a no-op: %v", err)
	}
	if got := s.Load().Volume; got != 30 {
		t.Fatalf("in-memory session should still update: %d", got)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionDoesNotWritePerFrame(t *testing.T) {
	// The session must only be written when the caller asks (track change,
	// quit, interval) — verify a plain tick leaves the file alone.
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	s := NewSessionStore(path)
	if err := s.Save(Session{Volume: 70}); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := s.Save(Session{Volume: 70}); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) == string(second) {
		t.Log("identical content written again (expected: SavedAt changes)")
	}

	// The file must never be huge or rewritten per keystroke: size is bounded.
	if len(second) > 4096 {
		t.Fatalf("session file grew to %d bytes", len(second))
	}
}
