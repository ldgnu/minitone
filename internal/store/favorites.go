package store

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

type FavEntry struct {
	Song      models.Song `json:"song"`
	AddedAt   time.Time   `json:"added_at"`
}

type Favorites struct {
	mu      sync.RWMutex
	items   []FavEntry
	byKey   map[string]int
	path    string
	persist bool
}

func NewFavorites(path string) *Favorites {
	f := &Favorites{
		path:    path,
		byKey:   make(map[string]int),
		persist: path != "",
	}
	if path != "" {
		_ = f.Load()
	}
	return f
}

// DefaultFavorites loads from the standard config path.
func DefaultFavorites() *Favorites {
	p, err := FavoritesPath()
	if err != nil {
		return NewFavorites("")
	}
	return NewFavorites(p)
}

func (f *Favorites) Load() error {
	if f.path == "" {
		return nil
	}
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []FavEntry
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = items
	f.byKey = make(map[string]int, len(items))
	for i, e := range items {
		f.byKey[e.Song.Key()] = i
	}
	return nil
}

func (f *Favorites) saveLocked() error {
	if !f.persist || f.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(f.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0o600)
}

func (f *Favorites) Add(song models.Song) bool {
	key := song.Key()
	if key == "" {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byKey[key]; ok {
		return false
	}
	f.byKey[key] = len(f.items)
	f.items = append(f.items, FavEntry{Song: song, AddedAt: time.Now()})
	_ = f.saveLocked()
	return true
}

func (f *Favorites) Remove(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	i, ok := f.byKey[key]
	if !ok {
		return false
	}
	f.items = append(f.items[:i], f.items[i+1:]...)
	// rebuild index
	f.byKey = make(map[string]int, len(f.items))
	for j, e := range f.items {
		f.byKey[e.Song.Key()] = j
	}
	_ = f.saveLocked()
	return true
}

func (f *Favorites) RemoveAt(index int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index < 0 || index >= len(f.items) {
		return false
	}
	f.items = append(f.items[:index], f.items[index+1:]...)
	f.byKey = make(map[string]int, len(f.items))
	for j, e := range f.items {
		f.byKey[e.Song.Key()] = j
	}
	_ = f.saveLocked()
	return true
}

func (f *Favorites) Toggle(song models.Song) (added bool) {
	key := song.Key()
	f.mu.RLock()
	_, exists := f.byKey[key]
	f.mu.RUnlock()
	if exists {
		f.Remove(key)
		return false
	}
	f.Add(song)
	return true
}

func (f *Favorites) Contains(song models.Song) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.byKey[song.Key()]
	return ok
}

func (f *Favorites) ContainsKey(key string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.byKey[key]
	return ok
}

func (f *Favorites) Items() []FavEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]FavEntry{}, f.items...)
}

func (f *Favorites) Songs() []models.Song {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]models.Song, len(f.items))
	for i, e := range f.items {
		out[i] = e.Song
	}
	return out
}

func (f *Favorites) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.items)
}

func (f *Favorites) Get(index int) *models.Song {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if index < 0 || index >= len(f.items) {
		return nil
	}
	s := f.items[index].Song
	return &s
}

// Search filters favorites by substring on title/artist/album.
func (f *Favorites) Search(query string, limit int) []models.Song {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []models.Song
	for _, e := range f.items {
		if q == "" ||
			strings.Contains(strings.ToLower(e.Song.Title), q) ||
			strings.Contains(strings.ToLower(e.Song.Artist), q) ||
			strings.Contains(strings.ToLower(e.Song.Album), q) {
			out = append(out, e.Song)
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}
