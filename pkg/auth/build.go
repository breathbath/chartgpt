package auth

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
)

func BuildHandler(successHandler msg.Handler, st storage.Client) (*Handler, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	handler, err := NewHandler(successHandler, st, cfg)
	if err != nil {
		return nil, err
	}

	return handler, nil
}
