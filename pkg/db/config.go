package db

import (
	"breathbathChatGPT/pkg/errs"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	ConnString string `envconfig:"MYSQL_CONN_STRING"`
}

func (c *Config) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.ConnString == "" {
		e.Errf("MYSQL_CONN_STRING cannot be empty")
	}

	return e
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)
	err = envconfig.Process("mysql", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load mysql config")
	}

	return cfg, nil
}
