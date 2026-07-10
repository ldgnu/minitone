package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

type Client struct {
	timeout time.Duration
}

func New() *Client {
	return &Client{timeout: 15 * time.Second}
}

func (c *Client) Search(query string, limit int) ([]models.Song, error) {
	return c.SearchContext(context.Background(), query, limit)
}

func (c *Client) SearchContext(ctx context.Context, query string, limit int) ([]models.Song, error) {
	if limit <= 0 {
		limit = 10
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return nil, fmt.Errorf("yt-dlp not found in PATH")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--flat-playlist",
		"--dump-single-json",
		"--no-warnings",
		"--no-playlist",
		fmt.Sprintf("ytsearch%d:%s", limit, query),
	)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return c.searchFallback(ctx, query, limit)
	}

	var result struct {
		Entries []struct {
			Title    string  `json:"title"`
			ID       string  `json:"id"`
			Duration float64 `json:"duration"`
			Webpage  string  `json:"webpage_url"`
			Channel  string  `json:"channel"`
			Uploader string  `json:"uploader"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out, &result); err != nil || len(result.Entries) == 0 {
		return c.searchFallback(ctx, query, limit)
	}

	songs := make([]models.Song, 0, len(result.Entries))
	for _, e := range result.Entries {
		if e.ID == "" {
			continue
		}
		artist := e.Channel
		if artist == "" {
			artist = e.Uploader
		}
		url := e.Webpage
		if url == "" {
			url = "https://www.youtube.com/watch?v=" + e.ID
		}
		songs = append(songs, models.Song{
			ID:       "yt:" + e.ID,
			Source:   models.SourceYouTube,
			SourceID: e.ID,
			Title:    e.Title,
			Artist:   artist,
			Duration: int(e.Duration),
			URL:      url,
		})
	}
	return songs, nil
}

func (c *Client) searchFallback(ctx context.Context, query string, limit int) ([]models.Song, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--print", "%(title)s\t%(id)s\t%(channel)s\t%(duration)s",
		"--no-warnings",
		fmt.Sprintf("ytsearch%d:%s", limit, query),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var songs []models.Song
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		id := strings.TrimSpace(parts[1])
		if id == "" {
			continue
		}
		artist := ""
		if len(parts) > 2 {
			artist = strings.TrimSpace(parts[2])
		}
		dur := 0
		if len(parts) > 3 {
			fmt.Sscanf(strings.TrimSpace(parts[3]), "%d", &dur)
		}
		songs = append(songs, models.Song{
			ID:       "yt:" + id,
			Source:   models.SourceYouTube,
			SourceID: id,
			Title:    strings.TrimSpace(parts[0]),
			Artist:   artist,
			Duration: dur,
			URL:      "https://www.youtube.com/watch?v=" + id,
		})
	}
	return songs, nil
}

// Resolve returns title and direct stream URL for a YouTube page URL.
func (c *Client) Resolve(url string) (string, string, error) {
	return c.ResolveContext(context.Background(), url)
}

func (c *Client) ResolveContext(ctx context.Context, url string) (string, string, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return "", "", fmt.Errorf("yt-dlp not found in PATH")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "bestaudio/best",
		"--print", "%(title)s",
		"--print", "%(url)s",
		"--no-warnings",
		"--no-playlist",
		url,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("yt-dlp resolve: %w", err)
	}
	lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
	if len(lines) < 2 || strings.TrimSpace(lines[1]) == "" {
		return "", "", fmt.Errorf("no stream found for %s", url)
	}
	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1]), nil
}

func (c *Client) ResolveSong(song models.Song) (string, error) {
	return c.ResolveSongContext(context.Background(), song)
}

func (c *Client) ResolveSongContext(ctx context.Context, song models.Song) (string, error) {
	url := song.URL
	if url == "" {
		if song.SourceID == "" {
			return "", fmt.Errorf("no youtube id")
		}
		url = "https://www.youtube.com/watch?v=" + song.SourceID
	}
	_, streamURL, err := c.ResolveContext(ctx, url)
	return streamURL, err
}
