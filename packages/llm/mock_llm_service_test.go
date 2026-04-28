package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockLLMService_GenerateSummaryStream(t *testing.T) {
	svc := NewMockLLMService()

	tokenChan := make(chan string, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := svc.GenerateSummaryStream(ctx, "test transcript", tokenChan)

	assert.NoError(t, err)
	assert.Equal(t, "這是MockLLM服務回傳的測試摘要。重點如下：一、測試內容的第一點。二、測試內容的第二點。三、測試內容的第三點。", summary)

	var tokens string
	for token := range tokenChan {
		tokens += token
	}
	assert.Equal(t, summary, tokens)
}