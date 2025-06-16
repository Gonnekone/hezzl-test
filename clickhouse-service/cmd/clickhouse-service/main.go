package main

import (
	"context"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/listener"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/storage/clickhouse"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/config"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/lib/logger/handlers/slogpretty"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/lib/logger/sl"
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

	clh, err := clickhouse.New(cfg.ClickHouseStorage)
	if err != nil {
		log.Error("failed to create clickhouse", sl.Err(err))
		os.Exit(1)
	}

	lis, err := listener.New(log, cfg.Nats, clh)
	if err != nil {
		log.Error("failed to create listener", sl.Err(err))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = lis.Start(ctx)
	if err != nil {
		log.Error("failed to start listener", sl.Err(err))
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("server started")

	<-done
	log.Info("stopping server")

	lis.Close()

	log.Debug("closing storage")

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
