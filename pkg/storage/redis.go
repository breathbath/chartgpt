package storage

import (
	"breathbathChatGPT/pkg/errs"
	"breathbathChatGPT/pkg/utils"
	"context"
	"encoding/json"
	"github.com/cenkalti/backoff/v4"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	base "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"time"
)

const IsNotLoggableContentCtxKey = "is_not_loggable"

type RedisConfig struct {
	Addr string `envconfig:"REDIS_ADDR"`
	Pass string `envconfig:"REDIS_PASS"`
}

func (c *RedisConfig) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.Addr == "" {
		e.Err("REDIS_ADDR cannot be empty")
	}

	return e
}

func LoadConfig() (cfg *RedisConfig, err error) {
	cfg = new(RedisConfig)
	err = envconfig.Process("redis", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load redis config")
	}

	return cfg, nil
}

type RedisClient struct {
	baseClient *base.Client
}

func NewClient(cfg *RedisConfig) (*RedisClient, error) {
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
	return &RedisClient{baseClient: rdb}, nil
}

func checkRedis(cl *base.Client) error {
	logrus.Infof("will ping redis")
	operation := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		status := cl.Ping(ctx)
		err := status.Err()
		if err != nil {
			logrus.Errorf("Failed to connect to redis: %v", err)
			return err
		}

		return nil
	}

	err := backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		return errors.Wrap(err, "failed to connect to redis")
	}

	return nil
}

func (c *RedisClient) Read(ctx context.Context, key string) (raw []byte, found bool, err error) {
	log := logrus.WithContext(ctx)

	val, err := c.baseClient.Get(ctx, key).Result()

	if err != nil {
		if err == base.Nil {
			log.Infof("nothing found in redis under key %q", key)
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "failed to get data from redis under key %q", key)
	}

	if ctx.Value(IsNotLoggableContentCtxKey) != nil {
		log.Debugf("successfully read data to redis under key %q", key)
	} else {
		log.Debugf("successfully read data %q from redis under key %q", string(raw), key)
	}

	return []byte(val), true, nil
}

func (c *RedisClient) Write(ctx context.Context, key string, raw []byte, exp time.Duration) error {
	log := logrus.WithContext(ctx)

	err := c.baseClient.Set(ctx, key, raw, exp).Err()
	if err != nil {
		return errors.Wrapf(err, "failed to write data to redis under key %q", key)
	}

	if ctx.Value(IsNotLoggableContentCtxKey) != nil {
		log.Debugf("wrote hidden data to redis under key %q", key)
	} else {
		log.Debugf("wrote data %q to redis under key %q", string(raw), key)
	}

	return nil
}

func (c *RedisClient) Delete(ctx context.Context, key string) error {
	log := logrus.WithContext(ctx)

	err := c.baseClient.Del(ctx, key).Err()

	if err != nil {
		return errors.Wrapf(err, "failed to delete data from redis under key %q", key)
	}

	log.Infof("deleted data from redis under key %q", key)
	return nil
}

func (c *RedisClient) Load(ctx context.Context, key string, target interface{}) (found bool, err error) {
	log := logrus.WithContext(ctx)

	targetType := utils.GetType(target)
	log.Debugf("will load %q for key %s", targetType, key)

	rawData, found, err := c.Read(ctx, key)
	if err != nil {
		return false, errors.Wrap(err, "failed to read data from storage")
	}

	if !found {
		return false, nil
	}

	err = json.Unmarshal(rawData, target)
	if err != nil {
		return false, errors.Wrapf(err, "failed to convert %q to %s", targetType, string(rawData))
	}

	return true, nil
}

func (c *RedisClient) Save(ctx context.Context, key string, data interface{}, validity time.Duration) error {
	rawBytes, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "failed to convert %q to json", utils.GetType(data))
	}

	return c.Write(ctx, key, rawBytes, validity)
}

func (c *RedisClient) FindKeys(ctx context.Context, pattern string) (keys []string, err error) {
	val := c.baseClient.Keys(ctx, pattern)
	if val.Err() != nil {
		return nil, errors.Wrapf(err, "failed to find keys in redis by pattern %q", pattern)
	}

	return val.Val(), nil
}
