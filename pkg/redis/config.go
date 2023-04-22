package redis

import (
	"breathbathChartGPT/pkg/errs"
	"github.com/kelseyhightower/envconfig"
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
	err = envconfig.Process("redis", cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
