package redis

import (
	"breathbathChartGPT/pkg/errs"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	Addr string `envconfig:"REDIS_ADDR"`
	Pass string `envconfig:"REDIS_PASS"`
}

func (c *Config) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.Addr == "" {
		e.Err("REDIS_ADDR cannot be empty")
	}

	return e
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("redis", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load redis config")
	}

	return cfg, nil
}
