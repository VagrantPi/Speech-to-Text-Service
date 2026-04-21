package mq

import "context"

type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}
