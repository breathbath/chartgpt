package telegram

import (
	"context"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
)

type StartHandler struct{}

func (sh *StartHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return utils.MatchesCommand(req.Message, "/start"), nil
}

func (sh *StartHandler) Handle(context.Context, *msg.Request) (*msg.Response, error) {
	return &msg.Response{
		Message: "The bot is already started",
		Type:    msg.Success,
	}, nil
}
