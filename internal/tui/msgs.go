package tui

import "github.com/ldgnu/minitone/internal/subsonic"

type errMsg struct{ err error }
func (e errMsg) Error() string { return e.err.Error() }

type artistsMsg struct {
	artists []subsonic.Artist
	err     error
}

type albumsMsg struct {
	albums []subsonic.Album
	err    error
}

type albumSongsMsg struct {
	songs  []subsonic.Song
	title  string
	artist string
	err    error
}

type searchMsg struct {
	songs []subsonic.Song
	err   error
}

type playlistsMsg struct {
	playlists []subsonic.Playlist
	err       error
}

type playlistSongsMsg struct {
	songs []subsonic.Song
	err   error
}

type nowPlayingMsg struct{}
type connectionOKMsg struct{}
type tickMsg struct{}
type eqTickMsg struct{}
