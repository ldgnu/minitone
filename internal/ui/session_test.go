package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/store"
)

func sessionModel(t *testing.T, restore bool) (Model, *store.SessionStore) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.json")
	sess := store.NewSessionStore(path)
	m := New(Deps{
		Player:  newTestPlayer(),
		Session: sess,
		Restore: restore,
		Theme:   "terminal",
	})
	m.width, m.height = 100, 30
	return m, sess
}

func TestSessionRestoresQueueAndPrefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	written := store.NewSessionStore(path)

	if err := written.Save(store.Session{
		Song:    models.Song{ID: "yt:a", Title: "A", Source: models.SourceYouTube},
		Queue:   []models.Song{{ID: "yt:a", Title: "A"}, {ID: "yt:b", Title: "B"}},
		Cursor:  1,
		Volume:  42,
		Shuffle: true,
		Repeat:  1,
	}); err != nil {
		t.Fatal(err)
	}

	m := New(Deps{Session: store.NewSessionStore(path), Restore: true, Theme: "terminal"})
	m.width, m.height = 100, 30
	m.restoreSession()

	if m.queue.Len() != 2 {
		t.Fatalf("queue len %d", m.queue.Len())
	}
	if m.queue.Cursor() != 1 {
		t.Fatalf("cursor %d", m.queue.Cursor())
	}
	if !m.queue.Shuffle() {
		t.Fatal("shuffle must be restored")
	}
	if m.queue.Repeat() != queue.RepeatAll {
		t.Fatalf("repeat %v", m.queue.Repeat())
	}
	if m.lastSong.Title != "A" {
		t.Fatalf("last song %q", m.lastSong.Title)
	}
	if m.notice.text == "" {
		t.Fatal("the user must be told the session came back")
	}
}

func TestSessionQueuesLastSongEvenIfAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	written := store.NewSessionStore(path)
	// The last track was played straight from search, never queued.
	if err := written.Save(store.Session{
		Song:  models.Song{ID: "yt:z", Title: "Z"},
		Queue: []models.Song{{ID: "yt:a", Title: "A"}},
	}); err != nil {
		t.Fatal(err)
	}

	m := New(Deps{Session: store.NewSessionStore(path), Restore: true, Theme: "terminal"})
	m.width, m.height = 100, 30
	m.restoreSession()

	if m.queue.Len() != 2 {
		t.Fatalf("the last song should be queued too: %d", m.queue.Len())
	}
	if cur := m.queue.Current(); cur == nil || cur.Song.Title != "Z" {
		t.Fatalf("current %+v", cur)
	}
}

func TestSessionDoesNotStartPlayingOnRestore(t *testing.T) {
	m, _ := sessionModel(t, true)
	if err := m.session.Save(store.Session{
		Song: models.Song{ID: "yt:a", Title: "A"},
	}); err != nil {
		t.Fatal(err)
	}
	m2 := m
	m2.restoreSession()

	// Restoring is only about state: nothing must be playing yet.
	if m2.pendingPlay.Title != "" {
		t.Fatalf("restore must not start playback: %+v", m2.pendingPlay)
	}
}

func TestSessionClearDiscardsIt(t *testing.T) {
	m, sess := sessionModel(t, true)
	if err := sess.Save(store.Session{
		Song:  models.Song{ID: "yt:a", Title: "A"},
		Queue: []models.Song{{ID: "yt:a", Title: "A"}},
	}); err != nil {
		t.Fatal(err)
	}
	m.restoreSession()
	if m.queue.Len() == 0 {
		t.Fatal("nothing was restored")
	}

	m.clearSession()
	if m.queue.Len() != 0 {
		t.Fatalf("queue len %d", m.queue.Len())
	}
	if !strings.Contains(m.notice.text, "cleared") {
		t.Fatalf("notice %q", m.notice.text)
	}
	// The stored file must be gone too, so a restart does not bring it back.
	if p := sess.Path(); p != "" {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("the session file should be removed, got %v", err)
		}
	}
}

func TestSessionDisabledDoesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	written := store.NewSessionStore(path)
	if err := written.Save(store.Session{
		Queue: []models.Song{{ID: "yt:a", Title: "A"}},
	}); err != nil {
		t.Fatal(err)
	}

	m := New(Deps{Session: store.NewSessionStore(path), Restore: false, Theme: "terminal"})
	m.width, m.height = 100, 30
	m.restoreSession()

	if m.queue.Len() != 0 {
		t.Fatalf("restore must be skipped when disabled: %d", m.queue.Len())
	}
}

func TestSaveSessionCapturesPlayback(t *testing.T) {
	m, sess := sessionModel(t, true)
	m.player.SetPreview("Song", "Artist", "Album", "youtube", 30, 200, 55)
	m.queue.Add(models.Song{ID: "yt:a", Title: "A"})
	m.queue.Add(models.Song{ID: "yt:b", Title: "B"})
	m.queue.SetCursor(1)
	m.queue.SetShuffle(true)

	m.saveSession()

	got := sess.Load()
	if len(got.Queue) != 2 {
		t.Fatalf("queue %+v", got.Queue)
	}
	if got.Cursor != 1 {
		t.Fatalf("cursor %d", got.Cursor)
	}
	if got.Position != 30 {
		t.Fatalf("position %v", got.Position)
	}
	if got.Volume != 55 {
		t.Fatalf("volume %d", got.Volume)
	}
	if !got.Shuffle {
		t.Fatal("shuffle must be saved")
	}
	if got.Song.Title != "Song" {
		t.Fatalf("song %q", got.Song.Title)
	}
}

func TestSaveSessionIsSkippedWhenDisabled(t *testing.T) {
	m, sess := sessionModel(t, false)
	m.queue.Add(models.Song{ID: "yt:a", Title: "A"})
	m.saveSession()
	if len(sess.Load().Queue) != 0 {
		t.Fatal("nothing should be persisted when restore_session is off")
	}
}
