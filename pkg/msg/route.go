package msg

import (
	"context"
	"github.com/pkg/errors"
)

type HandlerComposite struct {
	Handlers []Handler
}

type HandlerCondition struct {
	TrueHandler  Handler
	FalseHandler Handler
}

func (ch HandlerComposite) Handle(ctx context.Context, req *Request) (*Response, error) {
	for _, h := range ch.Handlers {
		canHandle, err := h.CanHandle(ctx, req)
		if err != nil {
			return nil, err
		}
		if canHandle {
			return h.Handle(ctx, req)
		}
	}

	return nil, errors.New("no matching handler found for the message")
}

func (ch HandlerComposite) CanHandle(ctx context.Context, req *Request) (bool, error) {
	return true, nil
}

func (ch HandlerCondition) Handle(ctx context.Context, req *Request) (*Response, error) {
	canHandle, err := ch.TrueHandler.CanHandle(ctx, req)
	if err != nil {
		return nil, err
	}

	if canHandle {
		return ch.TrueHandler.Handle(ctx, req)
	}

	return ch.FalseHandler.Handle(ctx, req)
}

func (ch HandlerCondition) CanHandle(ctx context.Context, req *Request) (bool, error) {
	return true, nil
}
