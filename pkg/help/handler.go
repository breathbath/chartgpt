package help

import (
	"context"
	"fmt"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"
)

type Result struct {
	Text, PredefinedOption string
}

const helpCommand = "/help"

type Provider interface {
	GetHelp(ctx context.Context, req *msg.Request) Result
}

type Handler struct {
	Providers     []Provider
	AdminDetector func(req *msg.Request) bool
	ModeDetector  func() bool
}

func NewHandler(modeDetector func() bool, adminDetector func(req *msg.Request) bool, providers []Provider) *Handler {
	return &Handler{
		Providers:     providers,
		AdminDetector: adminDetector,
		ModeDetector:  modeDetector,
	}
}

func (ch *Handler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, helpCommand) {
		return false, nil
	}

	if ch.ModeDetector() && !ch.AdminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (ch *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	op := &msg.Options{}
	op.WithPredefinedResponse(helpCommand)

	help := fmt.Sprintf(`List of available commands

%s: to show this help`, helpCommand)

	for _, prov := range ch.Providers {
		helpResult := prov.GetHelp(ctx, req)

		if helpResult.PredefinedOption != "" {
			op.WithPredefinedResponse(helpResult.PredefinedOption)
		}

		if helpResult.Text == "" {
			continue
		}

		help += "\n\n" + helpResult.Text
	}
	help += "\n"

	return &msg.Response{
		Message: help,
		Type:    msg.Success,
		Options: op,
	}, nil
}
