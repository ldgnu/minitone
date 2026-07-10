package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/events"
)

type State int

const (
	StateStopped State = iota
	StatePlaying
	StatePaused
)

func (s State) String() string {
	switch s {
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	default:
		return "stopped"
	}
}

type SongInfo struct {
	Title  string
	Artist string
	Album  string
	Source string
	URL    string
}

type Status struct {
	State    State
	Song     SongInfo
	Elapsed  float64
	Duration float64
	Volume   int
	Bitrate  int
}

type Player struct {
	cmd     *exec.Cmd
	conn    net.Conn
	reader  *bufio.Reader
	mu      sync.Mutex
	status  Status
	socket  string
	done    chan struct{}
	closed  bool
	onEnded func()
	prevVol int
}

func New() *Player {
	return &Player{
		socket:  fmt.Sprintf("/tmp/minitone-mpv-%d.sock", time.Now().UnixNano()),
		status:  Status{Volume: 70},
		prevVol: 70,
		done:    make(chan struct{}),
	}
}

// OnEnded registers a callback invoked when the current file ends.
func (p *Player) OnEnded(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onEnded = fn
}

func (p *Player) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv not found in PATH (install mpv)")
	}

	p.cmd = exec.Command("mpv",
		"--no-video",
		"--no-terminal",
		"--quiet",
		fmt.Sprintf("--input-ipc-server=%s", p.socket),
		"--idle=yes",
		"--keep-open=no",
		"--volume=70",
	)

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("mpv start: %w", err)
	}

	// Wait for IPC socket (up to ~2.5s).
	var conn net.Conn
	var lastErr error
	for i := 0; i < 50; i++ {
		conn, lastErr = net.DialTimeout("unix", p.socket, 100*time.Millisecond)
		if lastErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if conn == nil {
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		return fmt.Errorf("mpv IPC socket not ready: %v", lastErr)
	}

	p.conn = conn
	p.reader = bufio.NewReader(conn)
	go p.readLoop()
	// Observe properties without holding the lock during send.
	go p.observe()

	return nil
}

func (p *Player) observe() {
	props := []struct {
		id   int
		name string
	}{
		{1, "playback-time"},
		{2, "duration"},
		{3, "volume"},
		{4, "media-title"},
		{5, "audio-bitrate"},
		{6, "pause"},
	}
	for _, prop := range props {
		_ = p.sendCommand(map[string]any{
			"command": []any{"observe_property", prop.id, prop.name},
		})
	}
}

func (p *Player) readLoop() {
	for {
		select {
		case <-p.done:
			return
		default:
		}

		p.mu.Lock()
		r := p.reader
		p.mu.Unlock()
		if r == nil {
			return
		}

		line, err := r.ReadString('\n')
		if err != nil {
			select {
			case <-p.done:
				return
			default:
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}

		var ev struct {
			Event string          `json:"event"`
			Name  string          `json:"name"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}

		switch ev.Event {
		case "property-change":
			p.handlePropertyChange(ev.Name, ev.Data)
		case "end-file":
			p.mu.Lock()
			p.status.State = StateStopped
			p.status.Elapsed = 0
			onEnded := p.onEnded
			p.mu.Unlock()
			events.Global().Emit(events.EventSongStopped, p.Status().Song)
			if onEnded != nil {
				onEnded()
			}
		case "playback-restart":
			p.mu.Lock()
			p.status.State = StatePlaying
			p.mu.Unlock()
		case "file-loaded":
			_ = p.sendCommand(map[string]any{
				"command": []any{"get_property", "duration"},
			})
		}
	}
}

func (p *Player) handlePropertyChange(name string, data json.RawMessage) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch name {
	case "playback-time":
		var v float64
		if json.Unmarshal(data, &v) == nil {
			p.status.Elapsed = v
		}
	case "duration":
		var v float64
		if json.Unmarshal(data, &v) == nil {
			p.status.Duration = v
		}
	case "volume":
		var v float64
		if json.Unmarshal(data, &v) == nil {
			p.status.Volume = int(v)
		}
	case "media-title":
		var v string
		if json.Unmarshal(data, &v) == nil && p.status.Song.Title == "" {
			p.status.Song.Title = v
		}
	case "audio-bitrate":
		var v float64
		if json.Unmarshal(data, &v) == nil {
			p.status.Bitrate = int(v)
		}
	case "pause":
		var v bool
		if json.Unmarshal(data, &v) == nil {
			if v {
				p.status.State = StatePaused
			} else if p.status.Song.URL != "" {
				p.status.State = StatePlaying
			}
		}
	}
}

func (p *Player) sendCommand(cmd any) error {
	p.mu.Lock()
	conn := p.conn
	closed := p.closed
	p.mu.Unlock()
	if closed || conn == nil {
		return fmt.Errorf("mpv not connected")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil {
		return fmt.Errorf("mpv not connected")
	}
	_, err = p.conn.Write(append(data, '\n'))
	return err
}

func (p *Player) Play(url, title, artist, album, source string) error {
	if url == "" {
		return fmt.Errorf("empty stream URL")
	}

	p.mu.Lock()
	p.status.Song = SongInfo{Title: title, Artist: artist, Album: album, Source: source, URL: url}
	p.status.Elapsed = 0
	p.status.Duration = 0
	p.status.Bitrate = 0
	p.status.State = StatePlaying
	song := p.status.Song
	p.mu.Unlock()

	// Unpause in case previous track was paused.
	_ = p.sendCommand(map[string]any{
		"command": []any{"set_property", "pause", false},
	})
	if err := p.sendCommand(map[string]any{
		"command": []any{"loadfile", url, "replace"},
	}); err != nil {
		return err
	}

	events.Global().Emit(events.EventSongPlayed, song)
	return nil
}

func (p *Player) Stop() error {
	p.mu.Lock()
	p.status.State = StateStopped
	p.status.Song = SongInfo{}
	p.status.Elapsed = 0
	p.status.Duration = 0
	p.mu.Unlock()

	events.Global().Emit(events.EventSongStopped, nil)
	return p.sendCommand(map[string]any{
		"command": []any{"stop"},
	})
}

func (p *Player) TogglePause() error {
	p.mu.Lock()
	currentState := p.status.State
	p.mu.Unlock()

	switch currentState {
	case StatePlaying:
		if err := p.sendCommand(map[string]any{
			"command": []any{"set_property", "pause", true},
		}); err != nil {
			return err
		}
		p.mu.Lock()
		p.status.State = StatePaused
		p.mu.Unlock()
		events.Global().Emit(events.EventSongPaused, nil)
	case StatePaused:
		if err := p.sendCommand(map[string]any{
			"command": []any{"set_property", "pause", false},
		}); err != nil {
			return err
		}
		p.mu.Lock()
		p.status.State = StatePlaying
		p.mu.Unlock()
		events.Global().Emit(events.EventSongResumed, nil)
	}
	return nil
}

func (p *Player) SetVolume(v int) {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	p.mu.Lock()
	if v > 0 {
		p.prevVol = v
	}
	p.status.Volume = v
	p.mu.Unlock()
	_ = p.sendCommand(map[string]any{
		"command": []any{"set_property", "volume", v},
	})
	events.Global().Emit(events.EventVolumeChanged, v)
}

// ToggleMute mutes or restores previous volume.
func (p *Player) ToggleMute() {
	p.mu.Lock()
	vol := p.status.Volume
	prev := p.prevVol
	p.mu.Unlock()
	if vol == 0 {
		if prev <= 0 {
			prev = 70
		}
		p.SetVolume(prev)
	} else {
		p.SetVolume(0)
	}
}

func (p *Player) Seek(sec float64) {
	_ = p.sendCommand(map[string]any{
		"command": []any{"seek", sec, "relative"},
	})
}

func (p *Player) Volume() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status.Volume
}

func (p *Player) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

func (p *Player) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status.State == StatePlaying
}

func (p *Player) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	close(p.done)

	_ = p.sendCommand(map[string]any{
		"command": []any{"quit"},
	})

	p.mu.Lock()
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
	cmd := p.cmd
	socket := p.socket
	p.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	_ = os.Remove(socket)
}

func (p *Player) Elapsed() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status.Elapsed
}

func (p *Player) Duration() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status.Duration
}

// SocketPath returns the IPC socket path (useful for tests).
func (p *Player) SocketPath() string {
	return p.socket
}
