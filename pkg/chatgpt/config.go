package chatgpt

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"breathbathChatGPT/pkg/errs"
)

type Config struct {
	APIKey       string `envconfig:"CHATGPT_API_KEY"`
	DefaultModel string `envconfig:"CHATGPT_DEFAULT_MODEL"`
	ScopedMode   bool   `envconfig:"CHATGPT_SCOPED_MODE"`
}

func (c *Config) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.APIKey == "" {
		e.Errf("CHATGPT_API_KEY cannot be empty")
	}
	if c.DefaultModel == "" {
		e.Errf("CHATGPT_DEFAULT_MODEL cannot be empty")
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
