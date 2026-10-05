package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture builds a small library:
//
//	root/Artist A/Album 1/a1.mp3
//	root/Artist A/Album 1/a2.flac
//	root/Artist B/Album 2/b1.mp3
//	root/cover.jpg  (ignored)
//	root/.hidden/x.mp3 (ignored)
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Artist A", "Album 1", "a1.mp3"))
	writeFile(t, filepath.Join(root, "Artist A", "Album 1", "a2.flac"))
	writeFile(t, filepath.Join(root, "Artist B", "Album 2", "b1.mp3"))
	writeFile(t, filepath.Join(root, "cover.jpg"))
	writeFile(t, filepath.Join(root, ".hidden", "x.mp3"))
	return root
}

func scan(t *testing.T, root string) *Scanner {
	t.Helper()
	s := NewWithDirs([]string{root})
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestScanIndexesAudioOnly(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)
	if s.Len() != 3 {
		t.Fatalf("indexed %d tracks, want 3", s.Len())
	}
	for _, song := range s.All() {
		ext := filepath.Ext(song.FilePath)
		if ext == ".jpg" {
			t.Fatal("a non-audio file was indexed")
		}
		if filepath.Base(filepath.Dir(filepath.Dir(song.FilePath))) == ".hidden" {
			t.Fatal("a hidden directory was indexed")
		}
	}
}

func TestScanExtractsArtistAndAlbum(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	byTitle := map[string]models.Song{}
	for _, song := range s.All() {
		byTitle[song.Title] = song
	}

	a1, ok := byTitle["a1"]
	if !ok {
		t.Fatalf("a1 missing: %+v", byTitle)
	}
	if a1.Artist != "Artist A" {
		t.Fatalf("artist %q", a1.Artist)
	}
	if a1.Album != "Album 1" {
		t.Fatalf("album %q", a1.Album)
	}
	if a1.Source != models.SourceLocal {
		t.Fatalf("source %q", a1.Source)
	}
	if a1.Format != "mp3" {
		t.Fatalf("format %q", a1.Format)
	}
	if a1.AddedAt == 0 {
		t.Fatal("AddedAt must be set from the file mtime")
	}
	if a1.Size == 0 {
		t.Fatal("Size must be set from the file")
	}
}

func TestScanReportsProgress(t *testing.T) {
	root := fixture(t)
	s := NewWithDirs([]string{root})

	var seen []ScanStatus
	if err := s.ScanWithContext(context.Background(), func(st ScanStatus) {
		seen = append(seen, st)
	}); err != nil {
		t.Fatal(err)
	}
	if len(seen) == 0 {
		t.Fatal("no progress callbacks")
	}
	if !seen[len(seen)-1].Running {
		// The final callback should mark the scan as finished.
		t.Logf("last status: %+v", seen[len(seen)-1])
	}
	if s.Scanning() {
		t.Fatal("Scanning() must be false after Scan returns")
	}
	if s.LastScan().IsZero() {
		t.Fatal("LastScan must be set after a successful scan")
	}
}

func TestScanCancellation(t *testing.T) {
	root := fixture(t)
	s := NewWithDirs([]string{root})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.ScanWithContext(ctx, nil); err == nil {
		t.Fatal("a cancelled scan should return the context error")
	}
	if s.Scanning() {
		t.Fatal("Scanning must be false after a cancelled scan")
	}
}

func TestScanIsNotStacked(t *testing.T) {
	root := fixture(t)
	s := NewWithDirs([]string{root})
	s.scanMu.Lock()
	s.scanning = true
	s.scanMu.Unlock()

	if err := s.Scan(); err != nil {
		t.Fatalf("a second scan should be a no-op, got %v", err)
	}
	if s.Len() != 0 {
		t.Fatal("a stacked scan must not index anything")
	}
}

func TestSearchContextCancelled(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.SearchContext(ctx, "a", 10); err == nil {
		t.Fatal("a cancelled search should return an error")
	}
}

func TestGroupsCountEverything(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	groups := s.Groups()
	want := []string{"Artists", "Albums", "Folders", "Recently added", "Random"}
	if len(groups) != len(want) {
		t.Fatalf("groups %+v", groups)
	}
	for i, name := range want {
		if groups[i].Name != name {
			t.Fatalf("group %d = %q, want %q", i, groups[i].Name, name)
		}
		if groups[i].Count == 0 {
			t.Fatalf("group %q has count 0", groups[i].Name)
		}
	}
	if groups[0].Count != 2 { // Artist A, Artist B
		t.Fatalf("artists %d", groups[0].Count)
	}
	if groups[1].Count != 2 { // Album 1, Album 2
		t.Fatalf("albums %d", groups[1].Count)
	}
}

func TestArtistsAlbumsFolders(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	artists := s.Artists()
	if len(artists) != 2 || artists[0] != "Artist A" {
		t.Fatalf("artists %v", artists)
	}
	albums := s.Albums()
	if len(albums) != 2 || albums[0] != "Album 1" {
		t.Fatalf("albums %v", albums)
	}
	folders := s.Folders()
	if len(folders) != 2 {
		t.Fatalf("folders %v", folders)
	}

	if got := s.SongsByArtist("Artist A"); len(got) != 2 {
		t.Fatalf("Artist A tracks %d", len(got))
	}
	if got := s.SongsByArtist("artist a"); len(got) != 2 {
		t.Fatal("artist lookup must be case-insensitive")
	}
	if got := s.SongsByAlbum("Album 2"); len(got) != 1 {
		t.Fatalf("Album 2 tracks %d", len(got))
	}
	if got := s.SongsByAlbum("nope"); len(got) != 0 {
		t.Fatalf("unknown album returned %d", len(got))
	}

	folder := folders[0]
	if got := s.SongsByFolder(folder); len(got) != 2 {
		t.Fatalf("folder %q tracks %d", folder, len(got))
	}
}

func TestRecentAndRandom(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	recent := s.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("recent %d", len(recent))
	}
	for i := 1; i < len(recent); i++ {
		if recent[i-1].AddedAt < recent[i].AddedAt {
			t.Fatal("recent must be newest first")
		}
	}

	r1 := s.Random(2)
	if len(r1) != 2 {
		t.Fatalf("random %d", len(r1))
	}
	// Deterministic per call, so the UI does not reshuffle every frame.
	r2 := s.Random(2)
	for i := range r1 {
		if r1[i].FilePath != r2[i].FilePath {
			t.Fatal("Random must be stable between calls")
		}
	}

	if got := s.Random(100); len(got) != 3 {
		t.Fatalf("random capped at the library size: %d", len(got))
	}
	if got := s.Recent(0); len(got) != 3 {
		t.Fatalf("recent with n=0 should return everything: %d", len(got))
	}
}

func TestMissingFilesAreDetectedAndDropped(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)
	if s.Len() != 3 {
		t.Fatalf("len %d", s.Len())
	}

	// Delete one file behind the scanner's back.
	victim := s.All()[0].FilePath
	if err := os.Remove(victim); err != nil {
		t.Fatal(err)
	}

	missing := s.Missing()
	if len(missing) != 1 {
		t.Fatalf("missing %d", len(missing))
	}
	if missing[0].FilePath != victim {
		t.Fatalf("wrong missing file %q", missing[0].FilePath)
	}
	if s.Len() != 2 {
		t.Fatalf("the index must drop the dead file, len %d", s.Len())
	}
	// Running again finds nothing more.
	if again := s.Missing(); len(again) != 0 {
		t.Fatalf("second pass reported %d", len(again))
	}
}

func TestPlayable(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	good := s.All()[0]
	if !s.Playable(good) {
		t.Fatalf("%q should be playable", good.FilePath)
	}
	dead := good
	dead.FilePath = filepath.Join(root, "gone.mp3")
	if s.Playable(dead) {
		t.Fatal("a missing file must not be playable")
	}
	noFile := models.Song{Title: "remote", URL: "https://x"}
	if s.Playable(noFile) {
		t.Fatal("a URL song has no local file to check")
	}
}

func TestForget(t *testing.T) {
	root := fixture(t)
	s := scan(t, root)

	victim := s.All()[0].FilePath
	if !s.Forget(victim) {
		t.Fatal("Forget should report success")
	}
	if s.Len() != 2 {
		t.Fatalf("len %d", s.Len())
	}
	if s.Forget("/nowhere") {
		t.Fatal("forgetting an unknown path must report false")
	}
}

func TestEmptyLibrary(t *testing.T) {
	empty := t.TempDir()
	s := scan(t, empty)
	if s.Len() != 0 {
		t.Fatalf("len %d", s.Len())
	}
	if len(s.All()) != 0 {
		t.Fatal("All must be empty")
	}
	res, err := s.Search("anything", 10)
	if err != nil || len(res) != 0 {
		t.Fatalf("search on an empty library: %v %+v", err, res)
	}
	if !s.HasLibrary() {
		t.Fatal("HasLibrary should be true: the folder exists, it is just empty")
	}
	if s.LastScan().IsZero() {
		t.Fatal("an empty library is still a successful scan")
	}
}

func TestNoDirsConfigured(t *testing.T) {
	s := New()
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 0 {
		t.Fatalf("len %d", s.Len())
	}
	if s.HasLibrary() {
		t.Fatal("HasLibrary must be false with no dirs")
	}
	if len(s.Dirs()) != 0 {
		t.Fatalf("dirs %v", s.Dirs())
	}
}

func TestMissingDirIsSkipped(t *testing.T) {
	root := fixture(t)
	s := NewWithDirs([]string{root, filepath.Join(root, "does-not-exist")})
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 3 {
		t.Fatalf("a missing root must be skipped, len %d", s.Len())
	}
	if len(s.ExistingDirs()) != 1 {
		t.Fatalf("existing dirs %v", s.ExistingDirs())
	}
}

func TestAddRemoveDir(t *testing.T) {
	s := New()
	s.AddDir("/tmp/a")
	s.AddDir("/tmp/a") // duplicate
	s.AddDir("/tmp/b")
	s.AddDir("") // ignored
	if len(s.Dirs()) != 2 {
		t.Fatalf("dirs %v", s.Dirs())
	}
	// Paths are made absolute.
	for _, d := range s.Dirs() {
		if !filepath.IsAbs(d) {
			t.Fatalf("dir %q is not absolute", d)
		}
	}
	s.RemoveDir("/tmp/a")
	if len(s.Dirs()) != 1 {
		t.Fatalf("dirs %v", s.Dirs())
	}
	s.RemoveDir("/nowhere") // must not panic
}

func TestStatusAndLastError(t *testing.T) {
	s := New()
	if s.Scanning() {
		t.Fatal("a fresh scanner is not scanning")
	}
	if s.LastError() != nil {
		t.Fatalf("last error %v", s.LastError())
	}
	st := s.Status()
	if st.Running || st.Found != 0 {
		t.Fatalf("status %+v", st)
	}
}

func TestLargeLibraryScanIsBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	root := t.TempDir()
	for i := 0; i < 120; i++ {
		writeFile(t, filepath.Join(root, "Artist", "Album", "track"+string(rune('a'+i%26))+".mp3"))
	}
	start := time.Now()
	s := scan(t, root)
	if s.Len() == 0 {
		t.Fatal("nothing indexed")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Fatalf("scan took %s", elapsed)
	}
}
