package ui

import (
	"context"

	"github.com/ldgnu/minitone/internal/discover"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/source/youtube"
	"github.com/ldgnu/minitone/internal/store"
)

func teaContext() context.Context {
	// YouTube SearchContext already applies its own per-query timeout.
	return context.Background()
}

func refreshPlaylistTracks(ctx context.Context, pl store.Playlist, yt *youtube.Client) ([]models.Song, error) {
	return discover.Refresh(ctx, pl, yt.SearchContext, 8)
}
