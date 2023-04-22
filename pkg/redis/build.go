package redis

func BuildStorage() (*Storage, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	storage, err := NewStorage(cfg)
	if err != nil {
		return nil, err
	}

	return storage, nil
}
