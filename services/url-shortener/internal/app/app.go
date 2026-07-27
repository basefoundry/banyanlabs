package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage"
)

const serviceName = "url-shortener"

type App struct {
	logger        *slog.Logger
	loginFailures map[string]loginFailureState
	loginMu       sync.Mutex
	started       time.Time
	store         storage.Store
}

type Options struct {
	Logger *slog.Logger
	Store  storage.Store
}

type Health struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
}

func New(options Options) *App {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &App{
		logger:        logger.With(slog.String("component", "app")),
		loginFailures: make(map[string]loginFailureState),
		started:       time.Now().UTC(),
		store:         options.Store,
	}
}

func (app *App) Health(_ context.Context) Health {
	return Health{
		Service: serviceName,
		Status:  "ok",
		Uptime:  time.Since(app.started).Round(time.Millisecond).String(),
	}
}
