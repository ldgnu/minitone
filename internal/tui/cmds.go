// minitone - TUI for Apple Music via Cider
// by ldgnu <ldgnu@users.noreply.github.com>


package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/cider"
)

func checkConnection(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := client.NowPlaying()
		if err != nil {
			return errMsg{err}
		}
		return connectionOKMsg{}
	}
}

func fetchNowPlaying(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		np, err := client.NowPlaying()
		return nowPlayingMsg{np: np, err: err}
	}
}

func fetchIsPlaying(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		playing, err := client.IsPlaying()
		if err != nil {
			return isPlayingMsg{false}
		}
		return isPlayingMsg{playing}
	}
}

func fetchSearch(client *cider.Client, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := client.SearchAll(query, 10)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{results: results}
	}
}

func fetchDetail(client *cider.Client, kind, id string) tea.Cmd {
	return func() tea.Msg {
		detail, err := client.SearchDetail(kind, id)
		return searchDetailMsg{detail: detail, err: err}
	}
}

func fetchPlaylists(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		playlists, err := client.ListPlaylists()
		return playlistsMsg{playlists: playlists, err: err}
	}
}

func fetchPlaylistTracks(client *cider.Client, id string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := client.PlaylistTracks(id)
		return playlistTracksMsg{tracks: tracks, err: err}
	}
}

func fetchQueue(client *cider.Client) tea.Cmd {
	return func() tea.Msg {
		tracks, err := client.Queue()
		return queueMsg{tracks: tracks, err: err}
	}
}

func fetchLyrics(client *cider.Client, trackID string) tea.Cmd {
	return func() tea.Msg {
		text, err := client.Lyrics(trackID)
		if err != nil || text == "" {
			return lyricsMsg{err: err}
		}
		return lyricsMsg{text: text}
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
