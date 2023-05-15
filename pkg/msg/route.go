package msg

import (
	"context"
	"github.com/pkg/errors"
)

type Middleware interface {
	Handle(ctx context.Context, req *Request) (*Response, error)
}

type Router struct {
	Handlers    []Handler
	Middlewares []Middleware
}

func (ch *Router) UseMiddleware(m Middleware) {
	ch.Middlewares = append(ch.Middlewares, m)
}

func (ch *Router) Route(ctx context.Context, req *Request) (*Response, error) {
	for _, h := range ch.Handlers {
		for _, m := range ch.Middlewares {
			resp, err := m.Handle(ctx, req)
			if err != nil {
				return nil, err
			}

			if resp != nil {
				return resp, nil
			}
		}

		canHandle, err := h.CanHandle(ctx, req)
		if err != nil {
			return nil, err
		}
		if !canHandle {
			continue
		}

		return h.Handle(ctx, req)
	}

	return nil, errors.New("no matching handler found for the message")
}
