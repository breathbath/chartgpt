package telegram

import "breathbathChartGPT/pkg/msg"

func BuildBot(msgHandler msg.Handler) (*Bot, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	bot, err := NewBot(config, msgHandler)
	if err != nil {
		return nil, err
	}

	return bot, nil
}
