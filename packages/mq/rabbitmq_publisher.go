package mq

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQPublisher{conn: conn, ch: ch}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	err := p.ch.PublishWithContext(ctx,
		topic,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
	if err != nil {
		log.Printf("failed to publish to %s: %v", topic, err)
	}
	return err
}

func (p *RabbitMQPublisher) Close() error {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
