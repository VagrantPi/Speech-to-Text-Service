package repository

import "context"

// Publisher 是 outbox-relay 專屬的微型介面
// 它只關心能把 payload 丟到指定 topic，不在乎底層是 RabbitMQ 還是 Kafka
type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}
