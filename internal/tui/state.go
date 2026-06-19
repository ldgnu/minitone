// minitone - TUI for Apple Music via Cider
// by ldgnu <ldgnu@users.noreply.github.com>


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
