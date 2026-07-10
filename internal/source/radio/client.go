package radio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

// Radio Browser public API mirrors (first that works wins).
var apiBases = []string{
	"https://de1.api.radio-browser.info",
	"https://nl1.api.radio-browser.info",
	"https://at1.api.radio-browser.info",
}

type Station struct {
	ID          string `json:"stationuuid"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	URLResolved string `json:"url_resolved"`
	Homepage    string `json:"homepage"`
	Tags        string `json:"tags"`
	Country     string `json:"country"`
	Language    string `json:"language"`
	Bitrate     int    `json:"bitrate"`
	Codec       string `json:"codec"`
	Votes       int    `json:"votes"`
	Favicon     string `json:"favicon"`
}

func Search(query string, limit int) ([]models.Song, error) {
	return SearchContext(context.Background(), query, limit)
}

func SearchContext(ctx context.Context, query string, limit int) ([]models.Song, error) {
	if limit <= 0 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error

	for _, base := range apiBases {
		stations, err := searchOne(ctx, client, base, query, limit)
		if err != nil {
			lastErr = err
			continue
		}
		return stationsToSongs(stations), nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("radio browser: no servers available")
}

func searchOne(ctx context.Context, client *http.Client, base, query string, limit int) ([]Station, error) {
	u := base + "/json/stations/byname/" + url.PathEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("hidebroken", "true")
	q.Set("order", "votes")
	q.Set("reverse", "true")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", "minitone/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("radio browser: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("radio browser: HTTP %d", resp.StatusCode)
	}

	var stations []Station
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, fmt.Errorf("radio browser decode: %w", err)
	}
	return stations, nil
}

func stationsToSongs(stations []Station) []models.Song {
	songs := make([]models.Song, 0, len(stations))
	for _, s := range stations {
		stream := s.URLResolved
		if stream == "" {
			stream = s.URL
		}
		if stream == "" {
			continue
		}
		tags := strings.Split(s.Tags, ",")
		genre := ""
		if len(tags) > 0 {
			genre = strings.TrimSpace(tags[0])
		}
		songs = append(songs, models.Song{
			ID:        "radio:" + s.ID,
			Source:    models.SourceRadio,
			SourceID:  s.ID,
			Title:     s.Name,
			Artist:    s.Country,
			URL:       stream,
			Bitrate:   s.Bitrate,
			Format:    s.Codec,
			Genre:     genre,
			Thumbnail: s.Favicon,
		})
	}
	return songs
}

func SearchByTag(tag string, limit int) ([]models.Song, error) {
	return SearchByTagContext(context.Background(), tag, limit)
}

func SearchByTagContext(ctx context.Context, tag string, limit int) ([]models.Song, error) {
	if limit <= 0 {
		limit = 10
	}
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error
	for _, base := range apiBases {
		u := base + "/json/stations/bytag/" + url.PathEscape(tag)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		q := req.URL.Query()
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("hidebroken", "true")
		req.URL.RawQuery = q.Encode()
		req.Header.Set("User-Agent", "minitone/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		var stations []Station
		err = json.NewDecoder(resp.Body).Decode(&stations)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return stationsToSongs(stations), nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("radio browser: no servers available")
}
