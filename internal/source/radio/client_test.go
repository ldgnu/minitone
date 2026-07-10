package radio

import (
	"context"
	"testing"
	"time"
)

func TestSearchLive(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	songs, err := SearchContext(ctx, "jazz", 5)
	if err != nil {
		t.Skipf("radio browser unavailable: %v", err)
	}
	if len(songs) == 0 {
		t.Skip("no stations returned")
	}
	if songs[0].URL == "" {
		t.Fatal("missing stream url")
	}
	if songs[0].Source != "radio" {
		t.Fatalf("source %s", songs[0].Source)
	}
}
