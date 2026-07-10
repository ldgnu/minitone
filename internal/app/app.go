package app

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ldgnu/minitone/internal/config"
	"github.com/ldgnu/minitone/internal/models"
	"github.com/ldgnu/minitone/internal/player"
	"github.com/ldgnu/minitone/internal/queue"
	"github.com/ldgnu/minitone/internal/search"
	"github.com/ldgnu/minitone/internal/source/library"
	navSrc "github.com/ldgnu/minitone/internal/source/navidrome"
	"github.com/ldgnu/minitone/internal/source/radio"
	"github.com/ldgnu/minitone/internal/source/youtube"
	"github.com/ldgnu/minitone/internal/store"
	"github.com/ldgnu/minitone/internal/subsonic"
	"github.com/ldgnu/minitone/internal/ui"
)

// Version is set at link time via -ldflags "-X ...Version=x.y.z"
var Version = "0.2.0"

type App struct {
	cfg    *config.Config
	player *player.Player
	queue  *queue.Queue
	sm     *search.Manager
	favs   *store.Favorites
	hist   *store.History
	model  ui.Model
}

func New() *App {
	cfg := config.Load()

	p := player.New()
	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "minitone: player: %v\n", err)
		os.Exit(1)
	}
	p.SetVolume(cfg.Volume)

	q := queue.New()
	sm := search.NewManager()
	yt := youtube.New()
	favs := store.DefaultFavorites()
	hist := store.DefaultHistory()

	var nd *navSrc.Client
	if cfg.HasNavidrome() {
		sc := subsonic.NewClient(cfg.NavidromeURL, cfg.NavidromeUser, cfg.NavidromePass)
		if err := sc.Ping(); err != nil {
			fmt.Fprintf(os.Stderr, "minitone: navidrome ping failed: %v\n", err)
		} else {
			nd = navSrc.New(sc)
		}
	}

	ls := library.NewWithDirs(cfg.LibraryPaths)
	go func() {
		if err := ls.Scan(); err != nil {
			fmt.Fprintf(os.Stderr, "minitone: library scan: %v\n", err)
		}
	}()

	sm.AddSearcher(search.NewSearcher("YouTube", func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return yt.SearchContext(ctx, query, limit)
	}))

	sm.AddSearcher(search.NewSearcher("Radio", func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return radio.SearchContext(ctx, query, limit)
	}))

	if nd != nil {
		sm.AddSearcher(search.NewSearcher("Navidrome", func(ctx context.Context, query string, limit int) ([]models.Song, error) {
			return nd.Search(query, limit)
		}))
	}

	if ls != nil {
		sm.AddSearcher(search.NewSearcher("Library", func(ctx context.Context, query string, limit int) ([]models.Song, error) {
			return ls.Search(query, limit)
		}))
	}

	// Favorites searchable as a source group when you have some.
	sm.AddSearcher(search.NewSearcher("Favorites", func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return favs.Search(query, limit), nil
	}))

	return &App{
		cfg:    cfg,
		player: p,
		queue:  q,
		sm:     sm,
		favs:   favs,
		hist:   hist,
		model: ui.New(ui.Deps{
			Player:  p,
			Queue:   q,
			Search:  sm,
			YouTube: yt,
			Nav:     nd,
			Library: ls,
			Favs:    favs,
			History: hist,
			Theme:   cfg.Theme,
		}),
	}
}

func (a *App) Run() error {
	program := tea.NewProgram(a.model, tea.WithAltScreen())
	_, err := program.Run()
	a.sm.Cancel()
	a.player.Close()
	return err
}
