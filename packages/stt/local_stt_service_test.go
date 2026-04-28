package stt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalSTTService_Transcribe_Success(t *testing.T) {
	expectedText := "測試轉文字結果"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		_, err := r.MultipartReader()
		assert.NoError(t, err, "請求應該是 multipart form")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "` + expectedText + `"}`))
	}))
	defer server.Close()

	svc := NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{
		APIURL:     server.URL,
		HTTPClient: server.Client(),
	})

	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.Write([]byte("fake audio content"))
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := svc.Transcribe(ctx, tmpFile.Name())

	assert.NoError(t, err)
	assert.Equal(t, expectedText, text)
}

func TestLocalSTTService_Transcribe_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	svc := NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{
		APIURL:     server.URL,
		HTTPClient: server.Client(),
	})

	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.Write([]byte("fake audio content"))
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := svc.Transcribe(ctx, tmpFile.Name())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Empty(t, text)
}

func TestLocalSTTService_Transcribe_FileNotFound(t *testing.T) {
	svc := NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{
		APIURL:     "http://localhost:9000/asr",
		HTTPClient: &http.Client{},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := svc.Transcribe(ctx, "/nonexistent/path/audio.wav")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "無法開啟檔案")
	assert.Empty(t, text)
}

func TestLocalSTTService_Transcribe_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "valid but no text field"}`))
	}))
	defer server.Close()

	svc := NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{
		APIURL:     server.URL,
		HTTPClient: server.Client(),
	})

	tmpFile, err := os.CreateTemp("", "test_audio_*.wav")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.Write([]byte("fake audio content"))
	tmpFile.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	text, err := svc.Transcribe(ctx, tmpFile.Name())

	assert.NoError(t, err)
	assert.Equal(t, "valid but no text field", text)
}

func TestLocalSTTService_Transcribe_ContextCancel(t *testing.T) {
	svc := NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{
		APIURL:     "http://localhost:9000/asr",
		HTTPClient: &http.Client{},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	text, err := svc.Transcribe(ctx, "/fake/path.wav")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
	assert.Empty(t, text)
}

func TestLocalSTTService_NewLocalSTTServiceWithConfig_Defaults(t *testing.T) {
	svc := NewLocalSTTServiceWithConfig(LocalSTTServiceConfig{})

	assert.Equal(t, "http://localhost:9000/asr?task=transcribe&output=json", svc.config.APIURL)
	assert.NotNil(t, svc.config.HTTPClient)
}