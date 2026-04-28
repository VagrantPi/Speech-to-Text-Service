package llm

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-redis/redis_rate/v10"
	"github.com/sashabaranov/go-openai"
)

var systemPrompt = "你是一個專業的語音摘要助手。請根據以下逐字稿，整理出條理分明的重點摘要。"

type RateLimiterInterface interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

type LLMRepoInterface interface {
	GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (fullSummary string, err error)
}

type OpenAIStreamService struct {
	client  *openai.Client
	limiter RateLimiterInterface
	rpm     int
}

var _ LLMRepoInterface = (*OpenAIStreamService)(nil)

func NewOpenAIStreamService(apiKey string) (*OpenAIStreamService, error) {
	return NewOpenAIStreamServiceWithLimiter(apiKey, nil, 500)
}

func NewOpenAIStreamServiceWithLimiter(apiKey string, limiter RateLimiterInterface, rpm int) (*OpenAIStreamService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	return &OpenAIStreamService{
		client: openai.NewClient(apiKey),
		limiter: limiter,
		rpm:    rpm,
	}, nil
}

func (s *OpenAIStreamService) GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (string, error) {
	if s.limiter != nil {
		res, err := s.limiter.Allow(ctx, "rate_limit:openai:llm", redis_rate.PerMinute(s.rpm))
		if err != nil {
			return "", fmt.Errorf("failed to check rate limit: %w", err)
		}
		if res.Allowed == 0 {
			return "", &RateLimitedError{}
		}
	}

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: transcript},
		},
		Stream: true,
	}

	stream, err := s.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var fullSummary strings.Builder

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
			content := response.Choices[0].Delta.Content
			tokenChan <- content
			fullSummary.WriteString(content)
		}
	}

	close(tokenChan)
	return fullSummary.String(), nil
}
