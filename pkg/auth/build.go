package auth

import "breathbathChartGPT/pkg/msg"

func BuildHandler(successHandler msg.Handler, storage Storage) (*Handler, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	handler, err := NewHandler(successHandler, storage, cfg)
	if err != nil {
		return nil, err
	}

	return handler, nil
}
