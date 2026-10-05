package player

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAvailableAndNotInstalled(t *testing.T) {
	got := Available()
	_, err := exec.LookPath("mpv")
	if (err == nil) != got {
		t.Fatalf("Available()=%v but LookPath err=%v", got, err)
	}
	if !got {
		// ErrNotInstalled must be returned instead of a cryptic spawn error.
		p := New()
		if err := p.Start(); !errors.Is(err, ErrNotInstalled) {
			t.Fatalf("Start()=%v, want ErrNotInstalled", err)
		}
		p.Close()
	}
}

func TestStateStringsAndIcons(t *testing.T) {
	cases := []struct {
		s    State
		str  string
		icon string
	}{
		{StateStopped, "stopped", "■"},
		{StateLoading, "loading", "◌"},
		{StatePlaying, "playing", "▶"},
		{StatePaused, "paused", "❚❚"},
	}
	for _, c := range cases {
		if c.s.String() != c.str {
			t.Fatalf("%d.String()=%q want %q", c.s, c.s.String(), c.str)
		}
		if c.s.Icon() != c.icon {
			t.Fatalf("%d.Icon()=%q want %q", c.s, c.s.Icon(), c.icon)
		}
	}
}

// A stop we asked for must NOT look like "the song finished": otherwise
// pressing stop used to skip to the next queue item.
func TestEndFileStopDoesNotAdvance(t *testing.T) {
	p := New()
	defer p.Close()

	var ended, errored int
	var mu sync.Mutex
	p.OnEnded(func() { mu.Lock(); ended++; mu.Unlock() })
	p.OnError(func(error) { mu.Lock(); errored++; mu.Unlock() })

	p.handleEndFile("stop", "")
	mu.Lock()
	gotEnded, gotErr := ended, errored
	mu.Unlock()
	if gotEnded != 0 {
		t.Fatalf("end-file(stop) advanced the queue (%d times)", gotEnded)
	}
	if gotErr != 0 {
		t.Fatal("end-file(stop) reported an error")
	}
}

// Replacing a file makes mpv emit end-file(stop) for the previous track.
func TestEndFileReplaceDoesNotAdvance(t *testing.T) {
	p := New()
	defer p.Close()

	var ended int
	var mu sync.Mutex
	p.OnEnded(func() { mu.Lock(); ended++; mu.Unlock() })

	p.handleEndFile("redirect", "")
	mu.Lock()
	defer mu.Unlock()
	if ended != 0 {
		t.Fatal("end-file(redirect) advanced the queue")
	}
}

func TestEndFileEOFAdvances(t *testing.T) {
	p := New()
	defer p.Close()

	var ended int
	var mu sync.Mutex
	p.OnEnded(func() { mu.Lock(); ended++; mu.Unlock() })

	p.handleEndFile("eof", "")
	mu.Lock()
	defer mu.Unlock()
	if ended != 1 {
		t.Fatalf("end-file(eof) should advance once, got %d", ended)
	}
}

func TestEndFileErrorIsReported(t *testing.T) {
	p := New()
	defer p.Close()

	var got error
	var mu sync.Mutex
	p.OnError(func(e error) { mu.Lock(); got = e; mu.Unlock() })

	p.handleEndFile("error", "Failed to open")
	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("end-file(error) must surface an error")
	}
	if p.Status().State != StateStopped {
		t.Fatalf("state after failure = %v", p.Status().State)
	}
}

// A wedged mpv must not freeze whoever sends the command.
func TestSendCommandWhenNotConnected(t *testing.T) {
	p := New()
	defer p.Close()
	if err := p.sendCommand(map[string]any{"command": []any{"stop"}}); err == nil {
		t.Fatal("expected an error when mpv is not connected")
	}
	if p.Connected() {
		t.Fatal("Connected() must be false before Start")
	}
}

func TestPlayRejectsEmptyURL(t *testing.T) {
	p := New()
	defer p.Close()
	if err := p.Play("", "t", "a", "al", "youtube"); err == nil {
		t.Fatal("empty URL must be rejected")
	}
}

func TestStopIsIdempotent(t *testing.T) {
	p := New()
	defer p.Close()
	_ = p.Stop()
	_ = p.Stop()
	if p.Status().State != StateStopped {
		t.Fatal("state")
	}
	if !p.Status().Song.Empty() {
		t.Fatal("stop should clear the song")
	}
}

func TestCloseIsRaceFree(t *testing.T) {
	if !Available() {
		t.Skip("mpv not installed")
	}
	p := New()
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.VideoPlaying()
			_ = p.Status()
			p.StopVideo()
		}()
	}
	wg.Wait()
	p.Close()
	p.Close() // double close must not panic
}

// Play must report "loading" until mpv confirms, so the UI cannot show a
// track as playing while it is actually still starting (or already dead).
func TestPlayStartsInLoadingState(t *testing.T) {
	if !Available() {
		t.Skip("mpv not installed")
	}
	p := New()
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if err := p.Play("/nonexistent/track.mp3", "Ghost", "Nobody", "", "local"); err != nil {
		t.Fatal(err)
	}
	st := p.Status()
	if st.State != StateLoading {
		t.Fatalf("state right after Play = %v, want loading", st.State)
	}
	if st.Song.Title != "Ghost" {
		t.Fatalf("song metadata not published while loading: %+v", st.Song)
	}
}

// A track mpv cannot open must end in a reported error, not silence.
func TestUnplayableTrackReportsError(t *testing.T) {
	if !Available() {
		t.Skip("mpv not installed")
	}
	p := New()
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	errCh := make(chan error, 4)
	p.OnError(func(e error) {
		select {
		case errCh <- e:
		default:
		}
	})
	if err := p.Play("/nonexistent/track.mp3", "Ghost", "Nobody", "", "local"); err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-errCh:
		if e == nil {
			t.Fatal("nil error")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("mpv failed to open the track but reported nothing")
	}
}

func TestTogglePauseOnStoppedIsNoop(t *testing.T) {
	p := New()
	defer p.Close()
	if err := p.TogglePause(); err != nil {
		t.Fatal(err)
	}
	if p.Status().State != StateStopped {
		t.Fatalf("state = %v", p.Status().State)
	}
}

func TestVolumeClamped(t *testing.T) {
	p := New()
	defer p.Close()
	p.SetVolume(500)
	if p.Volume() != 100 {
		t.Fatalf("volume %d", p.Volume())
	}
	p.SetVolume(-20)
	if p.Volume() != 0 {
		t.Fatalf("volume %d", p.Volume())
	}
	p.ToggleMute()
	if p.Volume() != 100 {
		t.Fatalf("unmute restored %d, want previous 100", p.Volume())
	}
}

// The IPC socket must live in a directory we own and clean up completely.
func TestSocketCleanup(t *testing.T) {
	if !Available() {
		t.Skip("mpv not installed")
	}
	p := New()
	sock := p.SocketPath()

	dir := filepath.Dir(sock)
	if filepath.Base(dir) != "minitone-mpv" && !strings.HasPrefix(filepath.Base(dir), "minitone-mpv-") {
		t.Fatalf("socket should live in a private dir, got %q", sock)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("socket dir missing: %v", err)
	}

	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("mpv did not create the socket: %v", err)
	}

	p.Close()
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatalf("the socket survived Close: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the socket dir survived Close: %v", err)
	}
}
