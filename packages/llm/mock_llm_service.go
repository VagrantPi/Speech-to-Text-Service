package llm

import (
	"context"
	"strings"
	"time"
)

type MockLLMService struct{}

var _ LLMRepoInterface = (*MockLLMService)(nil)

func NewMockLLMService() *MockLLMService {
	return &MockLLMService{}
}

func (s *MockLLMService) GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (string, error) {
	time.Sleep(100 * time.Millisecond)

	mockResponse := "這是 Mock LLM 服務回傳的測試摘要。重點如下：一、測試內容的第一點。二、測試內容的第二點。三、測試內容的第三點。"

	words := strings.Fields(mockResponse)
	for _, word := range words {
		tokenChan <- word + " "
	}

	return mockResponse, nil
}
