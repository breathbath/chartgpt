package redis

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	base "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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
		logrus.Errorf("failed to ping redis %q", cfg.Addr)
		return nil, redisCheckErr
	}

	logrus.Infof("ping to redis %q is successful", cfg.Addr)
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
	log := logrus.WithContext(ctx)

	val, err := s.baseClient.Get(ctx, key).Result()

	if err != nil {
		if err == base.Nil {
			log.Infof("nothing found in redis under key %q", key)
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "failed to get data from redis under key %q", key)
	}

	log.Infof("got data from redis under key %q", key)

	return []byte(val), true, nil
}

func (s *Storage) Write(ctx context.Context, key string, raw []byte, exp time.Duration) error {
	log := logrus.WithContext(ctx)

	err := s.baseClient.Set(ctx, key, raw, exp).Err()
	if err != nil {
		return errors.Wrapf(err, "failed to write data to redis under key %q", key)
	}

	log.Infof("wrote data to redis under key %q", key)

	return nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	log := logrus.WithContext(ctx)

	err := s.baseClient.Del(ctx, key).Err()

	if err != nil {
		return errors.Wrapf(err, "failed to delete data from redis under key %q", key)
	}

	log.Infof("deleted data from redis under key %q", key)
	return nil
}
