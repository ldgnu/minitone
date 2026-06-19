package tui

type viewState int

const (
	viewConnecting viewState = iota
	viewNowPlaying
	viewSearch
	viewSearchDetail
	viewQueue
	viewLibrary
	viewPlaylistTracks
	viewLyrics
)
