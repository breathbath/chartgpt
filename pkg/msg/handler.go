package msg

import "context"

const CommandPrefix = "/"

type Handler interface {
	Handle(ctx context.Context, req *Request) (*Response, error)
	CanHandle(ctx context.Context, req *Request) (bool, error)
}
