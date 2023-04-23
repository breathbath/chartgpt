package msg

import (
	"context"
	"fmt"
)

const CommandPrefix = "/"

type CommandsHandler struct {
	PassHandler Handler
}

func (ch *CommandsHandler) Handle(ctx context.Context, req *Request) (*Response, error) {
	if !IsCommand(req.Message) {
		return ch.PassHandler.Handle(ctx, req)
	}

	if MatchCommand(req.Message, []string{"help"}) {
		return ch.handleHelp(ctx, req)
	}

	if MatchCommand(req.Message, []string{"start"}) {
		return ch.handleStart(ctx, req)
	}

	return ch.handleUnrecognisedCommand(ctx, req)
}

func (ch *CommandsHandler) handleHelp(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		Message: `list of available commands:
/logout to logout the current user
/help to show this help
`,
		Type: Success,
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
