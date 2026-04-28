package redis

import (
	"context"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Password string `mapstructure:"REDIS_PASSWORD"`
}

type RateLimiterInterface interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

type RedisClient struct {
	client  *redis.Client
	limiter *redis_rate.Limiter
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

	limiter := redis_rate.NewLimiter(client)
	return &RedisClient{client: client, limiter: limiter}, nil
}

func (c *RedisClient) GetLimiter() RateLimiterInterface {
	return c.limiter
}

func (c *RedisClient) Publish(ctx context.Context, channel string, payload string) error {
	return c.client.Publish(ctx, channel, payload).Err()
}

func (c *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.client.Subscribe(ctx, channel)
}
