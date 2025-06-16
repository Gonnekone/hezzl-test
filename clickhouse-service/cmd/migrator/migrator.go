package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var migrationsPath, direction string

	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&direction, "direction", "up", "migration direction (up/down)")

	cfg := config.MustLoad()

	if migrationsPath == "" {
		log.Fatal("migrations-path is required")
	}

	if strings.HasPrefix(cfg.ClickHouseStorage.Addr, "clickhouse") {
		cfg.ClickHouseStorage.Addr = "host.docker.internal" +
			strings.TrimPrefix(cfg.ClickHouseStorage.Addr, "clickhouse")
	}

	clickhouseDSN := cfg.ClickHouseStorage.DSN() + "?x-multi-statement=true"

	fmt.Printf("Connecting to ClickHouse: %s\n", clickhouseDSN)
	fmt.Printf("Migrations path: file://%s\n", migrationsPath)

	m, err := migrate.New("file://"+migrationsPath, clickhouseDSN)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	switch direction {
	case "up":
		fmt.Println("Applying UP migrations...")
		err = m.Up()

	case "down":
		fmt.Println("Applying DOWN migrations...")
		err = m.Down()

	default:
		log.Fatalf("Invalid direction: %s. Use 'up' or 'down'", direction)
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No changes to apply")
			return
		}
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migrations applied successfully")
}
