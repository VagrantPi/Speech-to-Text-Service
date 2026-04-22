package repository

import (
	"context"

	redisclient "speech.local/packages/redis"
)

type PubSubRepoImpl struct {
	redisClient *redisclient.RedisClient
}

func NewPubSubRepo(redisClient *redisclient.RedisClient) PubSubRepo {
	return &PubSubRepoImpl{redisClient: redisClient}
}

func (r *PubSubRepoImpl) Subscribe(ctx context.Context, channel string) (<-chan string, func() error, error) {
	pubsub := r.redisClient.Subscribe(ctx, channel)

	ch := make(chan string, 1)

	go func() {
		for msg := range pubsub.Channel() {
			select {
			case ch <- msg.Payload:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, pubsub.Close, nil
}
