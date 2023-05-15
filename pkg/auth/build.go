package auth

import (
	"context"
)

func BuildLoginHandler(
	us *UserStorage,
) (*LoginHandler, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	validationErr := cfg.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	err = MigrateUsers(context.Background(), cfg, us)
	if err != nil {
		return nil, err
	}

	handler := NewLoginHandler(us, cfg)

	return handler, nil
}
