package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFakeRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
}

func TestRedisClient_Publish(t *testing.T) {
	mr, client := newFakeRedis(t)
	defer client.Close()
	defer mr.Close()

	r := &RedisClient{client: client}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		err := r.Publish(ctx, "test-channel", `{"message":"hello"}`)
		assert.NoError(t, err)
	})
}

func TestRedisClient_Subscribe(t *testing.T) {
	mr, client := newFakeRedis(t)
	defer client.Close()
	defer mr.Close()

	r := &RedisClient{client: client}
	ctx := context.Background()

	t.Run("success - receive message", func(t *testing.T) {
		pubsub := r.Subscribe(ctx, "test-channel")
		require.NotNil(t, pubsub)

		err := client.Publish(ctx, "test-channel", "hello").Err()
		require.NoError(t, err)

		msg, err := pubsub.ReceiveMessage(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "hello", msg.Payload)

		pubsub.Close()
	})
}