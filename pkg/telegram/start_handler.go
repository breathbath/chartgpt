package telegram

import (
	"context"
	"strings"

	"breathbathChatGPT/pkg/msg"
)

type StartHandler struct{}

func (sh *StartHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, "/start"), nil
}

func (sh *StartHandler) Handle(context.Context, *msg.Request) (*msg.Response, error) {
	return &msg.Response{
		Message: "The bot is already started",
		Type:    msg.Success,
	}, nil
}
