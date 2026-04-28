package stt

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/stretchr/testify/assert"
)

type mockLimiter struct {
	allowFn func(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error) {
	return m.allowFn(ctx, key, limit)
}

func TestOpenAISTTService_Transcribe_WithRateLimiter(t *testing.T) {
	limiter := &mockLimiter{
		allowFn: func(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error) {
			return &redis_rate.Result{Allowed: 0, Remaining: 0, ResetAfter: 0}, nil
		},
	}

	svc, err := NewOpenAISTTServiceWithLimiter("test-key", limiter, 50)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = svc.Transcribe(ctx, "/nonexistent/file.wav")

	assert.Error(t, err)
	assert.IsType(t, &RateLimitedError{}, err)
}

func TestOpenAISTTService_Transcribe_RateLimiterAllows(t *testing.T) {
	limiter := &mockLimiter{
		allowFn: func(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error) {
			return &redis_rate.Result{Allowed: 1, Remaining: 0, ResetAfter: 0}, nil
		},
	}

	svc, err := NewOpenAISTTServiceWithLimiter("test-key", limiter, 50)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = svc.Transcribe(ctx, "/nonexistent/file.wav")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestRateLimitedError_IsRateLimited(t *testing.T) {
	err := &RateLimitedError{}
	assert.True(t, err.IsRateLimited())
	assert.Equal(t, "rate limited", err.Error())
}

func TestParseTranscriptionResponse(t *testing.T) {
	svc := &OpenAISTTService{}

	tests := []struct {
		name     string
		body    string
		want    string
		wantErr bool
	}{
		{
			name:     "valid response",
			body:     `{"text": "Hello world"}`,
			want:    "Hello world",
			wantErr: false,
		},
		{
			name:     "empty text",
			body:     `{"text": ""}`,
			want:    "",
			wantErr: false,
		},
		{
			name:     "invalid json",
			body:     `{invalid}`,
			want:    "",
			wantErr: true,
		},
		{
			name:     "missing text field",
			body:     `{}`,
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.parseTranscriptionResponse([]byte(tt.body))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}