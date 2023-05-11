package storage

import (
	"context"
	"time"
)

type Client interface {
	Read(ctx context.Context, key string) (raw []byte, found bool, err error)
	Write(ctx context.Context, key string, raw []byte, exp time.Duration) error
	Delete(ctx context.Context, key string) error
	Load(ctx context.Context, key string, target interface{}) (found bool, err error)
	Save(ctx context.Context, key string, data interface{}, validity time.Duration) error
}
