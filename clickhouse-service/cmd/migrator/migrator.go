package main

import (
	"errors"
	"flag"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/lib/logger/sl"
	"log/slog"
	"os"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	if strings.HasPrefix(cfg.ClickHouseStorage.Addr, "clickhouse") {
		cfg.ClickHouseStorage.Addr = "host.docker.internal" +
			strings.TrimPrefix(cfg.ClickHouseStorage.Addr, "clickhouse")
	}

	clickhouseDSN := cfg.ClickHouseStorage.DSN() + "?x-multi-statement=true"

	log.Debug("Connecting to ClickHouse", slog.String("dsn", clickhouseDSN))

	m, err := migrate.New("file://"+migrationsPath, clickhouseDSN)
	if err != nil {
		log.Error("failed to create migrate instance", sl.Err(err))
		os.Exit(1)
	}
	defer m.Close()

	switch direction {
	case "up":
		err = m.Up()

	case "down":
		err = m.Down()

	default:
		log.Error("Invalid direction", slog.String("direction", direction))
		os.Exit(1)
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("No changes to apply")
			return
		}
		log.Error("Migration failed", sl.Err(err))
		os.Exit(1)
	}

	log.Info("Migrations applied successfully")
}
