package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/config"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/models"
	"time"
)

type ClickHouseStorage struct {
	db         driver.Conn
	BatchTimer time.Duration
}

func New(cfg config.ClickHouseStorage) (*ClickHouseStorage, error) {
	const op = "storage.clickhouse.New"

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.DB,
			Username: cfg.User,
			Password: cfg.Password,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "go-client", Version: "0.1"},
			},
		},
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v) //nolint: forbidigo
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		var exception *clickhouse.Exception
		if errors.As(err, &exception) {
			//nolint: forbidigo
			fmt.Printf(
				"Exception [%d] %s \n%s\n",
				exception.Code,
				exception.Message,
				exception.StackTrace,
			)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &ClickHouseStorage{db: conn}, nil
}

func (s *ClickHouseStorage) LogGoods(
	ctx context.Context,
	goods []models.Good,
) error {
	const op = "storage.clickhouse.LogGoods"

	if len(goods) == 0 {
		return nil
	}

	batch, err := s.db.PrepareBatch(ctx, `
		INSERT INTO hezzl.goods (Id, ProjectId, Name, Description, Priority, Removed)
	`)
	if err != nil {
		return fmt.Errorf("%s: prepare batch: %w", op, err)
	}

	for i, good := range goods {
		var removed uint8
		if good.Removed {
			removed = 1
		}

		err = batch.Append(
			good.ID,
			good.ProjectID,
			good.Name,
			good.Description,
			good.Priority,
			removed,
		)
		if err != nil {
			return fmt.Errorf("%s: append item %d to batch: %w", op, i, err)
		}
	}

	if err = batch.Send(); err != nil {
		return fmt.Errorf("%s: send batch of %d items: %w", op, len(goods), err)
	}

	return nil
}
