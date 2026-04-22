package mq

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQConsumer(url string) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQConsumer{conn: conn, ch: ch}, nil
}

func (c *RabbitMQConsumer) Consume(ctx context.Context, queueName string, handler func(ctx context.Context, payload []byte) error) error {
	msgs, err := c.ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf("Listening on queue: %s", queueName)

	// TODO: 併發數量先寫死，為可優化空間
	sem := make(chan struct{}, 10)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			sem <- struct{}{}
			go func(m amqp.Delivery) {
				defer func() { <-sem }()
				if err := handler(ctx, m.Body); err != nil {
					m.Nack(false, false)
				} else {
					m.Ack(false)
				}
			}(msg)
		}
	}
}

func (c *RabbitMQConsumer) Close() error {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
