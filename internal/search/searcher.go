package search

import (
	"context"

	"github.com/ldgnu/minitone/internal/models"
)

type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]models.Song, error)
	Name() string
}

type SearcherFunc struct {
	Fn   func(ctx context.Context, query string, limit int) ([]models.Song, error)
	name string
}

func (s SearcherFunc) Search(ctx context.Context, query string, limit int) ([]models.Song, error) {
	return s.Fn(ctx, query, limit)
}

func (s SearcherFunc) Name() string {
	return s.name
}

func NewSearcher(name string, fn func(ctx context.Context, query string, limit int) ([]models.Song, error)) Searcher {
	return SearcherFunc{Fn: fn, name: name}
}

type Result struct {
	Source string
	Song   models.Song
	Score  float64
}
