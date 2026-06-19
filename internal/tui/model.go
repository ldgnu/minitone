package tui

import (
	"os"
	"strings"

	"github.com/ldgnu/amusic-cli/internal/cider"
	"github.com/ldgnu/amusic-cli/internal/music"
)

type Model struct {
	client *cider.Client

	state    viewState
	styles   Styles
	themeIdx int

	nowPlaying music.NowPlaying
	isPlaying  bool
	volume     int

	searchQuery    string
	searchResults  map[string][]music.SearchResult
	searchCursor   int
	searchCategory string

	detail music.SearchDetail

	queue         []music.Track
	queueCursor   int

	playlists      []music.Playlist
	playlistCursor int

	playlistTracks []music.Track
	ptCursor       int

	lyrics string

	err error

	width  int
	height int

	connected bool
	loading   bool
}

func NewModel(client *cider.Client) Model {
	themeName := strings.TrimSpace(os.Getenv("AMUSIC_THEME"))
	t, idx := themeByName(themeName)

	return Model{
		client:         client,
		state:          viewConnecting,
		styles:         NewStyles(t),
		themeIdx:       idx,
		searchCategory: "songs",
		volume:         70,
	}
}
