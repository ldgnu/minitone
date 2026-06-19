// minitone - TUI for Apple Music via Cider
// by ldgnu <ldgnu@users.noreply.github.com>


package cider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) Lyrics(trackID string) (string, error) {
	body, err := c.doGET("/api/v1/lyrics/" + url.PathEscape(trackID))
	if err != nil {
		return "", err
	}
	return parseLyrics(body), nil
}

func (c *Client) LyricsLRCLIB(track, artist string) (string, error) {
	q := url.Values{}
	q.Set("track_name", track)
	q.Set("artist_name", artist)

	req, err := http.NewRequest(http.MethodGet,
		"https://lrclib.net/api/get?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "minitone/0.1")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lrclib: HTTP %d", resp.StatusCode)
	}

	var payload struct {
		SyncedLyrics string `json:"syncedLyrics"`
		PlainLyrics  string `json:"plainLyrics"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	text := strings.TrimSpace(payload.SyncedLyrics)
	if text == "" {
		text = strings.TrimSpace(payload.PlainLyrics)
	}
	if text == "" {
		return "", fmt.Errorf("no lyrics found")
	}
	return stripTimestamps(text), nil
}

func parseLyrics(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" || raw == "[]" || raw == "{}" {
		return ""
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return raw
	}

	lines := make([]string, 0, 64)
	collectText(data, &lines)

	seen := map[string]bool{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func collectText(node any, out *[]string) {
	switch v := node.(type) {
	case map[string]any:
		for k, val := range v {
			if s, ok := val.(string); ok {
				switch strings.ToLower(k) {
				case "text", "line", "lyric", "lyrics", "content":
					*out = append(*out, s)
				}
			}
			collectText(val, out)
		}
	case []any:
		for _, item := range v {
			collectText(item, out)
		}
	}
}

func stripTimestamps(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		for strings.Contains(clean, "[") && strings.Contains(clean, "]") {
			start := strings.Index(clean, "[")
			end := strings.Index(clean, "]")
			if start < end {
				clean = strings.TrimSpace(clean[:start] + clean[end+1:])
			} else {
				break
			}
		}
		if clean != "" {
			out = append(out, clean)
		}
	}
	return strings.Join(out, "\n")
}
