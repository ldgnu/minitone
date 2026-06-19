// minitone - TUI pa' controlar Apple Music desde Cider
// Creado por ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

package cider

import (
	"fmt"
	"strings"

	"github.com/ldgnu/minitone/internal/music"
)

func (c *Client) ListPlaylists() ([]music.Playlist, error) {
	path := "/v1/me/library/playlists?limit=100"
	out := make([]music.Playlist, 0, 64)

	for pages := 0; pages < 20; pages++ {
		root, err := c.runV3(path)
		if err != nil {
			return nil, err
		}
		items, next := parseData(root)
		for _, it := range items {
			name := strings.TrimSpace(extractString(it, "attributes", "name"))
			id := strings.TrimSpace(extractString(it, "id"))
			if name == "" {
				continue
			}
			out = append(out, music.Playlist{
				ID:   id,
				Name: name,
				URL:  strings.TrimSpace(extractString(it, "attributes", "url")),
			})
		}
		if next == "" {
			break
		}
		path = next
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no playlists found")
	}
	return out, nil
}

func (c *Client) PlaylistTracks(playlistID string) ([]music.Track, error) {
	id := strings.TrimSpace(playlistID)
	if id == "" {
		return nil, fmt.Errorf("missing playlist id")
	}

	path := "/v1/me/library/playlists/" + id + "/tracks?limit=100"
	tracks := make([]music.Track, 0, 128)

	for pages := 0; pages < 20; pages++ {
		root, err := c.runV3(path)
		if err != nil {
			return nil, err
		}
		items, next := parseData(root)
		for _, it := range items {
			title := strings.TrimSpace(extractString(it, "attributes", "name"))
			if title == "" {
				continue
			}
			tracks = append(tracks, music.Track{
				ID:         strings.TrimSpace(extractString(it, "id")),
				Title:      title,
				Artist:     strings.TrimSpace(extractString(it, "attributes", "artistName")),
				Album:      strings.TrimSpace(extractString(it, "attributes", "albumName")),
				URL:        strings.TrimSpace(extractString(it, "attributes", "url")),
				DurationMS: extractInt64(it, "attributes", "durationInMillis"),
			})
		}
		if next == "" {
			break
		}
		path = next
	}

	return tracks, nil
}
