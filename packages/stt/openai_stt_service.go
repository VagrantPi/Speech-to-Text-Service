package stt

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type STTRepoInterface interface {
	Transcribe(ctx context.Context, localFilePath string) (string, error)
}

type OpenAISTTService struct {
	client *openai.Client
}

var _ STTRepoInterface = (*OpenAISTTService)(nil)

func NewOpenAISTTService(apiKey string) (*OpenAISTTService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	return &OpenAISTTService{
		client: openai.NewClient(apiKey),
	}, nil
}

func (s *OpenAISTTService) Transcribe(ctx context.Context, localFilePath string) (string, error) {
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
