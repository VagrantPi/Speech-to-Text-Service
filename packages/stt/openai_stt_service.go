package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/go-redis/redis_rate/v10"
	"go.uber.org/zap"
	"speech.local/packages/errors"
)

var sttLogger *zap.Logger

func init() {
	var err error
	sttLogger, err = zap.NewProduction()
	if err != nil {
		sttLogger = zap.NewNop()
	}
}

type RateLimiterInterface interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

type STTRepoInterface interface {
	Transcribe(ctx context.Context, localFilePath string) (string, error)
}

type OpenAISTTService struct {
	httpClient *http.Client
	limiter    RateLimiterInterface
	rpm        int
	apiKey     string
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
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		limiter: limiter,
		rpm:     rpm,
		apiKey:  apiKey,
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

	file, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	apiURL := "https://api.openai.com/v1/audio/transcriptions"
	if os.Getenv("WHISPER_API_URL") != "" {
		apiURL = os.Getenv("WHISPER_API_URL") + "/v1/audio/transcriptions"
	}

	var transcript string
	err = retry.Do(
		func() error {
			body, contentType, err := s.createMultipartBody(file)
			if err != nil {
				return err
			}

			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, body)
			if err != nil {
				return err
			}

			req.Header.Set("Authorization", "Bearer "+s.apiKey)
			req.Header.Set("Content-Type", contentType)

			resp, err := s.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := resp.Header.Get("Retry-After")
				sttLogger.Warn("OpenAI API 429 Rate Limited", zap.String("retry_after", retryAfter))
				return retry.Unrecoverable(errors.ErrThirdPartyRateLimited)
			}

			if resp.StatusCode >= 500 {
				respBody, _ := io.ReadAll(resp.Body)
				sttLogger.Warn("OpenAI API 5xx error", zap.Int("status_code", resp.StatusCode), zap.ByteString("body", respBody))
				return fmt.Errorf("%w: status code %d", errors.ErrThirdPartyServer, resp.StatusCode)
			}

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			transcript, err = s.parseTranscriptionResponse(respBody)
			return err
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			sttLogger.Warn("Retrying HTTP request to OpenAI STT", zap.Uint("attempt", n), zap.Error(err))
		}),
	)

	if err != nil {
		return "", err
	}

	return transcript, nil
}

func (s *OpenAISTTService) createMultipartBody(file *os.File) (*bytes.Buffer, string, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return nil, "", err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, "", err
	}

	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return nil, "", err
	}

	if err := writer.WriteField("response_format", "json"); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

func (s *OpenAISTTService) parseTranscriptionResponse(body []byte) (string, error) {
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Text, nil
}