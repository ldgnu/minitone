package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

// Session is the small amount of state worth surviving a restart.
//
// It is intentionally tiny: one track, its position, the queue and the
// playback preferences. Nothing here is written per frame — the UI saves on
// song change, on quit, and every SaveInterval.
type Session struct {
	Song     models.Song   `json:"song,omitempty"`
	Position float64       `json:"position,omitempty"`
	Queue    []models.Song `json:"queue,omitempty"`
	// Cursor is the index into Queue of the track that was playing.
	Cursor int `json:"cursor,omitempty"`

	Volume  int  `json:"volume"`
	Shuffle bool `json:"shuffle,omitempty"`
	Repeat  int  `json:"repeat,omitempty"`

	SavedAt time.Time `json:"saved_at"`
}

// Empty reports whether the session holds nothing worth restoring.
func (s Session) Empty() bool {
	return len(s.Queue) == 0 && !s.HasSong()
}

// HasSong reports whether a last track was recorded.
func (s Session) HasSong() bool {
	return s.Song.Title != "" || s.Song.ID != "" || s.Song.FilePath != ""
}

// SessionStore persists Session to ~/.config/minitone/session.json.
type SessionStore struct {
	mu      sync.Mutex
	path    string
	current Session
	dirty   bool
}

func NewSessionStore(path string) *SessionStore {
	s := &SessionStore{path: path}
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var cur Session
			if json.Unmarshal(data, &cur) == nil {
				s.current = cur
			}
		}
	}
	return s
}

// DefaultSessionStore loads (or prepares) the standard session file.
func DefaultSessionStore() *SessionStore {
	p, err := SessionPath()
	if err != nil {
		return NewSessionStore("")
	}
	return NewSessionStore(p)
}

// Load returns the last stored session (zero value when there is none).
func (s *SessionStore) Load() Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// Save writes the session to disk, creating the directory if needed.
func (s *SessionStore) Save(sess Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess.SavedAt = time.Now()
	s.current = sess
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	s.dirty = false
	return os.WriteFile(s.path, data, 0o600)
}

// Clear removes the stored session file and resets in-memory state.
func (s *SessionStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = Session{}
	s.dirty = false
	if s.path == "" {
		return nil
	}
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Path returns the session file location ("" when there is none).
func (s *SessionStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// SessionPath returns ~/.config/minitone/session.json.
func SessionPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}
