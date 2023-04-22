package chartgpt

import (
	"breathbathChartGPT/pkg/errs"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ApiKey string `envconfig:"CHARTGPT_API_KEY"`
	Model  string `envconfig:"CHARTGPT_MODEL"`
	Role   string `envconfig:"CHARTGPT_COMPLETIONS_ROLE"`
}

func (c *Config) Validate() *errs.Multi {
	e := errs.NewMulti()

	if c.ApiKey == "" {
		e.Err("CHARTGPT_API_KEY cannot be empty")
	}
	if c.Model == "" {
		e.Err("CHARTGPT_MODEL cannot be empty")
	}

	if c.ApiKey == "" {
		e.Err("CHARTGPT_COMPLETIONS_ROLE cannot be empty")
	}

	return e
}

func LoadConfig() (cfg *Config, err error) {
	err = envconfig.Process("chartgpt", cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
