// minitone - TUI pa' controlar Apple Music desde Cider
// by ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

package cider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ldgnu/minitone/internal/music"
)

type nowPlayingResponse struct {
	Info struct {
		Name                string  `json:"name"`
		ArtistName          string  `json:"artistName"`
		AlbumName           string  `json:"albumName"`
		DurationInMillis    int64   `json:"durationInMillis"`
		CurrentPlaybackTime float64 `json:"currentPlaybackTime"`
		RemainingTime       float64 `json:"remainingTime"`
		ShuffleMode         int     `json:"shuffleMode"`
		RepeatMode          int     `json:"repeatMode"`
		PlayParams          struct {
			ID string `json:"id"`
		} `json:"playParams"`
		Artwork struct {
			URL string `json:"url"`
		} `json:"artwork"`
	} `json:"info"`
}

func (c *Client) NowPlaying() (music.NowPlaying, error) {
	body, err := c.doGET("/api/v1/playback/now-playing")
	if err != nil {
		return music.NowPlaying{}, err
	}

	var resp nowPlayingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return music.NowPlaying{}, fmt.Errorf("parse error: %w", err)
	}

	i := resp.Info
	return music.NowPlaying{
		TrackID:      strings.TrimSpace(i.PlayParams.ID),
		Track:        strings.TrimSpace(i.Name),
		Artist:       strings.TrimSpace(i.ArtistName),
		Album:        strings.TrimSpace(i.AlbumName),
		ArtworkURL:   normalizeArtworkURL(i.Artwork.URL),
		DurationMS:   i.DurationInMillis,
		CurrentSec:   i.CurrentPlaybackTime,
		RemainingSec: i.RemainingTime,
		ShuffleMode:  i.ShuffleMode,
		RepeatMode:   i.RepeatMode,
	}, nil
}

func (c *Client) TogglePlayPause() error {
	return c.doPOST("/api/v1/playback/playpause", nil)
}

func (c *Client) Next() error {
	return c.doPOST("/api/v1/playback/next", nil)
}

func (c *Client) Previous() error {
	return c.doPOST("/api/v1/playback/previous", nil)
}

func (c *Client) Stop() error {
	return c.doPOST("/api/v1/playback/stop", nil)
}

func (c *Client) IsPlaying() (bool, error) {
	body, err := c.doGET("/api/v1/playback/is-playing")
	if err != nil {
		return false, err
	}
	var obj struct {
		IsPlaying bool `json:"is_playing"`
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return false, fmt.Errorf("parse error: %w", err)
	}
	return obj.IsPlaying, nil
}

func (c *Client) SetVolume(percent int) error {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return c.doPOST("/api/v1/playback/volume", map[string]float64{"volume": float64(percent) / 100.0})
}

func (c *Client) ToggleShuffle() error {
	return c.doPOST("/api/v1/playback/toggle-shuffle", nil)
}

func (c *Client) ToggleRepeat() error {
	return c.doPOST("/api/v1/playback/toggle-repeat", nil)
}

func (c *Client) PlayItem(itemType, id string) error {
	t := strings.TrimSpace(itemType)
	if t == "" {
		t = "songs"
	}
	return c.doPOST("/api/v1/playback/play-item", map[string]string{
		"type": t,
		"id":   strings.TrimSpace(id),
	})
}

func (c *Client) PlayURL(rawURL string) error {
	return c.doPOST("/api/v1/playback/play-url", map[string]string{
		"url": strings.TrimSpace(rawURL),
	})
}

func (c *Client) PlayLater(itemType, id string) error {
	t := strings.TrimSpace(itemType)
	if t == "" {
		t = "songs"
	}
	return c.doPOST("/api/v1/playback/play-later", map[string]string{
		"type": t,
		"id":   strings.TrimSpace(id),
	})
}

func (c *Client) Queue() ([]music.Track, error) {
	body, err := c.doGET("/api/v1/playback/queue")
	if err != nil {
		return nil, err
	}

	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	out := make([]music.Track, 0, len(raw))
	for _, item := range raw {
		title := strings.TrimSpace(extractString(item, "attributes", "name"))
		if title == "" {
			continue
		}
		out = append(out, music.Track{
			ID:         strings.TrimSpace(extractString(item, "attributes", "playParams", "id")),
			Title:      title,
			Artist:     strings.TrimSpace(extractString(item, "attributes", "artistName")),
			Album:      strings.TrimSpace(extractString(item, "attributes", "albumName")),
			URL:        strings.TrimSpace(extractString(item, "attributes", "url")),
			DurationMS: extractInt64(item, "attributes", "durationInMillis"),
		})
	}
	return out, nil
}

func normalizeArtworkURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "{w}", "600")
	raw = strings.ReplaceAll(raw, "{h}", "600")
	raw = strings.ReplaceAll(raw, "{f}", "jpg")
	return raw
}
