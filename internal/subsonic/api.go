package subsonic

import (
	"fmt"
	"strconv"
)

func (c *Client) GetArtists() ([]Artist, error) {
	sr, err := c.get("getArtists", nil)
	if err != nil {
		return nil, err
	}
	al, _ := sr["artists"].(map[string]any)
	idx, _ := al["index"].([]any)
	var artists []Artist
	for _, i := range idx {
		entry, _ := i.(map[string]any)
		arts, _ := entry["artist"].([]any)
		for _, a := range arts {
			ar, _ := a.(map[string]any)
			artists = append(artists, Artist{
				ID:   strVal(ar["id"]),
				Name: strVal(ar["name"]),
			})
		}
	}
	return artists, nil
}

func (c *Client) GetArtist(id string) ([]Album, error) {
	sr, err := c.get("getArtist", map[string]string{"id": id})
	if err != nil {
		return nil, err
	}
	// Subsonic wraps albums under artist.album
	if artist, ok := sr["artist"].(map[string]any); ok {
		if al, ok := artist["album"].([]any); ok {
			return parseAlbums(al), nil
		}
		// Single album object instead of array.
		if am, ok := artist["album"].(map[string]any); ok {
			return parseAlbums([]any{am}), nil
		}
	}
	if al, ok := sr["album"].([]any); ok {
		return parseAlbums(al), nil
	}
	return nil, nil
}

func (c *Client) GetAlbum(id string) ([]Song, error) {
	sr, err := c.get("getAlbum", map[string]string{"id": id})
	if err != nil {
		return nil, err
	}
	if album, ok := sr["album"].(map[string]any); ok {
		if ss, ok := album["song"].([]any); ok {
			return parseSongs(ss), nil
		}
		if sm, ok := album["song"].(map[string]any); ok {
			return parseSongs([]any{sm}), nil
		}
	}
	if ss, ok := sr["song"].([]any); ok {
		return parseSongs(ss), nil
	}
	return nil, nil
}

func (c *Client) GetPlaylists() ([]Playlist, error) {
	sr, err := c.get("getPlaylists", nil)
	if err != nil {
		return nil, err
	}
	pl, _ := sr["playlists"].(map[string]any)
	ps, _ := pl["playlist"].([]any)
	var out []Playlist
	for _, p := range ps {
		pm, _ := p.(map[string]any)
		out = append(out, Playlist{
			ID:        strVal(pm["id"]),
			Name:      strVal(pm["name"]),
			SongCount: intVal(pm["songCount"]),
			Duration:  intVal(pm["duration"]),
		})
	}
	return out, nil
}

func (c *Client) GetPlaylist(id string) ([]Song, error) {
	sr, err := c.get("getPlaylist", map[string]string{"id": id})
	if err != nil {
		return nil, err
	}
	ss, _ := sr["entry"].([]any)
	return parseSongs(ss), nil
}

func (c *Client) Search(query string, count int) ([]Song, error) {
	if count <= 0 {
		count = 20
	}
	// Prefer search3 (OpenSubsonic / modern Navidrome), fall back to search2.
	sr, err := c.get("search3", map[string]string{
		"query":        query,
		"songCount":    strconv.Itoa(count),
		"artistCount":  "0",
		"albumCount":   "0",
	})
	if err != nil {
		sr, err = c.get("search2", map[string]string{
			"query":       query,
			"songCount":   strconv.Itoa(count),
			"artistCount": "0",
			"albumCount":  "0",
		})
		if err != nil {
			return nil, err
		}
	}

	if sr3, ok := sr["searchResult3"].(map[string]any); ok {
		return parseSongField(sr3["song"]), nil
	}
	if sr2, ok := sr["searchResult2"].(map[string]any); ok {
		return parseSongField(sr2["song"]), nil
	}
	return nil, fmt.Errorf("empty result")
}

func parseSongField(v any) []Song {
	switch t := v.(type) {
	case []any:
		return parseSongs(t)
	case map[string]any:
		return parseSongs([]any{t})
	default:
		return nil
	}
}

func (c *Client) GetRandomSongs(count int) ([]Song, error) {
	sr, err := c.get("getRandomSongs", map[string]string{
		"size": strconv.Itoa(count),
	})
	if err != nil {
		return nil, err
	}
	ss, _ := sr["song"].([]any)
	return parseSongs(ss), nil
}

func (c *Client) ScrobbleNowPlaying(id string) {
	c.Scrobble(id, false)
}

func parseAlbums(raw []any) []Album {
	var out []Album
	for _, a := range raw {
		am, _ := a.(map[string]any)
		out = append(out, Album{
			ID:        strVal(am["id"]),
			Name:      strVal(am["name"]),
			Artist:    strVal(am["artist"]),
			ArtistID:  strVal(am["artistId"]),
			Year:      intVal(am["year"]),
			SongCount: intVal(am["songCount"]),
			Duration:  intVal(am["duration"]),
		})
	}
	return out
}

func parseSongs(raw []any) []Song {
	var out []Song
	for _, s := range raw {
		sm, _ := s.(map[string]any)
		out = append(out, Song{
			ID:         strVal(sm["id"]),
			Title:      strVal(sm["title"]),
			Artist:     strVal(sm["artist"]),
			ArtistID:   strVal(sm["artistId"]),
			Album:      strVal(sm["album"]),
			AlbumID:    strVal(sm["albumId"]),
			Track:      intVal(sm["track"]),
			Year:       intVal(sm["year"]),
			Duration:   intVal(sm["duration"]),
			BitRate:    intVal(sm["bitRate"]),
			Genre:      strVal(sm["genre"]),
			Size:       int64Val(sm["size"]),
			Suffix:     strVal(sm["suffix"]),
			Path:       strVal(sm["path"]),
			AlbumArtID: strVal(sm["coverArt"]),
		})
	}
	return out
}

func strVal(v any) string {
	s, _ := v.(string)
	return s
}

func intVal(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func int64Val(v any) int64 {
	f, _ := v.(float64)
	return int64(f)
}
