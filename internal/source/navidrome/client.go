package navidrome

import (
	"context"
	"fmt"

	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/subsonic"
)

type Client struct {
	api *subsonic.Client
}

func New(api *subsonic.Client) *Client {
	return &Client{api: api}
}

func (c *Client) Search(query string, limit int) ([]models.Song, error) {
	return c.SearchContext(context.Background(), query, limit)
}

// SearchContext is Search with cancellation, so a superseded query stops
// hitting the server immediately.
func (c *Client) SearchContext(ctx context.Context, query string, limit int) ([]models.Song, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	songs, err := c.api.SearchContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("navidrome search: %w", err)
	}

	result := make([]models.Song, 0, len(songs))
	for _, s := range songs {
		result = append(result, models.Song{
			ID:       "nav:" + models.Sanitize(s.ID),
			Source:   models.SourceNavidrome,
			SourceID: models.Sanitize(s.ID),
			Title:    models.Sanitize(s.Title),
			Artist:   models.Sanitize(s.Artist),
			Album:    models.Sanitize(s.Album),
			Duration: s.Duration,
		})
	}
	return result, nil
}

func (c *Client) StreamURL(songID string) string {
	return c.api.StreamURL(songID)
}

func (c *Client) GetArtists() ([]models.Song, error) {
	artists, err := c.api.GetArtists()
	if err != nil {
		return nil, fmt.Errorf("navidrome artists: %w", err)
	}

	result := make([]models.Song, 0, len(artists))
	for _, a := range artists {
		result = append(result, models.Song{
			ID:     "nav:" + a.ID,
			Source: models.SourceNavidrome,
			Title:  a.Name,
			Artist: a.Name,
			Genre:  "artist",
		})
	}
	return result, nil
}

func (c *Client) GetAlbums(artistID string) ([]models.Song, error) {
	albums, err := c.api.GetArtist(artistID)
	if err != nil {
		return nil, fmt.Errorf("navidrome albums: %w", err)
	}

	result := make([]models.Song, 0, len(albums))
	for _, a := range albums {
		result = append(result, models.Song{
			ID:     "nav:" + a.ID,
			Source: models.SourceNavidrome,
			Title:  a.Name,
			Artist: a.Artist,
			Album:  a.Name,
			Year:   a.Year,
			Genre:  "album",
		})
	}
	return result, nil
}

func (c *Client) GetAlbum(id string) ([]models.Song, error) {
	songs, err := c.api.GetAlbum(id)
	if err != nil {
		return nil, fmt.Errorf("navidrome album: %w", err)
	}

	result := make([]models.Song, 0, len(songs))
	for _, s := range songs {
		result = append(result, models.Song{
			ID:       "nav:" + models.Sanitize(s.ID),
			Source:   models.SourceNavidrome,
			SourceID: models.Sanitize(s.ID),
			Title:    models.Sanitize(s.Title),
			Artist:   models.Sanitize(s.Artist),
			Album:    models.Sanitize(s.Album),
			Duration: s.Duration,
		})
	}
	return result, nil
}
