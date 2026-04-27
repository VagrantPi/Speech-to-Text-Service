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
)

const (
	defaultOllamaURL = "http://localhost:11434/api/chat"
	defaultModel     = "qwen2.5:1.5b"
)

type LocalLLMService struct {
	client  *http.Client
	baseURL string
	model   string
}

var _ LLMRepoInterface = (*LocalLLMService)(nil)

func NewLocalLLMService() *LocalLLMService {
	baseURL := os.Getenv("OLLAMA_API_URL")
	model := os.Getenv("OLLAMA_MODEL")
	if baseURL == "" {
		baseURL = defaultOllamaURL
	}
	if model == "" {
		model = defaultModel
	}
	return &LocalLLMService{
		client: &http.Client{},
		baseURL: baseURL,
		model:   model,
	}
}

func NewLocalLLMServiceWithConfig(baseURL, model string) *LocalLLMService {
	return &LocalLLMService{
		client: &http.Client{},
		baseURL: baseURL,
		model:   model,
	}
}

// Ollama API 請求結構
type OllamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Ollama 串流回應結構
type OllamaStreamResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

func (s *LocalLLMService) GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (string, error) {
	// 確保結束後關閉 channel
	defer close(tokenChan)

	reqBody := OllamaChatRequest{
		Model: s.model,
		Messages: []Message{
			{Role: "system", Content: "你是一個專業的語音摘要助手。請根據以下逐字稿，直接输出一段簡潔的重點摘要。只需輸出摘要內容，不需要任何額外的說明、問候或詢問。"},
			{Role: "user", Content: "以下是需要摘要的逐字稿：\n" + transcript},
		},
		Stream: true,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 使用 json.Decoder 處理持續流入的 JSON 物件
	decoder := json.NewDecoder(resp.Body)
	var fullContent strings.Builder

	for {
		var streamResp OllamaStreamResponse
		// 解碼單個 JSON 物件
		if err := decoder.Decode(&streamResp); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("decode error: %w", err)
		}

		// 發送 token 到 channel
		if token := streamResp.Message.Content; token != "" {
			select {
			case tokenChan <- token:
				fullContent.WriteString(token)
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		if streamResp.Done {
			break
		}
	}

	return fullContent.String(), nil
}
