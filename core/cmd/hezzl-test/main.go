package main

import (
	"context"
	"errors"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/create"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/list"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/remove"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/reprioritize"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/update"
	"github.com/Gonnekone/hezzl-test/core/internal/producer"
	"github.com/Gonnekone/hezzl-test/core/internal/storage"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Gonnekone/hezzl-test/core/internal/config"
	mwLogger "github.com/Gonnekone/hezzl-test/core/internal/http-server/middleware/logger"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/handlers/slogpretty"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/sl"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info("starting up the application", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	superStorage, err := storage.New(cfg.PostgresStorage, cfg.RedisStorage)
	if err != nil {
		log.Error("failed to create storage", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	producer, err := producer.New(log, cfg.Nats)
	if err != nil {
		log.Error("failed to create producer", sl.Err(err))
		os.Exit(1)
	}

	router.Post("/good/create", create.New(log, superStorage, producer))
	router.Patch("/good/update", update.New(log, superStorage, producer))
	router.Delete("/good/remove", remove.New(log, superStorage, producer))
	router.Patch("/good/reprioritize", reprioritize.New(log, superStorage, producer))

	router.Get("/goods/list", list.New(log, superStorage))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", sl.Err(err))

		return
	}

	log.Debug("closing storage")

	superStorage.PostgresStorage.Close()
	superStorage.RedisStorage.Close()

	producer.Close()

	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()

	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
