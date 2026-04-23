package stt

import (
	"context"
	"time"
)

type MockSTTService struct{}

var _ STTRepoInterface = (*MockSTTService)(nil)

func NewMockSTTService() *MockSTTService {
	return &MockSTTService{}
}

func (s *MockSTTService) Transcribe(ctx context.Context, localFilePath string) (string, error) {
	time.Sleep(100 * time.Millisecond)
	return "這是 Mock STT 服務回傳的測試逐字稿。", nil
}
