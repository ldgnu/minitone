package tui

type viewState int

const (
	viewConnecting viewState = iota
	viewNowPlaying
	viewArtists
	viewArtistAlbums
	viewAlbumSongs
	viewSearch
	viewPlaylists
	viewPlaylistSongs
	viewLyrics
)
