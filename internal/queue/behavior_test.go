package queue

import (
	"testing"

	"github.com/ldgnu/minitone/internal/models"
)

func titles(q *Queue) []string {
	items := q.Items()
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Song.Title
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Regression: the old rebuildOrderLocked reshuffled on every single change,
// so adding a song scrambled the order of everything already queued.
func TestShuffleOrderIsStableAcrossAdd(t *testing.T) {
	q := New()
	q.SetShuffle(true)
	for _, s := range []string{"a", "b", "c", "d", "e", "f"} {
		q.Add(song(s))
	}

	before := make([]string, 0, len(q.Items()))
	for i := range q.Items() {
		before = append(before, q.playSeqTitle(i))
	}

	q.Add(song("g"))

	// The first six must still play in the same relative order, with "g" last.
	after := before
	got := make([]string, 0, len(before))
	for i := range q.Items() {
		if q.playSeqTitle(i) == "g" {
			continue
		}
		got = append(got, q.playSeqTitle(i))
	}
	if !eq(before, got) {
		t.Fatalf("adding a track reordered the queue:\nbefore %v\nafter  %v", before, got)
	}
	if q.playSeqTitle(len(q.Items())-1) != "g" {
		t.Fatalf("the new track should play last, order = %v", after)
	}
}

// playSeqTitle returns the title that plays at position i.
func (q *Queue) playSeqTitle(i int) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if i < 0 || i >= len(q.order) {
		return ""
	}
	for _, it := range q.items {
		if it.ID == q.order[i] {
			return it.Song.Title
		}
	}
	return ""
}

func TestShuffleOrderIsStableAcrossRemove(t *testing.T) {
	q := New()
	q.SetShuffle(true)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		q.Add(song(s))
	}
	before := playSeq(q)

	// Remove the very first one.
	q.Remove(0)

	after := playSeq(q)
	if len(after) != len(before)-1 {
		t.Fatalf("length %d vs %d", len(after), len(before))
	}
	// Everything except the removed track keeps its relative position.
	j := 0
	for i, title := range before {
		if i == 0 {
			continue
		}
		if after[j] != title {
			t.Fatalf("removing scrambled the order:\nbefore %v\nafter  %v", before, after)
		}
		j++
	}
}

// playSeq lists titles in play order.
func playSeq(q *Queue) []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]string, 0, len(q.order))
	for _, id := range q.order {
		for _, it := range q.items {
			if it.ID == id {
				out = append(out, it.Song.Title)
				break
			}
		}
	}
	return out
}

func TestShuffleOffKeepsListOrder(t *testing.T) {
	q := New()
	for _, s := range []string{"a", "b", "c"} {
		q.Add(song(s))
	}
	if got := playSeq(q); !eq(got, []string{"a", "b", "c"}) {
		t.Fatalf("play order %v", got)
	}
}

func TestMoveKeepsPlayingTrack(t *testing.T) {
	q := New()
	for _, s := range []string{"a", "b", "c", "d"} {
		q.Add(song(s))
	}
	q.SetCursor(1) // playing "b"

	if !q.Move(1, 3) {
		t.Fatal("Move should succeed")
	}
	if cur := q.Current(); cur == nil || cur.Song.Title != "b" {
		t.Fatalf("Move changed the playing track: %+v", cur)
	}
	if got := titles(q); !eq(got, []string{"a", "c", "d", "b"}) {
		t.Fatalf("list order %v", got)
	}
	if got := playSeq(q); !eq(got, []string{"a", "c", "d", "b"}) {
		t.Fatalf("play order %v", got)
	}
}

func TestMoveRejectsOutOfRange(t *testing.T) {
	q := New()
	q.Add(song("a"))
	if q.Move(0, 1) || q.Move(-1, 0) || q.Move(0, 0) || q.Move(5, 0) {
		t.Fatal("invalid moves must be rejected")
	}
}

func TestAddUnique(t *testing.T) {
	q := New()
	s := song("a")
	if !q.AddUnique(s) {
		t.Fatal("first add should succeed")
	}
	if q.AddUnique(s) {
		t.Fatal("duplicate must be rejected")
	}
	// A different song is fine.
	if !q.AddUnique(song("b")) {
		t.Fatal("different song should be added")
	}
	if q.Len() != 2 {
		t.Fatalf("len %d", q.Len())
	}
}

func TestAddNextPlaysNext(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("c"))
	q.SetCursor(0)

	q.AddNext(song("b"))
	// The new song is listed right after the current one...
	if got := titles(q); !eq(got, []string{"a", "b", "c"}) {
		t.Fatalf("list %v", got)
	}
	// ...and plays next.
	if n := q.PeekNext(); n == nil || n.Song.Title != "b" {
		t.Fatalf("PeekNext %+v", n)
	}
	if n := q.Next(); n == nil || n.Song.Title != "b" {
		t.Fatalf("Next %+v", n)
	}
}

func TestRemovePlayingKeepsPositionSane(t *testing.T) {
	q := New()
	for _, s := range []string{"a", "b", "c"} {
		q.Add(song(s))
	}
	q.SetCursor(1) // playing "b"
	if !q.Remove(1) {
		t.Fatal("remove failed")
	}
	if q.Len() != 2 {
		t.Fatalf("len %d", q.Len())
	}
	cur := q.Current()
	if cur == nil {
		t.Fatal("the queue should still have a current item")
	}
	// "c" now occupies that slot.
	if cur.Song.Title != "c" {
		t.Fatalf("current %q", cur.Song.Title)
	}
}

func TestRemoveLastEmptiesQueue(t *testing.T) {
	q := New()
	q.Add(song("a"))
	if !q.Remove(0) {
		t.Fatal("remove failed")
	}
	if q.Current() != nil || q.Cursor() != -1 {
		t.Fatal("an empty queue has no current item")
	}
	if q.Next() != nil || q.Prev() != nil || q.PeekNext() != nil {
		t.Fatal("navigation on an empty queue must be nil, not panic")
	}
}

func TestClearResetsEverything(t *testing.T) {
	q := New()
	q.SetShuffle(true)
	q.SetRepeat(RepeatOne)
	for _, s := range []string{"a", "b"} {
		q.Add(song(s))
	}
	q.Clear()
	if q.Len() != 0 || q.Cursor() != -1 || q.Next() != nil {
		t.Fatal("Clear must empty the queue")
	}
	// Preferences survive, they are not queue state.
	if !q.Shuffle() || q.Repeat() != RepeatOne {
		t.Fatal("Clear must keep shuffle/repeat preferences")
	}
	// And it must be reusable.
	q.Add(song("z"))
	if q.Current() == nil || q.Current().Song.Title != "z" {
		t.Fatal("the queue must work after Clear")
	}
}

func TestSnapshotAndRestore(t *testing.T) {
	q := New()
	for _, s := range []string{"a", "b", "c", "d"} {
		q.Add(song(s))
	}
	q.SetCursor(2)
	q.SetShuffle(true)
	q.SetRepeat(RepeatAll)

	songs, cursor, shuffle, repeat := q.Snapshot()
	if len(songs) != 4 || cursor != 2 || !shuffle || repeat != RepeatAll {
		t.Fatalf("snapshot: %d %v %v %v", len(songs), cursor, shuffle, repeat)
	}

	restored := New()
	restored.Restore(songs, cursor, shuffle, repeat)
	if restored.Len() != 4 {
		t.Fatalf("restored len %d", restored.Len())
	}
	if restored.Cursor() != 2 {
		t.Fatalf("restored cursor %d", restored.Cursor())
	}
	if !restored.Shuffle() || restored.Repeat() != RepeatAll {
		t.Fatal("preferences must be restored")
	}
	if cur := restored.Current(); cur == nil || cur.Song.Title != songs[2].Title {
		t.Fatalf("restored current %+v", cur)
	}
}

func TestRestoreReplacesPreviousContents(t *testing.T) {
	q := New()
	q.Add(song("old1"))
	q.Add(song("old2"))
	q.Restore([]models.Song{song("new")}, 0, false, RepeatOff)
	if q.Len() != 1 {
		t.Fatalf("Restore must replace, not append: len %d", q.Len())
	}
	if got := titles(q); !eq(got, []string{"new"}) {
		t.Fatalf("contents %v", got)
	}
}

func TestRestoreWithOutOfRangeCursor(t *testing.T) {
	q := New()
	q.Restore([]models.Song{song("a"), song("b")}, 99, false, RepeatOff)
	if q.Len() != 2 {
		t.Fatalf("len %d", q.Len())
	}
	// Must not panic and must stay usable.
	if n := q.Next(); n == nil {
		t.Fatal("navigation after restore should work")
	}
}

func TestRestoreEmpty(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Restore(nil, 0, false, RepeatOff)
	if q.Len() != 0 || q.Next() != nil {
		t.Fatal("restoring nothing must empty the queue")
	}
}

func TestSetShuffleKeepsPlayingTrack(t *testing.T) {
	q := New()
	for _, s := range []string{"a", "b", "c", "d", "e", "f"} {
		q.Add(song(s))
	}
	q.SetCursor(2)
	playing := q.Current().Song.Title

	q.SetShuffle(true)
	if cur := q.Current(); cur == nil || cur.Song.Title != playing {
		t.Fatalf("enabling shuffle changed the track: %+v", cur)
	}
	q.SetShuffle(false)
	if cur := q.Current(); cur == nil || cur.Song.Title != playing {
		t.Fatalf("disabling shuffle changed the track: %+v", cur)
	}
}

func TestRepeatOneRepeatsSameItem(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.SetCursor(0)
	q.SetRepeat(RepeatOne)
	for i := 0; i < 3; i++ {
		n := q.Next()
		if n == nil || n.Song.Title != "a" {
			t.Fatalf("repeat one played %+v", n)
		}
	}
}

func TestRepeatOffStopsAtEnd(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.SetCursor(1)
	if n := q.Next(); n != nil {
		t.Fatalf("should be past the end, got %+v", n)
	}
	// Once Next() ran off the end the position is -1, which means "stopped":
	// PeekNext then reports the first track, ready for the next Next().
	if cur := q.Current(); cur != nil {
		t.Fatalf("current after running off the end: %+v", cur)
	}
	if n := q.PeekNext(); n == nil || n.Song.Title != "a" {
		t.Fatalf("PeekNext from a stopped position should offer the first track, got %+v", n)
	}
}

func TestRepeatAllWraps(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.SetRepeat(RepeatAll)
	q.SetCursor(1)
	if n := q.Next(); n == nil || n.Song.Title != "a" {
		t.Fatalf("repeat all should wrap, got %+v", n)
	}
}

func TestPrevAtStart(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.SetCursor(0)
	if n := q.Prev(); n == nil || n.Song.Title != "a" {
		t.Fatalf("prev at the start should stay put, got %+v", n)
	}
}

func TestSetCursorRejectsOutOfRange(t *testing.T) {
	q := New()
	q.Add(song("a"))
	if q.SetCursor(-1) || q.SetCursor(5) {
		t.Fatal("out-of-range cursors must be rejected")
	}
	if !q.SetCursor(0) {
		t.Fatal("valid cursor should be accepted")
	}
}

func TestQueueConcurrentAccess(t *testing.T) {
	q := New()
	done := make(chan struct{})
	for i := 0; i < 4; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 200; j++ {
				q.Add(song("x"))
				q.Len()
				q.Cursor()
				q.Items()
				q.Shuffle()
				q.Repeat()
				q.Snapshot()
				_ = q.Next()
				_ = q.Current()
				q.Move(0, 1)
			}
		}(i)
	}
	for i := 0; i < 4; i++ {
		<-done
	}
}
