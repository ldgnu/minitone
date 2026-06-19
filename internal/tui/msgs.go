// minitone - TUI for Apple Music via Cider
// by ldgnu <ldgnu@users.noreply.github.com>


package tui

import (
	"github.com/ldgnu/minitone/internal/music"
)

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type nowPlayingMsg struct {
	np   music.NowPlaying
	err  error
}

type isPlayingMsg struct {
	playing bool
}

type searchResultsMsg struct {
	results map[string][]music.SearchResult
	err     error
}

type searchDetailMsg struct {
	detail music.SearchDetail
	err    error
}

type playlistsMsg struct {
	playlists []music.Playlist
	err       error
}

type playlistTracksMsg struct {
	tracks []music.Track
	err    error
}

type queueMsg struct {
	tracks []music.Track
	err    error
}

type lyricsMsg struct {
	text string
	err  error
}

type volumeMsg struct {
	percent int
}

type connectionOKMsg struct{}

type tickMsg struct{}

type eqTickMsg struct{}
