package auth

import (
	"breathbathChatGPT/pkg/storage"
	"context"
)

func BuildLoginHandler(
	db storage.Client,
) (*LoginHandler, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	validationErr := cfg.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	err = MigrateUsers(context.Background(), cfg, db)
	if err != nil {
		return nil, err
	}

	handler := NewLoginHandler(db, cfg)

	return handler, nil
}
