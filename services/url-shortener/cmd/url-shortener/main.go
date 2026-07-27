package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/app"
	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/config"
	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/httpserver"
	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/logging"
	"github.com/basefoundry/banyanlabs/services/url-shortener/internal/storage/sqlite"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "url-shortener: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger := logging.New(os.Stdout, cfg.LogLevel, cfg.ServiceName)

	db, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open sqlite database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("failed to close database", slog.Any("error", closeErr))
		}
	}()

	if err := sqlite.RunMigrations(ctx, db); err != nil {
		return fmt.Errorf("run sqlite migrations: %w", err)
	}

	application := app.New(app.Options{
		Logger: logger,
		Store:  sqlite.NewStore(db),
	})
	server := httpserver.New(httpserver.Options{
		Addr:   cfg.Addr,
		App:    application,
		Logger: logger,
	})

	logger.Info("starting service", slog.String("addr", cfg.Addr), slog.String("database_path", cfg.DatabasePath))
	return server.ListenAndServe(ctx)
}
