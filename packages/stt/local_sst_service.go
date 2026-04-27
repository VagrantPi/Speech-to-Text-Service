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
	"path/filepath"
)

type LocalSTTServiceConfig struct {
	APIURL     string
	HTTPClient *http.Client
}

type LocalSTTService struct {
	config LocalSTTServiceConfig
}

var _ STTRepoInterface = (*LocalSTTService)(nil)

func NewLocalSTTService(apiKey string) *LocalSTTService {
	apiURL := os.Getenv("WHISPER_API_URL")
	return NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{
		APIURL:     apiURL,
		HTTPClient: &http.Client{},
	})
}

func NewLocalSTTServiceWithConfig(config LocalSTTServiceConfig) *LocalSTTService {
	if config.APIURL == "" {
		config.APIURL = "http://localhost:9000/asr?task=transcribe&output=json"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{}
	}
	return &LocalSTTService{
		config: config,
	}
}

// Transcribe 將指定路徑的音檔發送至本地 Whisper API 進行語音轉文字
func (s *LocalSTTService) Transcribe(ctx context.Context, filePath string) (string, error) {
	// 檢查 context 是否已取消
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context 已取消: %w", err)
	}

	// 1. 開啟本地音檔
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("無法開啟檔案 %s: %w", filePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// 記錄關閉錯誤但不影響主要錯誤處理
			fmt.Printf("警告：關閉檔案時發生錯誤: %v\n", closeErr)
		}
	}()

	// 2. 準備 Multipart Form 資料
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 建立表單中的檔案欄位 (API 要求的欄位名稱為 audio_file)
	part, err := writer.CreateFormFile("audio_file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("建立表單失敗: %w", err)
	}

	// 將音檔內容寫入表單，同時檢查 context
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("複製檔案內容失敗: %w", err)
	}

	// 再次檢查 context
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context 在準備請求時被取消: %w", err)
	}

	// 必須關閉 Writer，確保寫入結尾的 boundary
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("關閉 form writer 失敗: %w", err)
	}

	// 3. 發送 HTTP POST 請求
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.APIURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("建立 HTTP 請求失敗: %w", err)
	}
	// 設定正確的 Content-Type (包含隨機生成的 boundary)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.config.HTTPClient.Do(req)
	if err != nil {
		// 檢查是否為 context 取消導致的錯誤
		if ctx.Err() != nil {
			return "", fmt.Errorf("請求被取消: %w", ctx.Err())
		}
		return "", fmt.Errorf("發送請求時發生錯誤: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("警告：關閉回應 body 時發生錯誤: %v\n", closeErr)
		}
	}()

	// 4. 解析回傳結果
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("讀取 API 回應失敗: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API 發生錯誤 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析 API 回應失敗: %w", err)
	}

	if text, ok := result["text"].(string); ok {
		return text, nil
	}

	return string(respBody), nil
}
