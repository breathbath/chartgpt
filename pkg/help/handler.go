package help

import (
	"context"
	"strings"

	"breathbathChatGPT/pkg/msg"
)

type Provider interface {
	GetHelp(ctx context.Context, req *msg.Request) string
}

type Handler struct {
	Providers []Provider
}

func (ch *Handler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, "/help"), nil
}

func (ch *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	help := `list of available commands
/help: to show this help`

	for _, prov := range ch.Providers {
		curHelp := prov.GetHelp(ctx, req)
		if curHelp == "" {
			continue
		}

		help += "\n" + curHelp
	}

	return &msg.Response{
		Message: help,
		Type:    msg.Success,
	}, nil
}
