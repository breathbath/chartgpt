package msg

import (
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"strings"
)

const CommandPrefix = "/"

type CommandsHandler struct {
	PassHandler Handler
	DynamicHelp string
}

func (ch *CommandsHandler) Handle(ctx context.Context, req *Request) (*Response, error) {
	if !strings.HasPrefix(req.Message, CommandPrefix) {
		return ch.PassHandler.Handle(ctx, req)
	}

	if utils.MatchesAny(req.Message, CommandPrefix, []string{"help"}) {
		return ch.handleHelp(ctx, req)
	}

	if utils.MatchesAny(req.Message, CommandPrefix, []string{"start"}) {
		return ch.handleStart(ctx, req)
	}

	return ch.handleUnrecognisedCommand(ctx, req)
}

func (ch *CommandsHandler) handleHelp(ctx context.Context, req *Request) (*Response, error) {
	msg := `list of available commands:
/logout to logout the current user
/help to show this help
`
	return &Response{
		Message: msg + "\n" + ch.DynamicHelp,
		Type:    Success,
	}, nil
}

func (ch *CommandsHandler) handleStart(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		Message: "The bot is already started",
		Type:    Success,
	}, nil
}

func (ch *CommandsHandler) handleUnrecognisedCommand(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		Message: fmt.Sprintf("unsupported command %q", req.Message),
		Type:    Error,
	}, nil
}
