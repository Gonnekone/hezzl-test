package storage

import (
	"github.com/Gonnekone/hezzl-test/core/internal/config"
	"github.com/Gonnekone/hezzl-test/core/internal/storage/postgres"
	"github.com/Gonnekone/hezzl-test/core/internal/storage/redis"
)

type Storage struct {
	*postgres.PostgresStorage
	*redis.RedisStorage
}

func New(
	pCfg config.PostgresStorage,
	rCfg config.RedisStorage,
) (*Storage, error) {
	postgresStorage, err := postgres.New(pCfg)
	if err != nil {
		return nil, err
	}

	redisStorage, err := redis.New(rCfg)
	if err != nil {
		return nil, err
	}

	return &Storage{
		PostgresStorage: postgresStorage,
		RedisStorage:    redisStorage,
	}, nil
}
