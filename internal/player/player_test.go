package player

import (
	"os/exec"
	"testing"
	"time"
)

func TestStateString(t *testing.T) {
	if StatePlaying.String() != "playing" {
		t.Fatal()
	}
	if StatePaused.String() != "paused" {
		t.Fatal()
	}
	if StateStopped.String() != "stopped" {
		t.Fatal()
	}
}

func TestNewDefaults(t *testing.T) {
	p := New()
	if p.Volume() != 70 {
		t.Fatalf("vol %d", p.Volume())
	}
	if p.Playing() {
		t.Fatal()
	}
	if p.SocketPath() == "" {
		t.Fatal()
	}
}

func TestStartAndClose(t *testing.T) {
	if _, err := exec.LookPath("mpv"); err != nil {
		t.Skip("mpv not installed")
	}
	p := New()
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	// Give observe a moment
	time.Sleep(100 * time.Millisecond)
	p.SetVolume(50)
	if p.Volume() != 50 {
		t.Fatalf("vol %d", p.Volume())
	}
	p.ToggleMute()
	if p.Volume() != 0 {
		t.Fatal("mute")
	}
	p.ToggleMute()
	if p.Volume() != 50 {
		t.Fatalf("unmute %d", p.Volume())
	}
	p.Close()
	// Double close must not panic
	p.Close()
}

func TestPlayEmptyURL(t *testing.T) {
	p := New()
	if err := p.Play("", "t", "a", "al", "x"); err == nil {
		t.Fatal("expected error")
	}
}
