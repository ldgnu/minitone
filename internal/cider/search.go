// minitone - TUI pa' controlar Apple Music desde Cider
// by ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

package cider

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ldgnu/minitone/internal/music"
)

func (c *Client) SearchAll(query string, limit int) (map[string][]music.SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	out := map[string][]music.SearchResult{
		"songs":     {},
		"artists":   {},
		"albums":    {},
		"playlists": {},
	}

	storefront := ciderStorefront()
	path := "/v1/catalog/" + url.PathEscape(storefront) +
		"/search?term=" + url.QueryEscape(q) +
		"&types=songs,artists,albums,playlists" +
		"&limit=" + strconv.Itoa(limit)

	root, err := c.runV3(path)
	if err != nil {
		return nil, err
	}

	results, _ := root["data"].(map[string]any)
	if results == nil {
		results, _ = root["results"].(map[string]any)
	}
	if results == nil {
		return out, nil
	}

	for _, kind := range []string{"songs", "artists", "albums", "playlists"} {
		bucket, _ := results[kind].(map[string]any)
		if bucket == nil {
			continue
		}
		rows, _ := bucket["data"].([]any)
		for _, row := range rows {
			it, _ := row.(map[string]any)
			if it == nil {
				continue
			}
			id := strings.TrimSpace(extractString(it, "id"))
			title := strings.TrimSpace(extractString(it, "attributes", "name"))
			if id == "" || title == "" {
				continue
			}
			r := music.SearchResult{
				ID:         id,
				Type:       kind,
				Title:      title,
				Artist:     strings.TrimSpace(extractString(it, "attributes", "artistName")),
				Album:      strings.TrimSpace(extractString(it, "attributes", "albumName")),
				URL:        strings.TrimSpace(extractString(it, "attributes", "url")),
				DurationMS: extractInt64(it, "attributes", "durationInMillis"),
			}
			out[kind] = append(out[kind], r)
		}
	}

	return out, nil
}

func (c *Client) SearchDetail(kind, id string) (music.SearchDetail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return music.SearchDetail{}, fmt.Errorf("missing id")
	}

	sf := ciderStorefront()
	switch kind {
	case "songs":
		return music.SearchDetail{}, fmt.Errorf("use PlayItem for songs")
	case "artists":
		return c.fetchArtist(sf, id)
	case "albums":
		return c.fetchAlbum(sf, id)
	case "playlists":
		return c.fetchPlaylist(sf, id)
	default:
		return music.SearchDetail{}, fmt.Errorf("unsupported type: %s", kind)
	}
}

func (c *Client) fetchAlbum(storefront, id string) (music.SearchDetail, error) {
	path := "/v1/catalog/" + url.PathEscape(storefront) + "/albums/" + url.PathEscape(id)
	root, err := c.runV3(path)
	if err != nil {
		return music.SearchDetail{}, err
	}

	detail := music.SearchDetail{Type: "albums"}
	items, _ := parseData(root)
	if len(items) > 0 {
		it := items[0]
		detail.Title = extractString(it, "attributes", "name")
		detail.Subtitle = extractString(it, "attributes", "artistName")
		detail.Description = extractString(it, "attributes", "url")
	}

	tracks, _ := c.fetchTracks("/v1/catalog/" + url.PathEscape(storefront) + "/albums/" + url.PathEscape(id) + "/tracks?limit=100")
	detail.Tracks = tracks
	return detail, nil
}

func (c *Client) fetchPlaylist(storefront, id string) (music.SearchDetail, error) {
	path := "/v1/catalog/" + url.PathEscape(storefront) + "/playlists/" + url.PathEscape(id)
	root, err := c.runV3(path)
	if err != nil {
		return music.SearchDetail{}, err
	}

	detail := music.SearchDetail{Type: "playlists"}
	items, _ := parseData(root)
	if len(items) > 0 {
		it := items[0]
		detail.Title = extractString(it, "attributes", "name")
		detail.Subtitle = extractString(it, "attributes", "curatorName")
		detail.Description = extractString(it, "attributes", "url")
	}

	tracks, _ := c.fetchTracks("/v1/catalog/" + url.PathEscape(storefront) + "/playlists/" + url.PathEscape(id) + "/tracks?limit=100")
	detail.Tracks = tracks
	return detail, nil
}

func (c *Client) fetchArtist(storefront, id string) (music.SearchDetail, error) {
	path := "/v1/catalog/" + url.PathEscape(storefront) + "/artists/" + url.PathEscape(id)
	root, err := c.runV3(path)
	if err != nil {
		return music.SearchDetail{}, err
	}

	items, _ := parseData(root)
	if len(items) == 0 {
		return music.SearchDetail{}, fmt.Errorf("artist not found")
	}
	it := items[0]
	detail := music.SearchDetail{
		Type:    "artists",
		Title:   extractString(it, "attributes", "name"),
		Subtitle: extractString(it, "attributes", "genreNames", "0"),
	}

	tracks, _ := c.fetchTracks("/v1/catalog/" + url.PathEscape(storefront) + "/artists/" + url.PathEscape(id) + "/view/top-songs?limit=25")
	detail.Tracks = tracks
	return detail, nil
}

func (c *Client) fetchTracks(path string) ([]music.Track, error) {
	root, err := c.runV3(path)
	if err != nil {
		return nil, err
	}
	items, _ := parseData(root)

	out := make([]music.Track, 0, len(items))
	for _, it := range items {
		title := strings.TrimSpace(extractString(it, "attributes", "name"))
		if title == "" {
			continue
		}
		out = append(out, music.Track{
			ID:         strings.TrimSpace(extractString(it, "id")),
			Title:      title,
			Artist:     strings.TrimSpace(extractString(it, "attributes", "artistName")),
			Album:      strings.TrimSpace(extractString(it, "attributes", "albumName")),
			URL:        strings.TrimSpace(extractString(it, "attributes", "url")),
			DurationMS: extractInt64(it, "attributes", "durationInMillis"),
		})
	}
	return out, nil
}

func parseData(root map[string]any) ([]map[string]any, string) {
	if root == nil {
		return nil, ""
	}
	container := root
	if d, ok := root["data"].(map[string]any); ok {
		container = d
	}

	raw, _ := container["data"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	next, _ := container["next"].(string)
	return out, next
}
