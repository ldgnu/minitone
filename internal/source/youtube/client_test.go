package youtube

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestSearchLive(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		t.Skip("yt-dlp missing")
	}
	c := New()
	c.timeout = 25 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	songs, err := c.SearchContext(ctx, "lofi hip hop", 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(songs) == 0 {
		t.Fatal("no results")
	}
	if songs[0].SourceID == "" {
		t.Fatal("missing id")
	}
}
