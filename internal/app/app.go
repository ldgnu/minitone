package app

import (
	"context"
	"fmt"
	"os"
	"time"

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
var Version = "0.3.0"

type App struct {
	cfg    *config.Config
	player *player.Player
	queue  *queue.Queue
	sm     *search.Manager
	model  ui.Model
	closed bool
}

// New wires every collaborator together.
//
// It never aborts because a dependency is missing: a missing mpv, a missing
// yt-dlp or an unreachable Navidrome are reported *inside* the UI, where the
// user can read what to do about it.
func New() *App {
	cfg := config.Load()

	q := queue.New()
	sm := search.NewManager()
	sm.SetDebounce(cfg.Debounce())
	if cfg.DefaultSource != "" && cfg.DefaultSource != "all" {
		sm.RestrictTo([]string{cfg.DefaultSource})
	}

	yt := youtube.New()
	favs := store.DefaultFavorites()
	hist := store.DefaultHistory()

	p := player.New()
	p.SetVolume(cfg.Volume)

	// mpv is optional at start-up: the UI explains itself if it is absent.
	if err := p.Start(); err != nil && err != player.ErrNotInstalled {
		fmt.Fprintf(os.Stderr, "minitone: player: %v\n", err)
	}
	if player.Available() {
		p.Start()
	}

	var nd *navSrc.Client
	if cfg.HasNavidrome() {
		sc := subsonic.NewClient(cfg.NavidromeURL, cfg.NavidromeUser, cfg.NavidromePass)
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := sc.PingContext(pingCtx)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "minitone: navidrome ping failed: %v\n", err)
		} else {
			nd = navSrc.New(sc)
		}
	}

	ls := library.NewWithDirs(cfg.LibraryPaths)

	sm.AddSearcher(search.NewSearcher(search.SourceYouTube, func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return yt.SearchContext(ctx, query, limit)
	}))
	sm.AddSearcher(search.NewSearcher(search.SourceRadio, func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return radio.SearchContext(ctx, query, limit)
	}))
	if nd != nil {
		sm.AddSearcher(search.NewSearcher(search.SourceNavidrome, func(ctx context.Context, query string, limit int) ([]models.Song, error) {
			return nd.SearchContext(ctx, query, limit)
		}))
	}
	sm.AddSearcher(search.NewSearcher(search.SourceLibrary, func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return ls.SearchContext(ctx, query, limit)
	}))
	sm.AddSearcher(search.NewSearcher(search.SourceFavorites, func(ctx context.Context, query string, limit int) ([]models.Song, error) {
		return favs.Search(query, limit), nil
	}))

	keys := ui.NewKeyMap()
	keys.Apply(cfg.KeyBindings)

	var session *store.SessionStore
	if cfg.RestoreSession {
		session = store.DefaultSessionStore()
	}

	m := ui.New(ui.Deps{
		Player:   p,
		Queue:    q,
		Search:   sm,
		YouTube:  yt,
		Nav:      nd,
		Library:  ls,
		Favs:     favs,
		History:  hist,
		Session:  session,
		Theme:    cfg.Theme,
		Debounce: cfg.Debounce(),
		Restore:  cfg.RestoreSession,
		Keys:     &keys,
	})

	return &App{
		cfg:    cfg,
		player: p,
		queue:  q,
		sm:     sm,
		model:  m,
	}
}

// Run starts the TUI and guarantees every resource is released.
func (a *App) Run() error {
	program := tea.NewProgram(a.model, tea.WithAltScreen())
	_, err := program.Run()

	a.shutdown()
	return err
}

// shutdown stops background work, saves the session and closes mpv once.
//
// Ordered on purpose: cancel the searches first so no late result can restart
// playback, then persist the session, then tear mpv down.
func (a *App) shutdown() {
	if a.closed {
		return
	}
	a.closed = true

	if a.sm != nil {
		a.sm.Cancel()
	}
	a.model.SaveSessionNow()
	if a.player != nil {
		a.player.Close()
	}
}
