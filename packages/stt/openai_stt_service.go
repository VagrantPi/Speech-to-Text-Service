package stt

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type OpenAISTTService struct {
	client *openai.Client
}

func NewOpenAISTTService(apiKey string) *OpenAISTTService {
	return &OpenAISTTService{
		client: openai.NewClient(apiKey),
	}
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
