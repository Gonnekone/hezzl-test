package postgres

import (
	"context"
	"fmt"
	"github.com/Gonnekone/hezzl-test/core/internal/config"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/list"
	"github.com/Gonnekone/hezzl-test/core/internal/models"
	"github.com/jackc/pgx/v5"
)

const defaultPriority = 0

var (
// place for custom errors
)

type PostgresStorage struct {
	db          *pgx.Conn
	maxPriority int
}

func New(cfg config.PostgresStorage) (*PostgresStorage, error) {
	const op = "storage.postgres.New"

	conn, err := pgx.Connect(context.Background(), cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &PostgresStorage{db: conn, maxPriority: defaultPriority}, nil
}

func (s *PostgresStorage) SaveGood(
	ctx context.Context,
	name string,
	projectID string,
) (*models.Good, error) {
	const op = "storage.postgres.SaveGood"

	if s.maxPriority == defaultPriority {
		err := s.fetchMaxPriority()
		if err != nil {
			return nil, fmt.Errorf("%s: fetch max priority: %w", op, err)
		}
	}

	s.maxPriority++

	query := `
		INSERT INTO goods(name, project_id, priority)
		VALUES ($1, $2, $3)
		RETURNING id, project_id, name, description, priority, removed, created_at
	`

	var good models.Good
	err := s.db.QueryRow(ctx, query, name, projectID, s.maxPriority).Scan(
		&good.ID,
		&good.ProjectID,
		&good.Name,
		&good.Description,
		&good.Priority,
		&good.Removed,
		&good.CreatedAt,
	)
	if err != nil {
		s.maxPriority--
		return nil, fmt.Errorf("%s: scan good: %w", op, err)
	}

	return &good, nil
}

func (s *PostgresStorage) UpdateGood(
	ctx context.Context,
	id string,
	projectID string,
	name string,
	desc string,
) (*models.Good, error) {
	const op = "storage.postgres.UpdateGood"

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	//nolint: errcheck
	defer tx.Rollback(ctx)

	//nolint: goconst
	lockQuery := `SELECT id FROM goods WHERE id = $1 AND project_id = $2 FOR UPDATE`
	if _, err := tx.Exec(ctx, lockQuery, id, projectID); err != nil {
		return nil, fmt.Errorf("%s: lock row: %w", op, err)
	}

	query := `UPDATE goods SET name = $1`
	args := []any{name}
	argIdx := 2

	if desc != "" {
		query += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, desc)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND project_id = $%d", argIdx, argIdx+1)
	args = append(args, id, projectID)

	query += ` RETURNING id, project_id, name, description, priority, removed, created_at`

	var res models.Good
	err = tx.QueryRow(ctx, query, args...).Scan(
		&res.ID,
		&res.ProjectID,
		&res.Name,
		&res.Description,
		&res.Priority,
		&res.Removed,
		&res.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: scan good: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return &res, nil
}

func (s *PostgresStorage) UpdateGoodsPriority(
	ctx context.Context,
	id string,
	projectID string,
	priority int,
) ([]models.Good, error) {
	const op = "storage.postgres.UpdateGoodsPriority"

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	//nolint: errcheck
	defer tx.Rollback(ctx)

	//nolint: goconst
	lockQuery := `SELECT id FROM goods WHERE id = $1 AND project_id = $2 FOR UPDATE`
	if _, err := tx.Exec(ctx, lockQuery, id, projectID); err != nil {
		return nil, fmt.Errorf("%s: lock row: %w", op, err)
	}

	query := `
		UPDATE goods SET priority = $1 WHERE id = $2 AND project_id = $3
		RETURNING id, project_id, name, description, priority, removed, created_at
	`

	var good models.Good
	err = tx.QueryRow(ctx, query, priority, id, projectID).Scan(
		&good.ID,
		&good.ProjectID,
		&good.Name,
		&good.Description,
		&good.Priority,
		&good.Removed,
		&good.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: update current: %w", op, err)
	}

	var res []models.Good
	res = append(res, good)

	query = `
		UPDATE goods
		SET priority = priority + 1
		WHERE priority >= $1 AND id != $2 AND project_id = $3 AND removed = false
		RETURNING id, project_id, name, description, priority, removed, created_at
	`

	rows, err := tx.Query(ctx, query, priority, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("%s: reprioritize: %w", op, err)
	}

	for rows.Next() {
		var g models.Good
		if err := rows.Scan(
			&g.ID,
			&g.ProjectID,
			&g.Name,
			&g.Description,
			&g.Priority,
			&g.Removed,
			&g.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan reprioritized: %w", op, err)
		}
		res = append(res, g)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	if s.maxPriority < priority {
		s.maxPriority = priority
	} else {
		s.maxPriority++
	}

	return res, nil
}

func (s *PostgresStorage) DeleteGood(
	ctx context.Context,
	id string,
	projectID string,
) (*models.Good, error) {
	const op = "storage.postgres.DeleteGood"

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	//nolint: errcheck
	defer tx.Rollback(ctx)

	//nolint: goconst
	lockQuery := `SELECT id FROM goods WHERE id = $1 AND project_id = $2 FOR UPDATE`
	if _, err := tx.Exec(ctx, lockQuery, id, projectID); err != nil {
		return nil, fmt.Errorf("%s: lock row: %w", op, err)
	}

	query := `
		UPDATE goods
		SET removed = TRUE
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, name, description, priority, removed, created_at
	`

	var good models.Good
	err = s.db.QueryRow(ctx, query, id, projectID).Scan(
		&good.ID,
		&good.ProjectID,
		&good.Name,
		&good.Description,
		&good.Priority,
		&good.Removed,
		&good.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: delete good: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return &good, nil
}

func (s *PostgresStorage) ListGoods(
	ctx context.Context,
	limit int,
	offset int,
) (*list.GoodListResponse, error) {
	const op = "storage.postgres.ListGoods"

	query := `
		SELECT * FROM goods
		ORDER BY id ASC
		LIMIT $1 OFFSET $2 
	`

	var res list.GoodListResponse
	rows, err := s.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: list goods: %w", op, err)
	}

	var total, removed int
	for rows.Next() {
		var good models.Good
		if err := rows.Scan(
			&good.ID,
			&good.ProjectID,
			&good.Name,
			&good.Description,
			&good.Priority,
			&good.Removed,
			&good.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: scan good %w", op, err)
		}

		if !good.Removed {
			res.Goods = append(res.Goods, good)
		} else {
			removed++
		}

		total++
	}

	res.Meta = list.GoodMetaListResponse{
		Total:   total,
		Removed: removed,
		Limit:   limit,
		Offset:  offset,
	}

	return &res, nil
}

func (s *PostgresStorage) GetGood(
	ctx context.Context,
	id string,
	projectID string,
) (*models.Good, error) {
	const op = "storage.postgres.GetGood"

	query := `
		SELECT * FROM goods
		WHERE id = $1 AND project_id = $2
	`

	var good models.Good
	if err := s.db.QueryRow(ctx, query, id, projectID).Scan(
		&good.ID,
		&good.ProjectID,
		&good.Name,
		&good.Description,
		&good.Priority,
		&good.Removed,
		&good.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("%s: get good: %w", op, err)
	}

	return &good, nil
}

func (s *PostgresStorage) Close() {
	s.db.Close(context.Background())
}

func (s *PostgresStorage) fetchMaxPriority() error {
	query := `
		SELECT MAX(priority) FROM goods
	`

	var maxPriority *int
	err := s.db.QueryRow(context.Background(), query).Scan(&maxPriority)
	if maxPriority == nil {
		return nil
	}

	if err != nil {
		return err
	}

	s.maxPriority = *maxPriority
	return err
}
