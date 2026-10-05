package player

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/events"
)

// ErrNotInstalled is returned when mpv cannot be found in PATH.
var ErrNotInstalled = errors.New("mpv not found in PATH (install mpv to play audio)")

// loadTimeout is how long we wait for mpv to report a file as loaded before
// declaring the track unplayable.
const loadTimeout = 25 * time.Second

// idleGracePeriod is how long we wait after mpv reports "idle" before
// concluding that the track we asked for could not be loaded.
const idleGracePeriod = 1500 * time.Millisecond

// loadConfirmDelay is how long after loadfile we ask mpv what really loaded.
const loadConfirmDelay = 1200 * time.Millisecond

type State int

const (
	StateStopped State = iota
	// StateLoading means a track was requested but mpv has not confirmed it
	// started yet. The UI shows it as "loading…" so the state never lies.
	StateLoading
	StatePlaying
	StatePaused
)

func (s State) String() string {
	switch s {
	case StateLoading:
		return "loading"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	default:
		return "stopped"
	}
}

// Icon returns a short glyph for the state, used by the player bar.
func (s State) Icon() string {
	switch s {
	case StateLoading:
		return "◌"
	case StatePlaying:
		return "▶"
	case StatePaused:
		return "❚❚"
	default:
		return "■"
	}
}

type SongInfo struct {
	Title  string
	Artist string
	Album  string
	Source string
	URL    string
}

func (s SongInfo) Empty() bool { return s.Title == "" && s.URL == "" }

type Status struct {
	State    State
	Song     SongInfo
	Elapsed  float64
	Duration float64
	Volume   int
	Bitrate  int
}

type Player struct {
	cmd    *exec.Cmd
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
	status Status
	socket string
	// sockDir holds the IPC socket. A private directory per player means a
	// crash leaves exactly one stray directory instead of a socket in /tmp
	// that nobody will ever clean up.
	sockDir string
	done    chan struct{}
	closed  bool
	onEnded func()
	onError func(error)
	prevVol int

	// gen increments on every Play/Stop so late IPC events belonging to a
	// previous track (or a stale load watchdog) can be ignored.
	gen         int64
	pendingSeek float64
	loadTimer   *time.Timer
	confirm     *time.Timer
	connErr     error

	// IPC command replies keyed by request_id (see query).
	reqID   int64
	pending map[int64]chan json.RawMessage

	// Separate mpv instance used for YouTube video playback (video mode).
	videoCmd     *exec.Cmd
	videoPlaying bool
	videoTitle   string
}

func New() *Player {
	return &Player{
		status:  Status{Volume: 70},
		prevVol: 70,
		done:    make(chan struct{}),
	}
}

// ensureSocket creates the private IPC directory and returns the socket path.
//
// It is lazy on purpose: a Player that is never started must not leave
// anything on disk (tests and the screenshot renderer create players without
// ever spawning mpv).
func (p *Player) ensureSocket() string {
	if p.socket != "" {
		return p.socket
	}
	dir, err := os.MkdirTemp("", "minitone-mpv-")
	if err != nil {
		// Fall back to the shared tmp: it still works, it just leaves a file.
		p.socket = fmt.Sprintf("/tmp/minitone-mpv-%d.sock", time.Now().UnixNano())
		return p.socket
	}
	p.sockDir = dir
	p.socket = filepath.Join(dir, "ipc.sock")
	return p.socket
}

// Available reports whether mpv is installed. It is safe to call before Start.
func Available() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

// OnEnded registers a callback invoked when the current file ends naturally.
func (p *Player) OnEnded(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onEnded = fn
}

// OnError registers a callback for playback failures (load errors, IPC loss).
func (p *Player) OnError(fn func(error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onError = fn
}

func (p *Player) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !Available() {
		return ErrNotInstalled
	}
	if p.cmd != nil {
		return nil // already running
	}
	sock := p.ensureSocket()

	vol := p.status.Volume
	if vol <= 0 {
		vol = 70
	}
	cmd := exec.Command("mpv",
		"--no-video",
		"--no-terminal",
		"--quiet",
		fmt.Sprintf("--input-ipc-server=%s", sock),
		"--idle=yes",
		"--keep-open=no",
		fmt.Sprintf("--volume=%d", vol),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mpv start: %w", err)
	}

	// Wait for the IPC socket to appear (up to ~2.5s).
	var conn net.Conn
	var lastErr error
	for i := 0; i < 50; i++ {
		conn, lastErr = net.DialTimeout("unix", sock, 100*time.Millisecond)
		if lastErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if conn == nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("mpv IPC socket not ready: %w", lastErr)
	}

	p.cmd = cmd
	p.conn = conn
	p.reader = bufio.NewReader(conn)
	go p.readLoop()
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
		{7, "idle-active"},
		{8, "path"},
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
			}
			// The pipe is gone: mpv died or the socket closed. Report it
			// instead of spinning silently forever.
			p.fail(fmt.Errorf("mpv connection lost: %w", err))
			return
		}

		var ev struct {
			Event     string          `json:"event"`
			Name      string          `json:"name"`
			Reason    string          `json:"reason"`
			FileError string          `json:"file_error"`
			RequestID int64           `json:"request_id"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}

		// A command reply (we sent request_id): hand the data to whoever
		// is waiting for it and move on.
		if ev.Event == "" && ev.RequestID != 0 {
			p.deliverReply(ev.RequestID, ev.Data)
			continue
		}

		switch ev.Event {
		case "property-change":
			p.handlePropertyChange(ev.Name, ev.Data)
		case "end-file":
			// NOTE: mpv puts "reason" in the event envelope, not in "data".
			p.handleEndFile(ev.Reason, ev.FileError)
		case "playback-restart":
			p.setState(StatePlaying)
		case "file-loaded":
			p.onFileLoaded()
		case "start-file":
			// Playback is being (re)started for a new file.
		case "idle":
			p.handleIdle()
		case "error":
			p.fail(fmt.Errorf("mpv: %s", strings.TrimSpace(string(ev.Data))))
		}
	}
}

// handleEndFile reacts to mpv's end-file event.
//
// reason is one of:
//   - "eof"    the track finished → advance the queue
//   - "error"  the track could not be played → report it
//   - "stop"   WE asked to stop (or replaced the file) → not an end
//   - "redirect" playlist moved on → not an end
//
// Only "eof" may advance the queue. Treating every end-file as "the song
// finished" is what used to make `stop` skip to the next track.
func (p *Player) handleEndFile(reason, fileErr string) {
	p.mu.Lock()
	p.status.Elapsed = 0
	switch reason {
	case "eof":
		p.status.State = StateStopped
		onEnded := p.onEnded
		song := p.status.Song
		p.mu.Unlock()
		events.Global().Emit(events.EventSongStopped, song)
		if onEnded != nil {
			onEnded()
		}

	case "error":
		p.status.State = StateStopped
		msg := fileErr
		if msg == "" {
			msg = "unknown error"
		}
		onErr := p.onError
		p.mu.Unlock()
		err := fmt.Errorf("cannot play this track: %s", msg)
		events.Global().Emit(events.EventPlayerError, err)
		if onErr != nil {
			onErr(err)
		}

	default:
		// "stop" / "redirect": an intentional transition, not a failure.
		p.mu.Unlock()
	}
}

// handleIdle reacts to mpv going idle, which happens when nothing is loaded.
func (p *Player) handleIdle() {
	p.mu.Lock()
	switch p.status.State {
	case StateLoading:
		gen, title := p.gen, p.status.Song.Title
		p.status.State = StateStopped
		p.mu.Unlock()
		// A failed load emits no error event in every mpv build; confirm.
		p.idleCheck(gen, title)
	case StatePlaying, StatePaused:
		p.status.State = StateStopped
		p.status.Elapsed = 0
		p.mu.Unlock()
	default:
		p.mu.Unlock()
	}
}

func (p *Player) onFileLoaded() {
	p.mu.Lock()
	p.status.State = StatePlaying
	p.status.Elapsed = 0
	seek := p.pendingSeek
	p.pendingSeek = 0
	if p.loadTimer != nil {
		p.loadTimer.Stop()
		p.loadTimer = nil
	}
	p.mu.Unlock()

	if seek > 0 {
		p.SeekAbsolute(seek)
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
		if json.Unmarshal(data, &v) == nil && v != "" && p.status.Song.Title == "" {
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
			switch {
			case v && !p.status.Song.Empty():
				p.status.State = StatePaused
			case !v && !p.status.Song.Empty() && p.status.Duration != 0:
				p.status.State = StatePlaying
			}
		}
	case "idle-active":
		// Informational only: the authoritative signal is the `idle` event.
		var v bool
		if json.Unmarshal(data, &v) == nil && v && p.status.State == StateLoading {
			p.idleCheck(p.gen, p.status.Song.Title)
		}
	case "path":
		var v string
		if json.Unmarshal(data, &v) == nil && v != "" && p.status.Song.URL == "" {
			p.status.Song.URL = v
		}
	}
}

// idleCheck confirms that mpv really went idle because the requested track
// could not be loaded, rather than because we replaced a finished track.
//
// mpv emits idle-active=true briefly after every loadfile (the old file ends
// first), so a single observation is not enough: we wait a short grace period
// and only report a failure if we are *still* loading afterwards.
func (p *Player) idleCheck(gen int64, title string) {
	time.AfterFunc(idleGracePeriod, func() {
		p.mu.Lock()
		stillLoading := p.status.State == StateLoading && p.gen == gen && !p.closed
		p.mu.Unlock()
		if !stillLoading {
			return
		}
		p.fail(fmt.Errorf("mpv could not open %q — file or stream unavailable", title))
	})
}

func (p *Player) setState(s State) {
	p.mu.Lock()
	p.status.State = s
	p.mu.Unlock()
}

// fail records a fatal playback error and notifies the UI once.
func (p *Player) fail(err error) {
	p.mu.Lock()
	if p.closed || p.connErr != nil {
		p.mu.Unlock()
		return
	}
	p.connErr = err
	p.status.State = StateStopped
	onErr := p.onError
	p.mu.Unlock()

	events.Global().Emit(events.EventPlayerError, err)
	if onErr != nil {
		onErr(err)
	}
}

// Err returns the last fatal playback error, if any.
func (p *Player) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connErr
}

// query issues a get_property over IPC and waits for mpv's reply.
//
// This is how we confirm a load actually happened: when mpv cannot open a URL
// it emits *no* event at all (it just stays idle), so event-only observation
// silently hides the failure.
func (p *Player) query(name string) (json.RawMessage, error) {
	p.mu.Lock()
	if p.closed || p.conn == nil || p.connErr != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("mpv not connected")
	}
	p.reqID++
	id := p.reqID
	ch := make(chan json.RawMessage, 1)
	if p.pending == nil {
		p.pending = make(map[int64]chan json.RawMessage)
	}
	p.pending[id] = ch
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
	}()

	if err := p.sendCommand(map[string]any{
		"command":    []any{"get_property", name},
		"request_id": id,
	}); err != nil {
		return nil, err
	}

	select {
	case data := <-ch:
		return data, nil
	case <-time.After(2 * time.Second):
		return nil, fmt.Errorf("mpv did not answer get_property %s", name)
	case <-p.done:
		return nil, fmt.Errorf("player closed")
	}
}

func (p *Player) deliverReply(id int64, data json.RawMessage) {
	p.mu.Lock()
	ch := p.pending[id]
	p.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- data:
	default:
	}
}

// confirmLoad asks mpv whether the file we requested is really the one loaded.
// mpv answers get_property on the IPC socket, so a bad URL is detected even
// though mpv emits no event for it.
func (p *Player) confirmLoad(gen int64, wantPath, title string) {
	time.AfterFunc(loadConfirmDelay, func() {
		p.mu.Lock()
		stale := p.gen != gen || p.status.State != StateLoading || p.closed
		p.mu.Unlock()
		if stale {
			return
		}

		pathRaw, err := p.query("path")
		if err != nil {
			return // connection trouble is reported elsewhere
		}
		var gotPath string
		_ = json.Unmarshal(pathRaw, &gotPath)

		p.mu.Lock()
		stale = p.gen != gen || p.status.State != StateLoading || p.closed
		p.mu.Unlock()
		if stale {
			return
		}
		if gotPath == "" || gotPath != wantPath {
			p.fail(fmt.Errorf("mpv could not open %q — file or stream unavailable", title))
		}
	})
}

// Connected reports whether the IPC channel to mpv is usable.
func (p *Player) Connected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.closed && p.conn != nil && p.connErr == nil
}

func (p *Player) sendCommand(cmd any) error {
	p.mu.Lock()
	conn := p.conn
	closed := p.closed
	connErr := p.connErr
	p.mu.Unlock()
	if closed || conn == nil || connErr != nil {
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
	// A wedged mpv must never freeze the UI goroutine that sends the command.
	_ = p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = p.conn.Write(append(data, '\n'))
	return err
}

// Play loads a URL. The state becomes StateLoading until mpv confirms via
// file-loaded / playback-restart, so the UI never shows a stale "playing".
func (p *Player) Play(url, title, artist, album, source string) error {
	return p.play(url, title, artist, album, source, 0)
}

// PlayAt is Play with a starting position (used to resume a session).
func (p *Player) PlayAt(url, title, artist, album, source string, pos float64) error {
	return p.play(url, title, artist, album, source, pos)
}

func (p *Player) play(url, title, artist, album, source string, pos float64) error {
	if url == "" {
		return fmt.Errorf("empty stream URL")
	}

	p.mu.Lock()
	p.gen++
	gen := p.gen
	p.status.Song = SongInfo{Title: title, Artist: artist, Album: album, Source: source, URL: url}
	p.status.Elapsed = 0
	p.status.Duration = 0
	p.status.Bitrate = 0
	p.status.State = StateLoading
	p.pendingSeek = pos
	song := p.status.Song
	if p.loadTimer != nil {
		p.loadTimer.Stop()
	}
	if p.confirm != nil {
		p.confirm.Stop()
	}
	p.loadTimer = time.AfterFunc(loadTimeout, func() {
		p.mu.Lock()
		stale := p.gen != gen || p.status.State != StateLoading
		p.mu.Unlock()
		if stale {
			return
		}
		p.fail(fmt.Errorf("mpv did not start playing %q (timeout after %s)", title, loadTimeout))
	})
	p.mu.Unlock()

	// Unpause in case the previous track was paused.
	_ = p.sendCommand(map[string]any{
		"command": []any{"set_property", "pause", false},
	})
	if err := p.sendCommand(map[string]any{
		"command": []any{"loadfile", url, "replace"},
	}); err != nil {
		p.mu.Lock()
		if p.loadTimer != nil {
			p.loadTimer.Stop()
			p.loadTimer = nil
		}
		p.mu.Unlock()
		return err
	}

	// Ask mpv afterwards what actually loaded, so a dead stream is reported.
	p.mu.Lock()
	if p.gen == gen {
		p.confirm = time.AfterFunc(loadConfirmDelay, func() { p.confirmLoad(gen, url, title) })
	}
	p.mu.Unlock()

	events.Global().Emit(events.EventSongPlayed, song)
	return nil
}

func (p *Player) Stop() error {
	p.mu.Lock()
	p.gen++
	if p.loadTimer != nil {
		p.loadTimer.Stop()
		p.loadTimer = nil
	}
	p.status.State = StateStopped
	p.status.Song = SongInfo{}
	p.status.Elapsed = 0
	p.status.Duration = 0
	p.pendingSeek = 0
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
		p.setState(StatePaused)
		events.Global().Emit(events.EventSongPaused, nil)
	case StatePaused:
		if err := p.sendCommand(map[string]any{
			"command": []any{"set_property", "pause", false},
		}); err != nil {
			return err
		}
		p.setState(StatePlaying)
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

// PlayVideo launches a standalone mpv window with video for the given URL
// (used for YouTube videos when video mode is enabled). It does not use the
// shared audio backend. The process is tracked so it can be stopped and its
// end is reported via the event bus.
func (p *Player) PlayVideo(url, title string) error {
	if url == "" {
		return fmt.Errorf("empty video URL")
	}

	cmd := exec.Command("mpv", "--no-terminal", "--quiet", "--title="+title, url)

	p.mu.Lock()
	old := p.videoCmd
	p.videoCmd = cmd
	p.videoTitle = title
	p.videoPlaying = true
	p.mu.Unlock()

	if old != nil && old.Process != nil {
		_ = old.Process.Kill()
	}

	if err := cmd.Start(); err != nil {
		p.mu.Lock()
		if p.videoCmd == cmd {
			p.videoPlaying = false
			p.videoTitle = ""
			p.videoCmd = nil
		}
		p.mu.Unlock()
		return err
	}

	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		if p.videoCmd == cmd {
			p.videoPlaying = false
			p.videoTitle = ""
			p.videoCmd = nil
		}
		p.mu.Unlock()
		events.Global().Emit(events.EventSongStopped, nil)
	}()
	return nil
}

// VideoPlaying reports whether a video mpv instance is running.
func (p *Player) VideoPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.videoPlaying
}

// VideoTitle returns the title of the currently playing video, if any.
func (p *Player) VideoTitle() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.videoTitle
}

// StopVideo kills the standalone video instance, if any.
func (p *Player) StopVideo() {
	p.mu.Lock()
	cmd := p.videoCmd
	p.videoCmd = nil
	p.videoPlaying = false
	p.videoTitle = ""
	p.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		go func() { _ = cmd.Wait() }()
	}
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

// Seek moves the playhead by sec seconds (negative to rewind).
func (p *Player) Seek(sec float64) {
	_ = p.sendCommand(map[string]any{
		"command": []any{"seek", sec, "relative"},
	})
}

// SeekAbsolute jumps to an absolute position in seconds.
func (p *Player) SeekAbsolute(sec float64) {
	if sec <= 0 {
		return
	}
	_ = p.sendCommand(map[string]any{
		"command": []any{"seek", sec, "absolute"},
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

// SetPreview sets a static status used only for rendering screenshots/previews.
func (p *Player) SetPreview(title, artist, album, source string, elapsed, duration float64, vol int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = Status{
		State:    StatePlaying,
		Song:     SongInfo{Title: title, Artist: artist, Album: album, Source: source},
		Elapsed:  elapsed,
		Duration: duration,
		Volume:   vol,
	}
}

// SetVideoPreview marks a video as playing (used for screenshot previews).
func (p *Player) SetVideoPreview(title string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.videoPlaying = true
	p.videoTitle = title
}

func (p *Player) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	if p.loadTimer != nil {
		p.loadTimer.Stop()
		p.loadTimer = nil
	}
	videoCmd := p.videoCmd
	p.videoCmd = nil
	p.videoPlaying = false
	p.videoTitle = ""
	p.mu.Unlock()

	// Stop any standalone video mpv instance.
	if videoCmd != nil && videoCmd.Process != nil {
		_ = videoCmd.Process.Kill()
		go func() { _ = videoCmd.Wait() }()
	}

	close(p.done)

	_ = p.sendCommand(map[string]any{
		"command": []any{"quit"},
	})

	p.mu.Lock()
	conn := p.conn
	p.conn = nil
	p.reader = nil
	cmd := p.cmd
	p.cmd = nil
	socket := p.socket
	sockDir := p.sockDir
	p.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
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
	// Remove the socket and, when we own a private directory, the whole thing.
	_ = os.Remove(socket)
	if sockDir != "" {
		_ = os.RemoveAll(sockDir)
	}
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

// SocketPath returns the IPC socket path, creating the private directory if
// needed. It is "" until then, so a player that was never started touches
// nothing on disk.
func (p *Player) SocketPath() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.socket == "" {
		// Pre-computed so tests can inspect it without starting mpv; Start()
		// reuses the same path.
		dir, err := os.MkdirTemp("", "minitone-mpv-")
		if err != nil {
			return ""
		}
		p.sockDir = dir
		p.socket = filepath.Join(dir, "ipc.sock")
	}
	return p.socket
}
