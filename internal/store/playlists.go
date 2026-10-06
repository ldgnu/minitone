package store

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

// Playlist kinds.
const (
	PlaylistStatic = "static" // imported from a YouTube URL, keeps resolved IDs
	PlaylistTaste  = "taste"  // built from a search query (genre/artist)
	PlaylistWeekly = "weekly" // auto-refreshed discovery mix
)

// Playlist is one named list of YouTube-resolved songs.
type Playlist struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Kind        string        `json:"kind"`
	Query       string        `json:"query,omitempty"`      // taste/weekly: ytsearch query
	SourceURL   string        `json:"source_url,omitempty"` // static: original YouTube playlist URL
	Tracks      []models.Song `json:"tracks"`
	UpdatedAt   time.Time     `json:"updated_at"`
	AutoRefresh bool          `json:"auto_refresh"`
}

// Playlists persists named playlists in ~/.config/minitone/playlists.json.
type Playlists struct {
	mu      sync.RWMutex
	items   []Playlist
	byID    map[string]int
	path    string
	persist bool
}

func NewPlaylists(path string) *Playlists {
	p := &Playlists{path: path, byID: make(map[string]int), persist: path != ""}
	if path != "" {
		_ = p.Load()
	}
	return p
}

func DefaultPlaylists() *Playlists {
	p, err := PlaylistsPath()
	if err != nil {
		return NewPlaylists("")
	}
	return NewPlaylists(p)
}

func (p *Playlists) Load() error {
	if p.path == "" {
		return nil
	}
	data, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []Playlist
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	for i := range items {
		for j := range items[i].Tracks {
			items[i].Tracks[j] = items[i].Tracks[j].Sanitized()
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = items
	p.byID = make(map[string]int, len(items))
	for i, pl := range items {
		p.byID[pl.ID] = i
	}
	return nil
}

func (p *Playlists) saveLocked() error {
	if !p.persist || p.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(p.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path, data, 0o600)
}

func (p *Playlists) List() []Playlist {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]Playlist{}, p.items...)
}

func (p *Playlists) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.items)
}

func (p *Playlists) Get(id string) *Playlist {
	p.mu.RLock()
	defer p.mu.RUnlock()
	i, ok := p.byID[id]
	if !ok {
		return nil
	}
	pl := p.items[i]
	return &pl
}

func (p *Playlists) GetAt(i int) *Playlist {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if i < 0 || i >= len(p.items) {
		return nil
	}
	pl := p.items[i]
	return &pl
}

// Upsert inserts or replaces a playlist by ID.
func (p *Playlists) Upsert(pl Playlist) {
	if pl.ID == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if i, ok := p.byID[pl.ID]; ok {
		p.items[i] = pl
	} else {
		p.byID[pl.ID] = len(p.items)
		p.items = append(p.items, pl)
	}
	_ = p.saveLocked()
}

// SetTracks replaces the tracks of a playlist and bumps UpdatedAt.
func (p *Playlists) SetTracks(id string, tracks []models.Song) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	i, ok := p.byID[id]
	if !ok {
		return false
	}
	p.items[i].Tracks = tracks
	p.items[i].UpdatedAt = time.Now()
	_ = p.saveLocked()
	return true
}

func (p *Playlists) RemoveAt(i int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if i < 0 || i >= len(p.items) {
		return false
	}
	p.items = append(p.items[:i], p.items[i+1:]...)
	p.byID = make(map[string]int, len(p.items))
	for j, pl := range p.items {
		p.byID[pl.ID] = j
	}
	_ = p.saveLocked()
	return true
}

// Stale reports whether a playlist needs a weekly refresh.
func (p *Playlists) Stale(id string, maxAge time.Duration) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	i, ok := p.byID[id]
	if !ok {
		return false
	}
	pl := p.items[i]
	if !pl.AutoRefresh {
		return false
	}
	return time.Since(pl.UpdatedAt) > maxAge
}

// StaleIDs returns IDs of auto-refresh playlists older than maxAge.
func (p *Playlists) StaleIDs(maxAge time.Duration) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []string
	for _, pl := range p.items {
		if pl.AutoRefresh && time.Since(pl.UpdatedAt) > maxAge {
			out = append(out, pl.ID)
		}
	}
	return out
}
