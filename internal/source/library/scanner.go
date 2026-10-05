package library

import (
	"context"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

// AudioExts lists the file extensions the scanner picks up.
var AudioExts = []string{
	".mp3", ".flac", ".ogg", ".oga", ".m4a", ".aac",
	".wav", ".wma", ".opus", ".alac", ".aiff", ".aif", ".webm", ".mka",
}

var audioExts = func() map[string]bool {
	m := make(map[string]bool, len(AudioExts))
	for _, e := range AudioExts {
		m[e] = true
	}
	return m
}()

// ScanStatus describes an in-flight or finished scan.
type ScanStatus struct {
	Running    bool
	Found      int
	Dirs       int
	Scanned    int
	CurrentDir string
	Err        error
}

type Scanner struct {
	mu      sync.RWMutex
	songs   []models.Song
	dirs    []string
	scanned time.Time
	lastErr error
	mtimes  map[string]time.Time // path -> mtime, to detect deleted files

	scanMu   sync.Mutex
	scanning bool
	progress ScanStatus
}

// New returns an empty scanner.
func New() *Scanner {
	return &Scanner{mtimes: map[string]time.Time{}}
}

// NewWithDirs returns a scanner for the given roots.
func NewWithDirs(dirs []string) *Scanner {
	s := New()
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

// ExistingDirs returns only the roots that are present on disk.
func (s *Scanner) ExistingDirs() []string {
	out := []string{}
	for _, d := range s.Dirs() {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

// HasLibrary reports whether at least one configured root exists.
func (s *Scanner) HasLibrary() bool { return len(s.ExistingDirs()) > 0 }

// Scan walks every configured root and replaces the index.
func (s *Scanner) Scan() error {
	return s.ScanWithContext(context.Background(), nil)
}

// ScanWithContext walks every root, reporting progress and honouring ctx.
//
// Progress reporting is what lets the UI show "scanning… 231/1200" instead of
// freezing with no feedback.
func (s *Scanner) ScanWithContext(ctx context.Context, onProgress func(ScanStatus)) error {
	s.scanMu.Lock()
	if s.scanning {
		s.scanMu.Unlock()
		return nil // already scanning; do not stack work
	}
	s.scanning = true
	s.progress = ScanStatus{Running: true, Dirs: len(s.Dirs())}
	s.scanMu.Unlock()

	finish := func(err error) {
		s.scanMu.Lock()
		s.scanning = false
		st := s.progress
		st.Running = false
		st.Err = err
		s.progress = st
		s.scanMu.Unlock()

		s.mu.Lock()
		s.lastErr = err
		if err == nil {
			s.scanned = time.Now()
		}
		s.mu.Unlock()

		if onProgress != nil {
			onProgress(st)
		}
	}

	s.update(func(st *ScanStatus) {
		st.Running = true
		st.Found = 0
		st.Scanned = 0
		st.CurrentDir = ""
	})

	dirs := s.Dirs()
	var mu sync.Mutex
	var found []models.Song
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	for _, dir := range dirs {
		if ctx.Err() != nil {
			break
		}
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s.update(func(st *ScanStatus) { st.CurrentDir = d })

			_ = filepath.Walk(d, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if ctx.Err() != nil {
					return filepath.SkipAll
				}
				if fi.IsDir() {
					if path != d && strings.HasPrefix(fi.Name(), ".") {
						return filepath.SkipDir
					}
					return nil
				}
				if !audioExts[strings.ToLower(filepath.Ext(path))] {
					return nil
				}

				song := songFor(path, fi)
				mu.Lock()
				found = append(found, song)
				n := len(found)
				mu.Unlock()

				// Throttle progress updates: one per file is far too chatty.
				if n%25 == 0 {
					s.update(func(st *ScanStatus) { st.Found = n })
					if onProgress != nil {
						onProgress(s.Status())
					}
				}
				return nil
			})

			s.update(func(st *ScanStatus) {
				st.Scanned++
				st.CurrentDir = ""
			})
		}(dir)
	}
	wg.Wait()

	sortSongs(found)

	s.mu.Lock()
	s.songs = found
	mt := make(map[string]time.Time, len(found))
	for _, song := range found {
		mt[song.FilePath] = time.Unix(song.AddedAt, 0)
	}
	s.mtimes = mt
	s.mu.Unlock()

	s.update(func(st *ScanStatus) { st.Found = len(found) })
	finish(ctx.Err())
	return ctx.Err()
}

func (s *Scanner) songForPath(path string) models.Song {
	fi, err := os.Stat(path)
	if err != nil {
		return songFor(path, nil)
	}
	return songFor(path, fi)
}

func songFor(path string, fi os.FileInfo) models.Song {
	name := filepath.Base(path)
	title := strings.TrimSuffix(name, filepath.Ext(name))
	song := models.Song{
		ID:       "local:" + path,
		Source:   models.SourceLocal,
		Title:    title,
		Artist:   extractArtist(path),
		Album:    extractAlbum(path),
		FilePath: path,
		Format:   strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
	}
	if fi != nil {
		song.AddedAt = fi.ModTime().Unix()
		song.Year = fi.ModTime().Year()
		song.Size = fi.Size()
	}
	return song
}

// update mutates the scan status under its own lock.
func (s *Scanner) update(fn func(*ScanStatus)) {
	s.scanMu.Lock()
	fn(&s.progress)
	s.scanMu.Unlock()
}

// Status returns a copy of the current scan status.
func (s *Scanner) Status() ScanStatus {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	return s.progress
}

// Scanning reports whether a scan is in flight.
func (s *Scanner) Scanning() bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	return s.scanning
}

// LastScan returns when the library was last indexed successfully.
func (s *Scanner) LastScan() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanned
}

// LastError returns the error of the last failed scan, if any.
func (s *Scanner) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastErr
}

// ── queries ────────────────────────────────────────────────────────────────

// Search filters the index by substring on title/artist/album/path.
func (s *Scanner) Search(query string, limit int) ([]models.Song, error) {
	return s.SearchContext(context.Background(), query, limit)
}

// SearchContext is Search with cancellation support.
func (s *Scanner) SearchContext(ctx context.Context, query string, limit int) ([]models.Song, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

// Group is a browse section of the local library.
type Group struct {
	Name  string
	Count int
}

// Groups returns the library sections shown in the browse panel.
func (s *Scanner) Groups() []Group {
	songs := s.All()
	artists := map[string]struct{}{}
	albums := map[string]struct{}{}
	folders := map[string]struct{}{}
	for _, song := range songs {
		if song.Artist != "" {
			artists[song.Artist] = struct{}{}
		}
		if song.Album != "" {
			albums[song.Album] = struct{}{}
		}
		if folder := folderOf(song.FilePath); folder != "" {
			folders[folder] = struct{}{}
		}
	}
	return []Group{
		{Name: "Artists", Count: len(artists)},
		{Name: "Albums", Count: len(albums)},
		{Name: "Folders", Count: len(folders)},
		{Name: "Recently added", Count: min(len(songs), 50)},
		{Name: "Random", Count: min(len(songs), 50)},
	}
}

// SongsByArtist returns every track of an artist, in album/title order.
func (s *Scanner) SongsByArtist(artist string) []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.Song
	for _, song := range s.songs {
		if strings.EqualFold(song.Artist, artist) {
			out = append(out, song)
		}
	}
	sortSongs(out)
	return out
}

// SongsByAlbum returns every track of an album.
func (s *Scanner) SongsByAlbum(album string) []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.Song
	for _, song := range s.songs {
		if strings.EqualFold(song.Album, album) {
			out = append(out, song)
		}
	}
	sortSongs(out)
	return out
}

// SongsByFolder returns the tracks directly inside a folder.
func (s *Scanner) SongsByFolder(folder string) []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.Song
	for _, song := range s.songs {
		if filepath.Dir(song.FilePath) == folder {
			out = append(out, song)
		}
	}
	sortSongs(out)
	return out
}

// Recent returns the n most recently added tracks.
func (s *Scanner) Recent(n int) []models.Song {
	songs := s.All()
	if n <= 0 {
		n = 50
	}
	sort.SliceStable(songs, func(i, j int) bool { return songs[i].AddedAt > songs[j].AddedAt })
	if len(songs) > n {
		songs = songs[:n]
	}
	return songs
}

// Random returns n random tracks (deterministic order per call).
func (s *Scanner) Random(n int) []models.Song {
	songs := s.All()
	total := len(songs)
	if n <= 0 {
		n = 50
	}
	if len(songs) > n {
		songs = songs[:n]
	}
	// Deterministic shuffle so the UI does not jump around between frames.
	// rand.Shuffle keeps panics out of an empty or single-element library.
	out := append([]models.Song{}, songs...)
	rand.New(rand.NewPCG(uint64(total)*2654435761, 1442695040888963407)).Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

// Artists returns every artist name, sorted.
func (s *Scanner) Artists() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := map[string]struct{}{}
	for _, song := range s.songs {
		if song.Artist != "" {
			set[song.Artist] = struct{}{}
		}
	}
	return sortedKeys(set)
}

// Albums returns every album name, sorted.
func (s *Scanner) Albums() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := map[string]struct{}{}
	for _, song := range s.songs {
		if song.Album != "" {
			set[song.Album] = struct{}{}
		}
	}
	return sortedKeys(set)
}

// Folders returns every folder containing audio, sorted.
func (s *Scanner) Folders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := map[string]struct{}{}
	for _, song := range s.songs {
		if d := folderOf(song.FilePath); d != "" {
			set[d] = struct{}{}
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := strings.ToLower(out[i]), strings.ToLower(out[j])
		if li != lj {
			return li < lj
		}
		return out[i] < out[j]
	})
	return out
}

// Forget drops a single path from the index (used when a file disappeared).
func (s *Scanner) Forget(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, song := range s.songs {
		if song.FilePath == path {
			s.songs = append(s.songs[:i], s.songs[i+1:]...)
			delete(s.mtimes, path)
			return true
		}
	}
	return false
}

// All returns every indexed track.
func (s *Scanner) All() []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Song{}, s.songs...)
}

// Len returns the number of indexed tracks.
func (s *Scanner) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.songs)
}

// Missing returns indexed songs whose file no longer exists on disk, and
// drops them from the index so the UI stops offering dead entries.
func (s *Scanner) Missing() []models.Song {
	s.mu.Lock()
	defer s.mu.Unlock()

	var missing []models.Song
	kept := s.songs[:0]
	for _, song := range s.songs {
		if _, err := os.Stat(song.FilePath); err != nil {
			missing = append(missing, song)
			delete(s.mtimes, song.FilePath)
			continue
		}
		kept = append(kept, song)
	}
	if len(missing) > 0 {
		s.songs = append([]models.Song{}, kept...)
	}
	return missing
}

// Playable reports whether a song has something mpv can open.
func (s *Scanner) Playable(song models.Song) bool {
	if song.FilePath == "" {
		return false
	}
	_, err := os.Stat(song.FilePath)
	return err == nil
}

func sortSongs(songs []models.Song) {
	sort.SliceStable(songs, func(i, j int) bool {
		ai, aj := strings.ToLower(songs[i].Artist), strings.ToLower(songs[j].Artist)
		if ai != aj {
			return ai < aj
		}
		li, lj := strings.ToLower(songs[i].Album), strings.ToLower(songs[j].Album)
		if li != lj {
			return li < lj
		}
		ti, tj := strings.ToLower(songs[i].Title), strings.ToLower(songs[j].Title)
		if ti != tj {
			return ti < tj
		}
		return songs[i].FilePath < songs[j].FilePath
	})
}

// folderOf returns the parent directory of a track.
func folderOf(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}

// Layout: .../Music/Artist/Album/track.ext or .../Artist/Album/track.ext
func extractArtist(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	for i, p := range parts {
		if strings.EqualFold(p, "Music") || strings.EqualFold(p, "Música") {
			if i+1 < len(parts)-1 {
				return parts[i+1]
			}
		}
	}
	// Artist = parent of the album directory.
	dir := filepath.Dir(path)
	artistDir := filepath.Dir(dir)
	base := filepath.Base(artistDir)
	if base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return ""
}

func extractAlbum(path string) string {
	return filepath.Base(filepath.Dir(path))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
