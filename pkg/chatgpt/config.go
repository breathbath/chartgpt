package chatgpt

import (
	"breathbathChatGPT/pkg/errs"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	ApiKey       string `envconfig:"CHATGPT_API_KEY"`
	DefaultModel string `envconfig:"CHATGPT_DEFAULT_MODEL"`
}

func (c *Config) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.ApiKey == "" {
		e.Err("CHATGPT_API_KEY cannot be empty")
	}
	if c.DefaultModel == "" {
		e.Err("CHATGPT_DEFAULT_MODEL cannot be empty")
	}

	return e
}

func LoadConfig() (cfg *Config, err error) {
	cfg = new(Config)

	err = envconfig.Process("chatgpt", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chatgpt config")
	}

	return cfg, nil
}
