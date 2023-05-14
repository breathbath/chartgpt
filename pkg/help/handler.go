package help

import (
	"breathbathChatGPT/pkg/msg"
	"context"
	"strings"
)

type Provider interface {
	GetHelp() string
}

type Handler struct {
	Providers []Provider
}

func (ch *Handler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, "/help"), nil
}

func (ch *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	help := `list of available commands
/help: to show this help`
	for _, prov := range ch.Providers {
		help += "\n" + prov.GetHelp()
	}

	return &msg.Response{
		Message: help,
		Type:    msg.Success,
	}, nil
}
