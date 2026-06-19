package tui

import (
	"os"
	"strings"

	"github.com/ldgnu/minitone/internal/subsonic"
)

type Model struct {
	client *subsonic.Client
	player *subsonic.Player

	state    viewState
	styles   Styles
	themeIdx int

	currentSong *subsonic.Song
	isPlaying   bool
	volume      int

	searchQuery    string
	searchResults  []subsonic.Song
	searchCursor   int

	albumSongs    []subsonic.Song
	albumCursor   int
	albumTitle    string
	albumSubtitle string

	playlists      []subsonic.Playlist
	playlistCursor int

	playlistSongs []subsonic.Song
	plsongCursor  int

	artists      []subsonic.Artist
	artistCursor int

	artistAlbums []subsonic.Album
	aaCursor     int

	queue      []subsonic.Song
	queueIdx   int

	err error

	width  int
	height int

	connected bool
	loading   bool
	eqBars    [8]int
}

func NewModel(client *subsonic.Client, player *subsonic.Player) Model {
	themeName := strings.TrimSpace(os.Getenv("AMUSIC_THEME"))
	t, idx := themeByName(themeName)

	return Model{
		client: client,
		player: player,
		styles: NewStyles(t),
		themeIdx: idx,
		volume: 70,
	}
}
