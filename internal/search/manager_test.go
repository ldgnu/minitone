package search

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ldgnu/minitone/internal/models"
)

func TestManagerAggregates(t *testing.T) {
	m := NewManager()
	m.AddSearcher(NewSearcher("YouTube", func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "yt-" + q, Source: models.SourceYouTube}}, nil
	}))
	m.AddSearcher(NewSearcher("Radio", func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		return []models.Song{{Title: "radio-" + q, Source: models.SourceRadio}}, nil
	}))

	var got models.SearchResults
	var wg sync.WaitGroup
	wg.Add(1)
	m.OnResult(func(r models.SearchResults) {
		got = r
		wg.Done()
	})

	m.Search("test")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for results")
	}

	if got.Total < 2 {
		t.Fatalf("expected >=2 results, got %+v", got)
	}
	if len(got.Groups) < 2 {
		t.Fatalf("expected groups, got %d", len(got.Groups))
	}
}

func TestManagerEmptyQuery(t *testing.T) {
	m := NewManager()
	var called bool
	m.OnResult(func(r models.SearchResults) {
		called = true
		if r.Total != 0 {
			t.Fatalf("expected empty")
		}
	})
	m.Search("")
	if !called {
		t.Fatal("onResult not called")
	}
}

func TestManagerCancel(t *testing.T) {
	m := NewManager()
	started := make(chan struct{})
	m.AddSearcher(NewSearcher("YouTube", func(ctx context.Context, q string, limit int) ([]models.Song, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return []models.Song{{Title: "late"}}, nil
		}
	}))

	var late bool
	m.OnResult(func(r models.SearchResults) {
		if r.Total > 0 {
			late = true
		}
	})
	m.Search("slow")
	<-started
	m.Cancel()
	time.Sleep(100 * time.Millisecond)
	if late {
		t.Fatal("cancelled search should not emit results")
	}
}
