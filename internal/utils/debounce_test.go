package utils

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncerFiresOnce(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)
	var n atomic.Int32
	for i := 0; i < 5; i++ {
		d.Reset(func() { n.Add(1) })
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)
	if n.Load() != 1 {
		t.Fatalf("expected 1 fire, got %d", n.Load())
	}
}

func TestDebouncerCancel(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)
	var n atomic.Int32
	d.Reset(func() { n.Add(1) })
	d.Cancel()
	time.Sleep(80 * time.Millisecond)
	if n.Load() != 0 {
		t.Fatalf("cancelled should not fire")
	}
}
