package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/subsonic"
)

func checkConnection(c *subsonic.Client) tea.Cmd {
	return func() tea.Msg {
		if err := c.Ping(); err != nil {
			return errMsg{err}
		}
		return connectionOKMsg{}
	}
}

func fetchArtists(c *subsonic.Client) tea.Cmd {
	return func() tea.Msg {
		artists, err := c.GetArtists()
		return artistsMsg{artists: artists, err: err}
	}
}

func fetchAlbums(c *subsonic.Client, artistID string) tea.Cmd {
	return func() tea.Msg {
		albums, err := c.GetArtist(artistID)
		return albumsMsg{albums: albums, err: err}
	}
}

func fetchAlbumSongs(c *subsonic.Client, albumID, title, artist string) tea.Cmd {
	return func() tea.Msg {
		songs, err := c.GetAlbum(albumID)
		return albumSongsMsg{songs: songs, title: title, artist: artist, err: err}
	}
}

func fetchSearch(c *subsonic.Client, query string) tea.Cmd {
	return func() tea.Msg {
		songs, err := c.Search(query, 20)
		return searchMsg{songs: songs, err: err}
	}
}

func fetchPlaylists(c *subsonic.Client) tea.Cmd {
	return func() tea.Msg {
		playlists, err := c.GetPlaylists()
		return playlistsMsg{playlists: playlists, err: err}
	}
}

func fetchPlaylistSongs(c *subsonic.Client, id string) tea.Cmd {
	return func() tea.Msg {
		songs, err := c.GetPlaylist(id)
		return playlistSongsMsg{songs: songs, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func eqTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return eqTickMsg{}
	})
}
