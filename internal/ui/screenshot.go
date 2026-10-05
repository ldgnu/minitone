package ui

import (
	"errors"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/search"
	"github.com/ldgnu/minitone/internal/source/library"
	"github.com/ldgnu/minitone/internal/store"
)

// Scenarios renders one static frame per UI state, so the README screenshots
// (and any manual check) exercise the real View() without touching mpv.
var Scenarios = []string{
	"welcome", "search", "playing", "queue", "favorites",
	"history", "library", "details", "help", "video",
	"searching", "error", "compact", "narrow",
}

// Screenshot renders a static frame of the UI for the given scenario.
//
// The "compact" and "narrow" scenarios force a small window, so they can be
// checked for overflow independently of the caller's size.
func Screenshot(scenario string, w, h int, theme string) string {
	switch scenario {
	case "compact":
		if w >= 64 {
			w, h = 64, 16
		}
	case "narrow":
		if w >= 40 {
			w, h = 40, 14
		}
	}

	p := player.New()
	q := queue.New()
	favs := store.NewFavorites("")
	hist := store.NewHistory("", store.DefaultHistoryMax)

	m := New(Deps{Player: p, Queue: q, Favs: favs, History: hist, Theme: theme})
	m.width = w
	m.height = h
	m.sources = []search.Source{
		{Name: search.SourceYouTube, Enabled: true},
		{Name: search.SourceRadio, Enabled: true},
		{Name: search.SourceNavidrome, Enabled: true},
		{Name: search.SourceLibrary, Enabled: true},
	}

	yt := func(title, artist string, dur, bit int) models.Song {
		return models.Song{
			ID: "yt:" + title, Source: models.SourceYouTube, SourceID: "abc",
			Title: title, Artist: artist, Duration: dur, Bitrate: bit,
			Thumbnail: "https://i.ytimg.com/vi/abc/hqdefault.jpg",
		}
	}
	radio := func(title, genre string) models.Song {
		return models.Song{
			ID:     "radio:" + title,
			Source: models.SourceRadio,
			Title:  title,
			Genre:  genre,
			URL:    "https://stream.example/" + title,
		}
	}
	local := func(title, artist, album string, dur int) models.Song {
		return models.Song{
			ID: "local:" + title, Source: models.SourceLocal,
			Title: title, Artist: artist, Album: album, Duration: dur,
			FilePath: "/home/you/Music/" + artist + "/" + album + "/" + title + ".flac",
			Format:   "flac",
		}
	}

	groups := func(query string, items ...models.Song) {
		m.search.query = query
		m.search.term = query
		m.search.bySource = map[string][]models.Song{}
		m.search.pending = map[string]bool{}
		m.search.failures = map[string]error{}
		bySource := map[string][]models.Song{}
		bySource[search.SourceYouTube] = items
		m.search.bySource = bySource
	}

	switch scenario {
	case "playing":
		p.SetPreview("Midnight City", "M83", "Hurry Up, We're Dreaming", "youtube", 102, 244, 72)
		q.Add(yt("Teardrop", "Massive Attack", 320, 320))
		q.Add(yt("Strobe", "deadmau5", 600, 256))
		q.Add(radio("Jazz Radio Berlin", "jazz"))
		q.Add(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		q.SetCursor(0)
		m.lastSong = yt("Midnight City", "M83", 244, 320)
		favs.Add(m.lastSong)

	case "search":
		groups("lofi",
			yt("lofi hip hop radio 📚 - beats to relax/study to", "Lofi Girl", 0, 0),
			yt("1 A.M Study Music - [lofi hip hop]", "Chillhop Music", 7200, 320),
			yt("lofi beats vol. 1", "kupla", 3540, 256),
			yt("sleepy.town (lofi mix)", "marshmallow", 4200, 192),
		)
		m.search.bySource[search.SourceRadio] = []models.Song{
			radio("Lofi Radio", "lofi"),
			radio("Jazz Radio Berlin", "jazz"),
		}
		m.focus = FocusList
		m.search.cursor = 1

	case "searching":
		m.search.query = "radiohead"
		m.search.term = "radiohead"
		m.search.running = true
		m.search.pending = map[string]bool{
			search.SourceYouTube: true, search.SourceRadio: true,
		}
		m.spinnerFrame = 3

	case "error":
		groups("jazz")
		m.search.failures[search.SourceRadio] = errOffline
		m.search.bySource[search.SourceRadio] = nil
		m.search.pending = map[string]bool{}
		m.notice = retryNotice("Radio unavailable", errOffline)

	case "queue":
		p.SetPreview("Teardrop", "Massive Attack", "", "youtube", 88, 320, 70)
		q.Add(yt("Teardrop", "Massive Attack", 320, 320))
		q.Add(yt("Angel", "Massive Attack", 280, 320))
		q.Add(yt("Strobe", "deadmau5", 600, 256))
		q.Add(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		q.Add(radio("Jazz Radio Berlin", "jazz"))
		q.SetCursor(2)
		m.panel = PanelQueue
		m.panelCursor = 2

	case "favorites":
		favs.Add(yt("Midnight City", "M83", 244, 320))
		favs.Add(yt("Teardrop", "Massive Attack", 320, 320))
		favs.Add(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		favs.Add(radio("Jazz Radio Berlin", "jazz"))
		m.panel = PanelFavorites

	case "history":
		hist.Push(yt("Midnight City", "M83", 244, 320))
		hist.Push(yt("Strobe", "deadmau5", 600, 256))
		hist.Push(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		hist.Push(radio("Lofi Radio", "lofi"))
		m.panel = PanelHistory
		m.panelCursor = 1

	case "library":
		m.lib.open = true
		m.lib.level = libSections
		m.lib.sections = []library.Group{
			{Name: "Artists", Count: 28},
			{Name: "Albums", Count: 31},
			{Name: "Folders", Count: 12},
			{Name: "Recently added", Count: 50},
			{Name: "Random", Count: 50},
		}
		m.lib.status = "412 tracks · scanned 3/3 folders"
		m.lib.cursor = 1
		m.panel = PanelLibrary

	case "details":
		m.details = yt("lofi hip hop radio 📚 - beats to relax/study to", "Lofi Girl", 0, 0)
		m.details.Bitrate = 128
		m.details.Format = "webm"
		m.panel = PanelDetails

	case "help":
		m.panel = PanelHelp

	case "video":
		m.videoMode = true
		m.lastSong = yt("lofi hip hop radio 📚", "Lofi Girl", 0, 0)
		p.SetVideoPreview("lofi hip hop radio 📚")
		m.notice = warnNotice("video mode on", "YouTube plays in an mpv window", "")

	case "compact":
		groups("lofi",
			yt("lofi hip hop radio 📚 - beats to relax/study to", "Lofi Girl", 0, 0),
			yt("1 A.M Study Music - [lofi hip hop]", "Chillhop Music", 7200, 320),
		)
		p.SetPreview("Midnight City", "M83", "", "youtube", 62, 244, 55)
		m.lastSong = yt("Midnight City", "M83", 244, 0)
		q.Add(m.lastSong)
		q.Add(yt("Teardrop", "Massive Attack", 320, 0))

	case "narrow":
		groups("dj")
		m.search.bySource[search.SourceYouTube] = []models.Song{
			yt("a very long youtube title that will not fit at all", "Some Channel", 3600, 320),
		}
		m.focus = FocusList
		p.SetPreview("Midnight City", "M83", "", "youtube", 62, 244, 55)
		m.lastSong = yt("Midnight City", "M83", 244, 0)

	default: // welcome
		m.search.query = ""
	}

	m.clampCursors()
	return m.View()
}

// errOffline is the error used by the "source unavailable" screenshot.
var errOffline = errors.New("all mirrors unreachable (dial tcp: timeout)")
