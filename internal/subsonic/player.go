package subsonic

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Player struct {
	client  *Client
	mpv     *exec.Cmd
	mpvLock sync.Mutex

	currentSong   *Song
	volume        int
	isPaused      bool
	playlist      []Song
	playlistIdx   int
	listenStart   time.Time

	OnSongChange func(song Song)
}

func NewPlayer(client *Client) *Player {
	p := &Player{
		client: client,
		volume: 70,
		playlistIdx: -1,
	}
	return p
}

func (p *Player) PlaySong(song Song) error {
	p.mpvLock.Lock()
	defer p.mpvLock.Unlock()

	p.stopMPV()

	p.currentSong = &song
	p.isPaused = false
	p.listenStart = time.Now()

	url := p.client.StreamURL(song.ID)

	p.mpv = exec.Command("mpv",
		"--no-video",
		"--quiet",
		"--no-terminal",
		"--volume="+strconv.Itoa(p.volume),
		url,
	)
	p.mpv.Stdout = os.Stderr
	p.mpv.Stderr = os.Stderr

	if err := p.mpv.Start(); err != nil {
		return fmt.Errorf("mpv error: %w", err)
	}

	go func() {
		p.mpv.Wait()
		p.mpvLock.Lock()
		p.mpv = nil
		p.mpvLock.Unlock()
	}()

	go func() {
		p.client.ScrobbleNowPlaying(song.ID)
	}()

	if p.OnSongChange != nil {
		p.OnSongChange(song)
	}

	return nil
}

func (p *Player) PlayPlaylist(songs []Song, startIdx int) {
	p.playlist = songs
	p.playlistIdx = startIdx
	if startIdx >= 0 && startIdx < len(songs) {
		p.PlaySong(songs[startIdx])
	}
}

func (p *Player) TogglePause() error {
	p.mpvLock.Lock()
	defer p.mpvLock.Unlock()

	if p.mpv == nil {
		return nil
	}

	if p.isPaused {
		p.mpv.Process.Signal(syscall.SIGCONT)
	} else {
		p.mpv.Process.Signal(syscall.SIGSTOP)
	}
	p.isPaused = !p.isPaused
	return nil
}

func (p *Player) Next() error {
	if len(p.playlist) == 0 || p.playlistIdx < 0 {
		return nil
	}
	if p.playlistIdx+1 >= len(p.playlist) {
		return nil
	}
	p.playlistIdx++
	return p.PlaySong(p.playlist[p.playlistIdx])
}

func (p *Player) Previous() error {
	if len(p.playlist) == 0 || p.playlistIdx <= 0 {
		return nil
	}
	p.playlistIdx--
	return p.PlaySong(p.playlist[p.playlistIdx])
}

func (p *Player) Stop() {
	p.mpvLock.Lock()
	defer p.mpvLock.Unlock()
	p.stopMPV()
	p.currentSong = nil
	p.isPaused = false
}

func (p *Player) stopMPV() {
	if p.mpv != nil && p.mpv.Process != nil {
		p.mpv.Process.Kill()
		p.mpv.Wait()
		p.mpv = nil
	}
}

func (p *Player) SetVolume(percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	p.volume = percent

	p.mpvLock.Lock()
	defer p.mpvLock.Unlock()
	if p.mpv != nil && p.mpv.Process != nil {
		p.mpv.Process.Signal(syscall.Signal(0))
	}
}

func (p *Player) GetVolume() int {
	return p.volume
}

func (p *Player) IsPlaying() bool {
	p.mpvLock.Lock()
	defer p.mpvLock.Unlock()
	return p.mpv != nil && !p.isPaused
}

func (p *Player) NowPlaying() *Song {
	p.mpvLock.Lock()
	defer p.mpvLock.Unlock()
	return p.currentSong
}

func (p *Player) Elapsed() int {
	if p.listenStart.IsZero() {
		return 0
	}
	elapsed := int(time.Since(p.listenStart).Seconds())
	if p.currentSong != nil && elapsed > p.currentSong.Duration {
		return p.currentSong.Duration
	}
	return elapsed
}

func (p *Player) FormatProgress() string {
	song := p.NowPlaying()
	if song == nil {
		return ""
	}
	elapsed := p.Elapsed()
	current := fmt.Sprintf("%d:%02d", elapsed/60, elapsed%60)
	total := fmt.Sprintf("%d:%02d", song.Duration/60, song.Duration%60)
	return current + " / " + total
}

func (p *Player) ProgressBar(width int) string {
	song := p.NowPlaying()
	if song == nil || song.Duration <= 0 {
		return strings.Repeat("░", width)
	}
	ratio := float64(p.Elapsed()) / float64(song.Duration)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func (p *Player) Close() {
	p.Stop()
}

func (p *Player) SetPlaylist(songs []Song, idx int) {
	p.playlist = songs
	p.playlistIdx = idx
}
