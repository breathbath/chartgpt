package telegram

import "breathbathChatGPT/pkg/msg"

func BuildBot(r *msg.Router) (*Bot, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	bot, err := NewBot(config, r)
	if err != nil {
		return nil, err
	}

	return bot, nil
}
