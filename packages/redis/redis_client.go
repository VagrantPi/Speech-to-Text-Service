package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Password string `mapstructure:"REDIS_PASSWORD"`
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(cfg Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host,
		Password: cfg.Password,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{client: client}, nil
}

func (c *RedisClient) Publish(ctx context.Context, channel string, payload string) error {
	return c.client.Publish(ctx, channel, payload).Err()
}

func (c *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.client.Subscribe(ctx, channel)
}
