package ui

import (
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/store"
)

// Screenshot renders a static frame of the UI for the given scenario so it can
// be captured to an image (see the --screenshot flag). It never touches mpv.
func Screenshot(scenario string, w, h int, theme string) string {
	p := player.New()
	q := queue.New()
	favs := store.NewFavorites("")
	hist := store.NewHistory("", store.DefaultHistoryMax)

	m := New(Deps{Player: p, Queue: q, Favs: favs, History: hist, Theme: theme})
	m.width = w
	m.height = h

	yt := func(title, artist string, dur, bit int) models.Song {
		return models.Song{Source: models.SourceYouTube, Title: title, Artist: artist, Duration: dur, Bitrate: bit}
	}
	radio := func(title, genre string) models.Song {
		return models.Song{Source: models.SourceRadio, Title: title, Genre: genre, Duration: 0}
	}
	local := func(title, artist, album string, dur int) models.Song {
		return models.Song{Source: models.SourceLocal, Title: title, Artist: artist, Album: album, Duration: dur}
	}

	switch scenario {
	case "playing":
		p.SetPreview("Midnight City", "M83", "Hurry Up, We're Dreaming", "youtube", 102, 244, 72)
		q.Add(yt("Teardrop", "Massive Attack", 320, 320))
		q.Add(yt("Strobe", "deadmau5", 600, 256))
		q.Add(radio("Jazz Radio", "jazz"))
		m.searchQuery = ""

	case "search":
		m.searchQuery = "lofi"
		m.searchGroup = 0
		m.searchCursor = 1
		m.searchResults = models.SearchResults{
			Query: "lofi",
			Groups: []models.SearchResultGroup{
				{Source: models.SourceYouTube, Name: "YouTube", Items: []models.Song{
					yt("lofi hip hop radio 📚 - beats to relax/study to", "Lofi Girl", 0, 0),
					yt("1 A.M Study Music - [lofi hip hop]", "Chillhop Music", 7200, 320),
					yt("lofi beats vol. 1", "kupla", 3540, 256),
					yt("sleepy.town (lofi mix)", "marshmallow", 4200, 192),
				}},
				{Source: models.SourceRadio, Name: "Radio", Items: []models.Song{
					radio("Lofi Radio", "lofi"),
					radio("Jazz Radio", "jazz"),
				}},
			},
		}

	case "favorites":
		favs.Add(yt("Midnight City", "M83", 244, 320))
		favs.Add(yt("Teardrop", "Massive Attack", 320, 320))
		favs.Add(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		favs.Add(radio("Jazz Radio", "jazz"))
		m.showPanel = PanelFavorites
		m.panelCursor = 0

	case "history":
		hist.Push(yt("Midnight City", "M83", 244, 320))
		hist.Push(yt("Strobe", "deadmau5", 600, 256))
		hist.Push(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		hist.Push(radio("Lofi Radio", "lofi"))
		m.showPanel = PanelHistory
		m.panelCursor = 1

	case "queue":
		p.SetPreview("Teardrop", "Massive Attack", "", "youtube", 88, 320, 70)
		q.Add(yt("Teardrop", "Massive Attack", 320, 320))
		q.Add(yt("Angel", "Massive Attack", 280, 320))
		q.Add(yt("Strobe", "deadmau5", 600, 256))
		q.Add(local("Redbone", "Childish Gambino", "Awaken, My Love!", 327))
		q.Add(radio("Jazz Radio", "jazz"))
		q.SetCursor(0)
		m.showPanel = PanelQueue
		m.panelCursor = 2

	case "video":
		m.videoMode = true
		m.searchQuery = "lofi girl"
		m.searchGroup = 0
		m.searchCursor = 0
		m.searchResults = models.SearchResults{
			Query: "lofi girl",
			Groups: []models.SearchResultGroup{
				{Source: models.SourceYouTube, Name: "YouTube", Items: []models.Song{
					yt("lofi hip hop radio 📚 - beats to relax/study to", "Lofi Girl", 0, 0),
					yt("synthwave radio 🌌 - beats to chill/game to", "Lofi Girl", 0, 0),
				}},
			},
		}
		p.SetVideoPreview("lofi hip hop radio 📚 - beats to relax/study to")

	case "help":
		m.showPanel = PanelHelp

	default: // welcome
		m.searchQuery = ""
	}

	return m.View()
}
