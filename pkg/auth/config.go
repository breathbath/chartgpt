package auth

import (
	"breathbathChatGPT/pkg/errs"
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"time"
)

type RawConfig struct {
	SessionDuration string `envconfig:"AUTH_SESSION_DURATION"`
	Users           string `envconfig:"AUTH_USERS"`
}

type Config struct {
	SessionDuration time.Duration
	Users           []ConfiguredUser
}

func (c *Config) Validate() *errs.Multi {
	multiErr := errs.NewMulti()
	if len(c.Users) == 0 {
		multiErr.Err("AUTH_USERS cannot be empty")
	}

	for _, u := range c.Users {
		multiErr.Add(u.Validate())
	}

	return multiErr
}

func (rc *RawConfig) ToConfig() (*Config, *errs.Multi) {
	multiErr := errs.NewMulti()

	cfg := &Config{}

	if rc.SessionDuration != "" {
		sessionDur, err := time.ParseDuration(rc.SessionDuration)
		if err != nil {
			multiErr.Add(errors.Wrapf(err, "failed to parse duration %q", rc.SessionDuration))
		} else {
			cfg.SessionDuration = sessionDur
		}
	}

	if rc.Users != "" {
		usersFromConfig := []ConfiguredUser{}
		err := json.Unmarshal([]byte(rc.Users), &usersFromConfig)
		if err != nil {
			multiErr.Add(errors.Wrapf(err, "failed to parse users from JSON format %q", rc.Users))
		} else {
			cfg.Users = usersFromConfig
		}
	}

	return cfg, multiErr
}

func LoadConfig() (*Config, error) {
	rawCfg := new(RawConfig)
	err := envconfig.Process("auth", rawCfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load auth config")
	}

	cfg, convErr := rawCfg.ToConfig()
	if convErr.HasErrors() {
		return nil, convErr
	}

	return cfg, nil
}
