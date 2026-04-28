package mq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AMQPInterface interface {
	Dial(url string) (*amqp.Connection, error)
	Channel(*amqp.Connection) (*amqp.Channel, error)
}

type RealAMQP struct{}

func (r *RealAMQP) Dial(url string) (*amqp.Connection, error) {
	return amqp.Dial(url)
}

func (r *RealAMQP) Channel(conn *amqp.Connection) (*amqp.Channel, error) {
	return conn.Channel()
}

type RabbitMQConsumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	amqp AMQPInterface
}

func NewRabbitMQConsumer(url string) (*RabbitMQConsumer, error) {
	return NewRabbitMQConsumerWithAMQP(url, &RealAMQP{})
}

func NewRabbitMQConsumerWithAMQP(url string, amqpClient AMQPInterface) (*RabbitMQConsumer, error) {
	return NewRabbitMQConsumerWithAMQPAndPrefetch(url, 1, amqpClient)
}

func NewRabbitMQConsumerWithAMQPAndPrefetch(url string, prefetchCount int, amqpClient AMQPInterface) (*RabbitMQConsumer, error) {
	conn, err := amqpClient.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := amqpClient.Channel(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQConsumer{conn: conn, ch: ch, amqp: amqpClient}, nil
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			go func(m amqp.Delivery) {
				msgCtx, cancel := context.WithCancel(ctx)
				defer cancel()

				if err := handler(msgCtx, m.Body); err != nil {
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

type RabbitMQPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	amqp AMQPInterface
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	return NewRabbitMQPublisherWithAMQP(url, &RealAMQP{})
}

func NewRabbitMQPublisherWithAMQP(url string, amqpClient AMQPInterface) (*RabbitMQPublisher, error) {
	conn, err := amqpClient.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := amqpClient.Channel(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQPublisher{conn: conn, ch: ch, amqp: amqpClient}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	err := p.ch.PublishWithContext(ctx,
		"task_exchange",
		topic,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
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