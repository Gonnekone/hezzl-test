package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Gonnekone/hezzl-test/core/internal/config"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/list"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
// place for custom errors
)

type RedisStorage struct {
	client *redis.Client
}

func (s *RedisStorage) Close() error {
	return s.client.Close()
}

func New(cfg config.RedisStorage) (*RedisStorage, error) {
	const op = "storage.redis.New"

	rdb := redis.NewClient(&redis.Options{
		Addr:       cfg.Addr,
		Password:   cfg.Password,
		DB:         cfg.DB,
		MaxRetries: cfg.MaxRetries,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RedisStorage{client: rdb}, nil
}

func (s *RedisStorage) SaveListInCache(ctx context.Context, list list.GoodListResponse) error {
	const op = "storage.redis.SaveList"

	key := fmt.Sprintf("limit:%d-offset:%d", list.Meta.Limit, list.Meta.Offset)

	listJSON, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal list: %w", op, err)
	}

	err = s.client.Set(ctx, key, listJSON, time.Minute).Err()
	if err != nil {
		return fmt.Errorf("%s: failed to save list to redis: %w", op, err)
	}

	return nil
}

func (s *RedisStorage) GetCachedList(ctx context.Context, limit, offset int) ([]byte, error) {
	const op = "storage.redis.GetList"

	key := fmt.Sprintf("limit:%d-offset:%d", limit, offset)

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("%s: no data found for limit %d and offset %d", op, limit, offset)
		}
		return nil, fmt.Errorf("%s: failed to get list from redis: %w", op, err)
	}

	//var response list.GoodListResponse
	//if err := json.Unmarshal(data, &response); err != nil {
	//	return nil, fmt.Errorf("%s: failed to unmarshal list: %w", op, err)
	//}
	//
	//return &response, nil

	//if len(data) == 0 {
	//	return nil, fmt.Errorf("%s: no data found for limit %d and offset %d", op, limit, offset)
	//}

	return data, nil
}

// В идеале хранить просто список товаров, но так как по отдельности
// мы к ним на чтение не обращаемся, логика удаления отдельных товаров
// становится слишком сложной, врядли будет возможно учесть все корнер кейсы,
// так что проще удалить весь кеш, тем более мы храним всего минуту
func (s *RedisStorage) InvalidList(ctx context.Context) error {
	const op = "storage.redis.InvalidList"

	pattern := "limit:*-offset:*"
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()

	pipe := s.client.Pipeline()

	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("%s: scan error: %w", op, err)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("%s: delete keys error: %w", op, err)
	}

	return nil
}
