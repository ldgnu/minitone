package ui

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/source/library"
)

// The local library browser: Artists / Albums / Folders / Recently added /
// Random, as a three-level drill-down over the scanner index. No new subsystem
// and no new dependency — just an index we already build.

type libLevel int

const (
	libSections libLevel = iota // Artists / Albums / Folders / Recent / Random
	libNames                    // one artist, one album, one folder…
	libTracks                   // the tracks themselves
)

// libraryState is the local-library browser.
type libraryState struct {
	open     bool
	level    libLevel
	cursor   int // cursor in the current level
	scroll   int
	sections []library.Group
	names    []string
	items    []models.Song
	status   string
}

// reset returns to the section list.
func (l *libraryState) reset() {
	l.level = libSections
	l.cursor = 0
	l.scroll = 0
	l.names = nil
	l.items = nil
}

// len is the number of rows at the current level.
func (l libraryState) len() int {
	switch l.level {
	case libSections:
		return len(l.sections)
	case libNames:
		return len(l.names)
	default:
		return len(l.items)
	}
}

// enterLibrary drills down one level, or plays the highlighted track.
func (m *Model) enterLibrary() tea.Cmd {
	if m.libraryScanner == nil {
		m.notice = infoNotice("no library configured", "set library_paths in config.json", "")
		return nil
	}
	sc := m.libraryScanner
	lib := &m.lib

	switch lib.level {
	case libSections:
		if lib.cursor < 0 || lib.cursor >= len(lib.sections) {
			return nil
		}
		sec := lib.sections[lib.cursor]
		switch sec.Name {
		case "Artists":
			lib.names = sc.Artists()
			lib.level = libNames
		case "Albums":
			lib.names = sc.Albums()
			lib.level = libNames
		case "Folders":
			lib.names = sc.Folders()
			lib.level = libNames
		case "Recently added":
			lib.items = sc.Recent(200)
			lib.level = libTracks
		case "Random":
			lib.items = sc.Random(50)
			lib.level = libTracks
		default:
			return nil
		}
		lib.cursor = 0
		lib.scroll = 0
		if lib.len() == 0 {
			lib.reset()
			m.notice = infoNotice(sec.Name+" is empty", "add audio files to your library paths", "")
		}
		return nil

	case libNames:
		if lib.cursor < 0 || lib.cursor >= len(lib.names) {
			return nil
		}
		name := lib.names[lib.cursor]
		switch lib.sections[lib.cursor].Name {
		case "Artists":
			lib.items = sc.SongsByArtist(name)
		case "Albums":
			lib.items = sc.SongsByAlbum(name)
		case "Folders":
			lib.items = sc.SongsByFolder(name)
		}
		lib.level = libTracks
		lib.cursor = 0
		lib.scroll = 0
		return nil
	}

	// libTracks: play.
	if lib.cursor < 0 || lib.cursor >= len(lib.items) {
		return nil
	}
	song := lib.items[lib.cursor]
	if !sc.Playable(song) {
		m.notice = errorNotice("file missing", song.FilePath, "[d] forget  [esc] close")
		return nil
	}
	m.playToken++
	m.queue.Add(song)
	m.queue.SetCursor(m.queue.Len() - 1)
	m.panel = PanelNone
	m.lib.open = false
	m.focus = FocusList
	return m.resolveAndPlay(song)
}

// enqueueLibraryItems queues every track visible at the track level.
func (m *Model) enqueueLibraryItems() {
	if m.lib.level != libTracks || len(m.lib.items) == 0 {
		m.notice = infoNotice("nothing to enqueue", "press enter on a section first", "")
		return
	}
	n := 0
	for _, s := range m.lib.items {
		if m.queue.AddUnique(s) {
			n++
		}
	}
	if n == 0 {
		m.notice = okNotice("everything is already queued", "", "")
		return
	}
	m.notice = okNotice(fmt.Sprintf("queued %d tracks (%d total)", n, m.queue.Len()), "", "")
}

// forgetLibraryItem drops a missing file from the index so it stops showing up.
func (m *Model) forgetLibraryItem() {
	if m.lib.level != libTracks || m.lib.cursor < 0 || m.lib.cursor >= len(m.lib.items) {
		return
	}
	song := m.lib.items[m.lib.cursor]
	if m.libraryScanner == nil {
		return
	}
	m.libraryScanner.Forget(song.FilePath)
	m.lib.items = removeSong(m.lib.items, song)
	m.refreshLibrarySections()
	m.notice = okNotice("removed from library index", filepath.Base(song.FilePath), "")
}

func removeSong(songs []models.Song, target models.Song) []models.Song {
	out := songs[:0]
	for _, s := range songs {
		if s.FilePath == target.FilePath {
			continue
		}
		out = append(out, s)
	}
	return append([]models.Song{}, out...)
}

// libraryTitle is the heading of the library panel.
func (m Model) libraryTitle() string {
	switch m.lib.level {
	case libNames:
		if m.lib.cursor < len(m.lib.names) {
			return m.lib.names[m.lib.cursor]
		}
		return "—"
	case libTracks:
		return fmt.Sprintf("%d tracks", len(m.lib.items))
	default:
		return "Local library"
	}
}
