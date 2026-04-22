package repository

import "context"

type PubSubRepo interface {
	Subscribe(ctx context.Context, channel string) (<-chan string, func() error, error)
}
