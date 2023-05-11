package storage

func BuildRedisClient() (*RedisClient, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	c, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return c, nil
}
