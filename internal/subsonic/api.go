package subsonic

import (
	"strconv"
)

func (c *Client) GetArtists() ([]Artist, error) {
	sr, err := c.get("getArtists", nil)
	if err != nil {
		return nil, err
	}

	artistsMap, _ := sr["artists"].(map[string]any)
	if artistsMap == nil {
		return nil, nil
	}
	idxList, _ := artistsMap["index"].([]any)

	var out []Artist
	for _, idx := range idxList {
		idxM, _ := idx.(map[string]any)
		if idxM == nil {
			continue
		}
		artists, _ := idxM["artist"].([]any)
		for _, a := range artists {
			am, _ := a.(map[string]any)
			if am == nil {
				continue
			}
			out = append(out, Artist{
				ID:   strVal(am, "id"),
				Name: strVal(am, "name"),
			})
		}
	}
	return out, nil
}

func (c *Client) GetArtist(id string) ([]Album, error) {
	sr, err := c.get("getArtist", map[string]string{"id": id})
	if err != nil {
		return nil, err
	}

	artist, _ := sr["artist"].(map[string]any)
	if artist == nil {
		return nil, nil
	}
	albums, _ := artist["album"].([]any)
	var out []Album
	for _, a := range albums {
		am, _ := a.(map[string]any)
		if am == nil {
			continue
		}
		out = append(out, Album{
			ID:        strVal(am, "id"),
			Name:      strVal(am, "name"),
			Artist:    strVal(am, "artist"),
			ArtistID:  strVal(am, "artistId"),
			Year:      intVal(am, "year"),
			SongCount: intVal(am, "songCount"),
			Duration:  intVal(am, "duration"),
		})
	}
	return out, nil
}

func (c *Client) GetAlbum(id string) ([]Song, error) {
	sr, err := c.get("getAlbum", map[string]string{"id": id})
	if err != nil {
		return nil, err
	}

	album, _ := sr["album"].(map[string]any)
	if album == nil {
		return nil, nil
	}
	songs, _ := album["song"].([]any)
	return parseSongs(songs), nil
}

func (c *Client) GetPlaylists() ([]Playlist, error) {
	sr, err := c.get("getPlaylists", nil)
	if err != nil {
		return nil, err
	}

	playlists, _ := sr["playlist"].([]any)
	var out []Playlist
	for _, p := range playlists {
		pm, _ := p.(map[string]any)
		if pm == nil {
			continue
		}
		out = append(out, Playlist{
			ID:        strVal(pm, "id"),
			Name:      strVal(pm, "name"),
			SongCount: intVal(pm, "songCount"),
			Duration:  intVal(pm, "duration"),
		})
	}
	return out, nil
}

func (c *Client) GetPlaylist(id string) ([]Song, error) {
	sr, err := c.get("getPlaylist", map[string]string{"id": id})
	if err != nil {
		return nil, err
	}

	songs, _ := sr["entry"].([]any)
	return parseSongs(songs), nil
}

func (c *Client) Search(query string, limit int) ([]Song, error) {
	if limit < 1 {
		limit = 20
	}
	sr, err := c.get("search3", map[string]string{
		"query": query,
		"songCount": strconv.Itoa(limit),
	})
	if err != nil {
		return nil, err
	}

	sr3, _ := sr["searchResult3"].(map[string]any)
	if sr3 == nil {
		return nil, nil
	}
	songs, _ := sr3["song"].([]any)
	return parseSongs(songs), nil
}

func (c *Client) GetRandomSongs(limit int) ([]Song, error) {
	if limit < 1 {
		limit = 20
	}
	sr, err := c.get("getRandomSongs", map[string]string{
		"size": strconv.Itoa(limit),
	})
	if err != nil {
		return nil, err
	}

	songs, _ := sr["song"].([]any)
	return parseSongs(songs), nil
}

func (c *Client) GetNowPlaying() ([]Song, error) {
	sr, err := c.get("getNowPlaying", nil)
	if err != nil {
		return nil, err
	}

	entries, _ := sr["entry"].([]any)
	return parseSongs(entries), nil
}

func (c *Client) ScrobbleNowPlaying(id string) error {
	return c.Scrobble(id, false)
}

func parseSongs(raw []any) []Song {
	var out []Song
	for _, s := range raw {
		sm, _ := s.(map[string]any)
		if sm == nil {
			continue
		}
		out = append(out, Song{
			ID:         strVal(sm, "id"),
			Title:      strVal(sm, "title"),
			Artist:     strVal(sm, "artist"),
			ArtistID:   strVal(sm, "artistId"),
			Album:      strVal(sm, "album"),
			AlbumID:    strVal(sm, "albumId"),
			Track:      intVal(sm, "track"),
			Year:       intVal(sm, "year"),
			Duration:   intVal(sm, "duration"),
			BitRate:    intVal(sm, "bitRate"),
			Genre:      strVal(sm, "genre"),
			Suffix:     strVal(sm, "suffix"),
			AlbumArtID: strVal(sm, "coverArt"),
		})
	}
	return out
}

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}
