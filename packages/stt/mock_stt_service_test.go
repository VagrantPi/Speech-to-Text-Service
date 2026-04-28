package stt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockSTTService_Transcribe(t *testing.T) {
	svc := NewMockSTTService()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := svc.Transcribe(ctx, "test audio file.wav")

	assert.NoError(t, err)
	assert.Equal(t, "這是 Mock STT 服務回傳的測試逐字稿。", text)
}

func TestMockSTTService_TranscribeContextCancel(t *testing.T) {
	svc := NewMockSTTService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	text, err := svc.Transcribe(ctx, "test audio file.wav")

	assert.NoError(t, err)
	assert.Equal(t, "這是 Mock STT 服務回傳的測試逐字稿。", text)
}