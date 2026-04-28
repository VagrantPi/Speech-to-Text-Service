package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/go-redis/redis_rate/v10"
	"go.uber.org/zap"
	"speech.local/packages/errors"
)

var systemPrompt = "你是一個專業的語音摘要助手。請根據以下逐字稿，整理出條理分明的重點摘要。"

var llmLogger *zap.Logger

func init() {
	var err error
	llmLogger, err = zap.NewProduction()
	if err != nil {
		llmLogger = zap.NewNop()
	}
}

type RateLimiterInterface interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

type LLMRepoInterface interface {
	GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (fullSummary string, err error)
}

type OpenAIStreamService struct {
	httpClient *http.Client
	limiter    RateLimiterInterface
	rpm        int
	apiKey     string
}

var _ LLMRepoInterface = (*OpenAIStreamService)(nil)

func NewOpenAIStreamService(apiKey string) (*OpenAIStreamService, error) {
	return NewOpenAIStreamServiceWithLimiter(apiKey, nil, 500)
}

func NewOpenAIStreamServiceWithLimiter(apiKey string, limiter RateLimiterInterface, rpm int) (*OpenAIStreamService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	
	if limiter == nil {
		limiter = &noopRateLimiter{}
	}

	return &OpenAIStreamService{
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
		limiter: limiter,
		rpm:     rpm,
		apiKey:  apiKey,
	}, nil
}

type noopRateLimiter struct{}

func (n *noopRateLimiter) Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error) {
	return &redis_rate.Result{Allowed: 1}, nil
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

	apiURL := "https://api.openai.com/v1/chat/completions"
	if os.Getenv("OLLAMA_API_URL") != "" {
		apiURL = os.Getenv("OLLAMA_API_URL") + "/v1/chat/completions"
	}

	reqBody := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": transcript},
		},
		"stream": true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	var streamErr error
	var fullSummary string

	err = retry.Do(
		func() error {
			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+s.apiKey)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")

			resp, err := s.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := resp.Header.Get("Retry-After")
				llmLogger.Warn("OpenAI API 429 Rate Limited", zap.String("retry_after", retryAfter))
				return retry.Unrecoverable(errors.ErrThirdPartyRateLimited)
			}

			if resp.StatusCode >= 500 {
				respBody, _ := io.ReadAll(resp.Body)
				llmLogger.Warn("OpenAI API 5xx error", zap.Int("status_code", resp.StatusCode), zap.ByteString("body", respBody))
				return fmt.Errorf("%w: status code %d", errors.ErrThirdPartyServer, resp.StatusCode)
			}

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
			}

			fullSummary, streamErr = s.readStreamResponse(resp.Body, tokenChan)
			return streamErr
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			llmLogger.Warn("Retrying HTTP request to OpenAI LLM", zap.Uint("attempt", n), zap.Error(err))
		}),
	)

	if err != nil {
		return "", err
	}

	return fullSummary, nil
}

func (s *OpenAIStreamService) readStreamResponse(body io.Reader, tokenChan chan<- string) (string, error) {
	var fullSummary strings.Builder
	decoder := ioReaderToStringReader(body)

	for {
		line, err := decoder.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			tokenChan <- content
			fullSummary.WriteString(content)
		}
	}

	close(tokenChan)
	return fullSummary.String(), nil
}

type stringReader struct {
	reader io.Reader
}

func ioReaderToStringReader(r io.Reader) *stringReader {
	return &stringReader{reader: r}
}

func (sr *stringReader) ReadString(delim byte) (string, error) {
	var buf bytes.Buffer
	for {
		b := make([]byte, 1)
		n, err := sr.reader.Read(b)
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				return buf.String(), nil
			}
			return "", err
		}
		if n == 0 {
			continue
		}
		if b[0] == delim {
			return buf.String(), nil
		}
		buf.WriteByte(b[0])
	}
}