package telegram

import (
	"breathbathChatGPT/pkg/msg"
	"context"
	"strings"
)

type StartHandler struct{}

func (sh *StartHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, "/start"), nil
}

func (sh *StartHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	return &msg.Response{
		Message: "The bot is already started",
		Type:    msg.Success,
	}, nil
}
