// minitone - TUI pa' controlar Apple Music desde Cider
// Creado por ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

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
