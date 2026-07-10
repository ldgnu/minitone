package events

import "testing"

func TestBusEmit(t *testing.T) {
	b := NewBus()
	var got any
	b.Subscribe(EventSongPlayed, func(e Event) {
		got = e.Data
	})
	b.Emit(EventSongPlayed, "hello")
	if got != "hello" {
		t.Fatalf("got %#v", got)
	}
}

func TestGlobal(t *testing.T) {
	if Global() == nil {
		t.Fatal()
	}
	if Global() != Global() {
		t.Fatal("singleton")
	}
}
