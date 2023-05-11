package chatgpt

import "breathbathChatGPT/pkg/storage"

func BuildChatCompletionHandler(db storage.Client) (h *ChatCompletionHandler, help string, err error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, "", err
	}

	loader := &Loader{
		db:  db,
		cfg: config,
	}

	commands := BuildCommandHandlers(db, config, loader)

	chatCompletionHandler, help, err := NewChatCompletionHandler(config, db, commands, loader)
	if err != nil {
		return nil, "", err
	}

	return chatCompletionHandler, help, nil
}
