package llm

import (
	"context"
	"time"
)

type MockLLMService struct{}

var _ LLMRepoInterface = (*MockLLMService)(nil)

func NewMockLLMService() *MockLLMService {
	return &MockLLMService{}
}

func (s *MockLLMService) GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (string, error) {
	mockResponse := "這是MockLLM服務回傳的測試摘要。重點如下：一、測試內容的第一點。二、測試內容的第二點。三、測試內容的第三點。"

	runes := []rune(mockResponse)
	for _, r := range runes {
		time.Sleep(100 * time.Millisecond)
		tokenChan <- string(r)
	}
	close(tokenChan)

	return mockResponse, nil
}
