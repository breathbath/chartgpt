package telegram

import (
	"breathbathChartGPT/pkg/errs"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	APIToken string `envconfig:"TELEGRAM_ACCESS_TOKEN"`
}

func (c *Config) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.APIToken == "" {
		e.Err("TELEGRAM_ACCESS_TOKEN cannot be empty")
	}

	return e
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("telegram", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load telegram config")
	}

	return cfg, nil
}
