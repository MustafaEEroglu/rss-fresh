// rss-fresh is a single-binary RSS reader: HTTP API + SPA + cron worker + Telegram.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
	"github.com/mustafaeeroglu/rss-fresh/internal/db"
	"github.com/mustafaeeroglu/rss-fresh/internal/httpapi"
	"github.com/mustafaeeroglu/rss-fresh/internal/retention"
	"github.com/mustafaeeroglu/rss-fresh/internal/rss"
	"github.com/mustafaeeroglu/rss-fresh/internal/scheduler"
	"github.com/mustafaeeroglu/rss-fresh/internal/telegram"
	"github.com/mustafaeeroglu/rss-fresh/web"
)

func main() {
	// Subcommand "healthcheck" used by the Docker HEALTHCHECK directive: pings
	// /api/v1/healthz on PORT and exits 0/1.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	_ = godotenv.Load() // silently no-op if .env is missing

	logger := newLogger(os.Getenv("LOG_LEVEL"))

	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(rootCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db open: %w", err)
	}
	defer database.Close()

	if err := database.EnsureSchema(rootCtx); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	notifier, err := telegram.New(cfg, database, log)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	go notifier.Run(rootCtx)
	defer notifier.Close()

	fetcher := rss.New(cfg, database, log, notifier)

	sch, err := scheduler.New(cfg.TZ, log)
	if err != nil {
		return fmt.Errorf("scheduler: %w", err)
	}
	if err := sch.AddCron("rss-fetch", cfg.FetchCron, func(ctx context.Context) {
		fetcher.Tick(ctx)
	}); err != nil {
		return err
	}
	if err := sch.AddCron("daily-digest", cfg.DigestCron, func(ctx context.Context) {
		notifier.SendDigest(ctx)
	}); err != nil {
		return err
	}
	purger := retention.New(database, log, cfg.RetentionDays)
	if cfg.RetentionDays > 0 {
		if err := sch.AddCron("article-retention", cfg.RetentionCron, func(ctx context.Context) {
			purger.Tick(ctx)
		}); err != nil {
			return err
		}
	}
	sch.Start()
	defer func() {
		if err := sch.Stop(); err != nil {
			log.Warn("scheduler shutdown", "err", err)
		}
	}()

	srv := httpapi.NewServer(cfg, database, log, fetcher, web.FS())
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      35 * time.Second, // slightly > chi Timeout middleware (30 s)
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", server.Addr, "version", cfg.Version)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-rootCtx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("http shutdown", "err", err)
	}
	return nil
}

func runHealthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://" + net.JoinHostPort("127.0.0.1", port) + "/api/v1/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}
