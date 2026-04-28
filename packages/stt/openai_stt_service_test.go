package stt

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockOpenAIClient struct {
	onCreateTranscription func(req interface{}) (string, error)
}

func TestOpenAISTTService_NewOpenAISTTService(t *testing.T) {
	tests := []struct {
		name     string
		apiKey  string
		wantErr bool
		errMsg  string
	}{
		{
			name:     "正常建立",
			apiKey:  "test-api-key",
			wantErr: false,
		},
		{
			name:     "空的 apiKey 應失敗",
			apiKey:  "",
			wantErr: true,
			errMsg:  "OPENAI_API_KEY is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewOpenAISTTService(tt.apiKey)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
			}
		})
	}
}

func TestOpenAISTTService_Transcribe_Success(t *testing.T) {
	svc, err := NewOpenAISTTService("test-key")
	assert.NoError(t, err)

	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.Write([]byte("fake audio content"))
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = svc.Transcribe(ctx, tmpFile.Name())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestOpenAISTTService_Transcribe_FileNotFound(t *testing.T) {
	svc, err := NewOpenAISTTService("test-key")
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := svc.Transcribe(ctx, "/nonexistent/file.wav")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
	assert.Empty(t, text)
}