package chartgpt

var api *Handler

func BuildHandler() (*Handler, error) {
	if api != nil {
		return api, nil
	}

	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	api, err = NewHandler(config)
	if err != nil {
		return nil, err
	}

	return api, nil
}
