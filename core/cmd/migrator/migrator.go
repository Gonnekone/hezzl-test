package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Gonnekone/hezzl-test/core/internal/config"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/sl"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"log/slog"
	"os"
)

func main() {
	var migrationsPath, direction string

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&direction, "direction", "up", "migration direction (up/down)")

	cfg := config.MustLoad()

	if migrationsPath == "" {
		log.Error("migrations-path is required")
		os.Exit(1)
	}

	if cfg.PostgresStorage.Host == "postgres" {
		cfg.PostgresStorage.Host = "host.docker.internal"
	}

	m, err := migrate.New("file://"+migrationsPath, cfg.PostgresStorage.DSN()+"?sslmode=disable")
	if err != nil {
		msg := fmt.Sprintf("file://%s %s?sslmode=disable",
			migrationsPath,
			cfg.PostgresStorage.DSN(),
		)

		//nolint: sloglint
		log.Debug(msg)
		log.Error("failed to create migrate instance", sl.Err(err))
		os.Exit(1)
	}

	switch direction {
	case "up":
		err = m.Up()

	case "down":
		err = m.Down()
	}
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no changes to apply")

			return
		}
		log.Error("failed to apply migrations", sl.Err(err))
		os.Exit(1)
	}

	log.Info("migrations applied successfully")
}
