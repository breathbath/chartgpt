package telegram

import (
	"breathbathChatGPT/pkg/msg"
	"gorm.io/gorm"
)

func BuildBot(r *msg.Router, dbConn *gorm.DB) (*Bot, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	bot, err := NewBot(config, r, dbConn)
	if err != nil {
		return nil, err
	}

	return bot, nil
}
