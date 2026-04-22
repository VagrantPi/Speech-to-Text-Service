package repository

import "context"

type PubSubRepo interface {
	Publish(ctx context.Context, channel string, payload string) error
}
