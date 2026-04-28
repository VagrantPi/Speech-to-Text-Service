package stt

import (
	"context"
	"fmt"

	"github.com/go-redis/redis_rate/v10"
	"github.com/sashabaranov/go-openai"
)

type RateLimiterInterface interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

type STTRepoInterface interface {
	Transcribe(ctx context.Context, localFilePath string) (string, error)
}

type OpenAISTTService struct {
	client  *openai.Client
	limiter RateLimiterInterface
	rpm     int
}

var _ STTRepoInterface = (*OpenAISTTService)(nil)

func NewOpenAISTTService(apiKey string) (*OpenAISTTService, error) {
	return NewOpenAISTTServiceWithLimiter(apiKey, nil, 50)
}

func NewOpenAISTTServiceWithLimiter(apiKey string, limiter RateLimiterInterface, rpm int) (*OpenAISTTService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	return &OpenAISTTService{
		client: openai.NewClient(apiKey),
		limiter: limiter,
		rpm:    rpm,
	}, nil
}

func (s *OpenAISTTService) Transcribe(ctx context.Context, localFilePath string) (string, error) {
	if s.limiter != nil {
		res, err := s.limiter.Allow(ctx, "rate_limit:openai:stt", redis_rate.PerMinute(s.rpm))
		if err != nil {
			return "", fmt.Errorf("failed to check rate limit: %w", err)
		}
		if res.Allowed == 0 {
			return "", &RateLimitedError{}
		}
	}

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: localFilePath,
	}
	resp, err := s.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}
