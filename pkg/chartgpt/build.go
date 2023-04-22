package chartgpt

func BuildHandler() (*Handler, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	api, err := NewHandler(config)
	if err != nil {
		return nil, err
	}

	return api, nil
}
