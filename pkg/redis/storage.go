package redis

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	base "github.com/redis/go-redis/v9"
	"time"
)

type Storage struct {
	baseClient *base.Client
}

func NewStorage(cfg *Config) (*Storage, error) {
	err := cfg.Validate()
	if err.HasErrors() {
		return nil, err
	}

	rdb := base.NewClient(&base.Options{
		Addr:     cfg.Addr,
		Password: cfg.Pass,
		DB:       0,
	})

	redisCheckErr := checkRedis(rdb)
	if redisCheckErr != nil {
		return nil, redisCheckErr
	}

	return &Storage{baseClient: rdb}, nil
}

func checkRedis(cl *base.Client) error {
	operation := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		status := cl.Ping(ctx)
		return status.Err()
	}

	err := backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		return errors.Wrap(err, "failed to connect to redis")
	}

	return nil
}

func (s *Storage) Read(ctx context.Context, key string) (raw []byte, found bool, err error) {
	val, err := s.baseClient.Get(ctx, key).Result()
	if err != nil {
		if err == base.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	return []byte(val), true, nil
}

func (s *Storage) Write(ctx context.Context, key string, raw []byte, exp time.Duration) error {
	return s.baseClient.Set(ctx, key, raw, exp).Err()
}
