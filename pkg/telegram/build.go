package telegram

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"encoding/json"
	"gorm.io/gorm"
)

func BuildBot(r *msg.Router, dbConn *gorm.DB, cache storage.Client, delayedMessagesCallback func(input json.RawMessage) ([]msg.ResponseMessage, error)) (*Bot, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	bot, err := NewBot(config, r, dbConn, cache, delayedMessagesCallback)
	if err != nil {
		return nil, err
	}

	return bot, nil
}
