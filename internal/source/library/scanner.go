package library

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ldgnu/minitone/internal/models"
)

var audioExts = map[string]bool{
	".mp3": true, ".flac": true, ".ogg": true, ".m4a": true,
	".wav": true, ".aac": true, ".wma": true, ".opus": true,
	".alac": true, ".aiff": true, ".webm": true,
}

type Scanner struct {
	mu    sync.RWMutex
	songs []models.Song
	dirs  []string
}

func New() *Scanner {
	return &Scanner{}
}

func NewWithDirs(dirs []string) *Scanner {
	s := &Scanner{}
	for _, d := range dirs {
		s.AddDir(d)
	}
	return s
}

func (s *Scanner) AddDir(dir string) {
	if dir == "" {
		return
	}
	abs, err := filepath.Abs(dir)
	if err == nil {
		dir = abs
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.dirs {
		if d == dir {
			return
		}
	}
	s.dirs = append(s.dirs, dir)
}

func (s *Scanner) RemoveDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, d := range s.dirs {
		if d == dir {
			s.dirs = append(s.dirs[:i], s.dirs[i+1:]...)
			return
		}
	}
}

func (s *Scanner) Dirs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string{}, s.dirs...)
}

func (s *Scanner) Scan() error {
	s.mu.Lock()
	dirs := append([]string{}, s.dirs...)
	s.mu.Unlock()

	var mu sync.Mutex
	var found []models.Song
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			_ = filepath.Walk(d, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if fi.IsDir() {
					if strings.HasPrefix(fi.Name(), ".") && path != d {
						return filepath.SkipDir
					}
					return nil
				}
				ext := strings.ToLower(filepath.Ext(path))
				if !audioExts[ext] {
					return nil
				}

				song := models.Song{
					ID:       "local:" + path,
					Source:   models.SourceLocal,
					Title:    strings.TrimSuffix(fi.Name(), filepath.Ext(fi.Name())),
					Artist:   extractArtist(path),
					Album:    extractAlbum(path),
					FilePath: path,
					Format:   strings.TrimPrefix(ext, "."),
				}

				mu.Lock()
				found = append(found, song)
				mu.Unlock()
				return nil
			})
		}(dir)
	}
	wg.Wait()

	s.mu.Lock()
	s.songs = found
	s.mu.Unlock()
	return nil
}

func (s *Scanner) Search(query string, limit int) ([]models.Song, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	lq := strings.ToLower(query)

	var results []models.Song
	for _, song := range s.songs {
		if strings.Contains(strings.ToLower(song.Title), lq) ||
			strings.Contains(strings.ToLower(song.Artist), lq) ||
			strings.Contains(strings.ToLower(song.Album), lq) ||
			strings.Contains(strings.ToLower(song.FilePath), lq) {
			results = append(results, song)
		}
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (s *Scanner) All() []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Song{}, s.songs...)
}

func (s *Scanner) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.songs)
}

// Layout: .../Music/Artist/Album/track.ext or .../Artist/Album/track.ext
func extractArtist(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	for i, p := range parts {
		if strings.EqualFold(p, "Music") || strings.EqualFold(p, "Música") || strings.EqualFold(p, "music") {
			if i+1 < len(parts)-1 {
				return parts[i+1]
			}
		}
	}
	// Artist = parent of album directory.
	dir := filepath.Dir(path)       // album
	artistDir := filepath.Dir(dir)  // artist
	base := filepath.Base(artistDir)
	if base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return ""
}

func extractAlbum(path string) string {
	return filepath.Base(filepath.Dir(path))
}
