package store

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

const DefaultHistoryMax = 200

type HistEntry struct {
	Song     models.Song `json:"song"`
	PlayedAt time.Time   `json:"played_at"`
}

type History struct {
	mu      sync.RWMutex
	items   []HistEntry // newest first
	max     int
	path    string
	persist bool
}

func NewHistory(path string, max int) *History {
	if max <= 0 {
		max = DefaultHistoryMax
	}
	h := &History{
		path:    path,
		max:     max,
		persist: path != "",
	}
	if path != "" {
		_ = h.Load()
	}
	return h
}

func DefaultHistory() *History {
	p, err := HistoryPath()
	if err != nil {
		return NewHistory("", DefaultHistoryMax)
	}
	return NewHistory(p, DefaultHistoryMax)
}

func (h *History) Load() error {
	if h.path == "" {
		return nil
	}
	data, err := os.ReadFile(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []HistEntry
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.items = items
	if len(h.items) > h.max {
		h.items = h.items[:h.max]
	}
	return nil
}

func (h *History) saveLocked() error {
	if !h.persist || h.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(h.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, data, 0o600)
}

// Push records a play. Dedupes consecutive identical keys; moves existing to front.
func (h *History) Push(song models.Song) {
	if song.Title == "" && song.ID == "" && song.URL == "" && song.FilePath == "" {
		return
	}
	key := song.Key()
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove existing same key.
	for i, e := range h.items {
		if e.Song.Key() == key {
			h.items = append(h.items[:i], h.items[i+1:]...)
			break
		}
	}
	h.items = append([]HistEntry{{Song: song, PlayedAt: time.Now()}}, h.items...)
	if len(h.items) > h.max {
		h.items = h.items[:h.max]
	}
	_ = h.saveLocked()
}

func (h *History) Items() []HistEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]HistEntry{}, h.items...)
}

func (h *History) Songs() []models.Song {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]models.Song, len(h.items))
	for i, e := range h.items {
		out[i] = e.Song
	}
	return out
}

func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.items)
}

func (h *History) Get(index int) *models.Song {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if index < 0 || index >= len(h.items) {
		return nil
	}
	s := h.items[index].Song
	return &s
}

func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.items = nil
	_ = h.saveLocked()
}

func (h *History) RemoveAt(index int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if index < 0 || index >= len(h.items) {
		return false
	}
	h.items = append(h.items[:index], h.items[index+1:]...)
	_ = h.saveLocked()
	return true
}
