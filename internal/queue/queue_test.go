package queue

import (
	"testing"

	"github.com/ldgnu/minitone/internal/models"
)

func song(title string) models.Song {
	return models.Song{Title: title, ID: title}
}

func TestAddAndCurrent(t *testing.T) {
	q := New()
	if q.Current() != nil {
		t.Fatal("empty queue should have nil current")
	}
	q.Add(song("a"))
	q.Add(song("b"))
	cur := q.Current()
	if cur == nil || cur.Song.Title != "a" {
		t.Fatalf("expected a, got %+v", cur)
	}
	if q.Len() != 2 {
		t.Fatalf("len=%d", q.Len())
	}
}

func TestNextRepeatOff(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.Add(song("c"))

	// Current is a; Next advances to b
	n := q.Next()
	if n == nil || n.Song.Title != "b" {
		t.Fatalf("expected b, got %+v", n)
	}
	n = q.Next()
	if n == nil || n.Song.Title != "c" {
		t.Fatalf("expected c, got %+v", n)
	}
	n = q.Next()
	if n != nil {
		t.Fatalf("expected end of queue, got %+v", n)
	}
}

func TestNextRepeatAll(t *testing.T) {
	q := New()
	q.SetRepeat(RepeatAll)
	q.Add(song("a"))
	q.Add(song("b"))

	// Start at a, next -> b, next -> a
	if q.Current().Song.Title != "a" {
		t.Fatal("start a")
	}
	if q.Next().Song.Title != "b" {
		t.Fatal("next b")
	}
	if q.Next().Song.Title != "a" {
		t.Fatal("wrap a")
	}
}

func TestNextRepeatOne(t *testing.T) {
	q := New()
	q.SetRepeat(RepeatOne)
	q.Add(song("a"))
	q.Add(song("b"))
	for i := 0; i < 3; i++ {
		n := q.Next()
		if n == nil || n.Song.Title != "a" {
			t.Fatalf("repeat one should stay on a, got %+v", n)
		}
	}
}

func TestPrev(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.Add(song("c"))
	_ = q.Next() // b
	_ = q.Next() // c
	p := q.Prev()
	if p == nil || p.Song.Title != "b" {
		t.Fatalf("expected b, got %+v", p)
	}
}

func TestRemove(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.Add(song("c"))
	if !q.Remove(1) {
		t.Fatal("remove failed")
	}
	if q.Len() != 2 {
		t.Fatal("len")
	}
	titles := []string{}
	for _, it := range q.Items() {
		titles = append(titles, it.Song.Title)
	}
	if titles[0] != "a" || titles[1] != "c" {
		t.Fatalf("got %v", titles)
	}
}

func TestClear(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Clear()
	if q.Len() != 0 || q.Current() != nil {
		t.Fatal("clear failed")
	}
}

func TestShuffleKeepsCurrent(t *testing.T) {
	q := New()
	for _, t := range []string{"a", "b", "c", "d", "e"} {
		q.Add(song(t))
	}
	_ = q.Next() // b
	cur := q.Current().Song.Title
	q.SetShuffle(true)
	if q.Current() == nil || q.Current().Song.Title != cur {
		t.Fatalf("shuffle should keep current %s, got %+v", cur, q.Current())
	}
	if !q.Shuffle() {
		t.Fatal("shuffle flag")
	}
}

func TestAddNext(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("c"))
	q.AddNext(song("b"))
	items := q.Items()
	if len(items) != 3 {
		t.Fatalf("len %d", len(items))
	}
	// After current (a), next insert should be b then c
	if items[0].Song.Title != "a" || items[1].Song.Title != "b" || items[2].Song.Title != "c" {
		t.Fatalf("order %+v %+v %+v", items[0].Song.Title, items[1].Song.Title, items[2].Song.Title)
	}
}

func TestSetCursor(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.Add(song("c"))
	if !q.SetCursor(2) {
		t.Fatal("set cursor")
	}
	if q.Current().Song.Title != "c" {
		t.Fatal("cursor not c")
	}
}

func TestMove(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	q.Add(song("c"))
	if !q.Move(0, 2) {
		t.Fatal("move")
	}
	items := q.Items()
	if items[0].Song.Title != "b" || items[1].Song.Title != "c" || items[2].Song.Title != "a" {
		t.Fatalf("bad order")
	}
}

func TestPeekNext(t *testing.T) {
	q := New()
	q.Add(song("a"))
	q.Add(song("b"))
	pk := q.PeekNext()
	if pk == nil || pk.Song.Title != "b" {
		t.Fatalf("peek %+v", pk)
	}
	// Should not advance
	if q.Current().Song.Title != "a" {
		t.Fatal("peek advanced")
	}
}

func TestConcurrentAccess(t *testing.T) {
	q := New()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			q.Add(song("x"))
			_ = q.Next()
			_ = q.Items()
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		q.Add(song("y"))
		_ = q.Current()
		_ = q.Len()
	}
	<-done
}
