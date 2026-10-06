package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ldgnu/minitone/internal/source/youtube"
	"github.com/ldgnu/minitone/internal/store"
)

// ImportPlaylist fetches a public YouTube playlist URL and stores it.
// Usage: minitone --import-playlist <URL> [name]
func ImportPlaylist(url, name string) {
	url = strings.TrimSpace(url)
	if url == "" {
		fmt.Fprintln(os.Stderr, "minitone: empty playlist URL")
		os.Exit(2)
	}
	yt := youtube.New()
	tracks, err := yt.FetchPlaylist(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "minitone: import failed: %v\n", err)
		os.Exit(1)
	}
	if name == "" {
		name = "YouTube import"
	}
	id := "yt-" + slugify(url)
	playls := store.DefaultPlaylists()
	playls.Upsert(store.Playlist{
		ID:          id,
		Name:        name,
		Kind:        store.PlaylistStatic,
		SourceURL:   url,
		Tracks:      tracks,
		UpdatedAt:   time.Now(),
		AutoRefresh: false,
	})
	fmt.Printf("imported %d tracks into %q\n", len(tracks), name)
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if len(out) > 32 {
		out = out[len(out)-32:]
	}
	if out == "" {
		out = fmt.Sprintf("%d", time.Now().Unix())
	}
	return out
}
