package worker

import "context"

type MQConsumer interface {
	Consume(ctx context.Context, queueName string, handler func(ctx context.Context, payload []byte) error) error
}
