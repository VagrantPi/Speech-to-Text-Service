package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRoundTripper struct {
	mock.Mock
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestLocalLLMService_GenerateSummaryStream(t *testing.T) {
	tests := []struct {
		name           string
		transcript     string
		responses      []OllamaStreamResponse
		httpStatusCode int
		httpErr        error
		expectedSummary string
		expectedErr     string
	}{
		{
			name:       "success",
			transcript: "Hello world",
			responses: []OllamaStreamResponse{
				{Message: Message{Content: "這是"}, Done: false},
				{Message: Message{Content: "摘要"}, Done: false},
				{Message: Message{Content: "內容"}, Done: true},
			},
			httpStatusCode:  http.StatusOK,
			expectedSummary: "這是摘要內容",
		},
		{
			name:           "http error",
			transcript:     "Hello",
			httpErr:        assert.AnError,
			expectedErr:    "request failed:",
		},
		{
			name:           "non-200 status",
			transcript:     "Hello",
			httpStatusCode: http.StatusInternalServerError,
			expectedErr:    "unexpected status code: 500",
		},
		{
			name:           "empty transcript",
			transcript:     "",
			responses:      []OllamaStreamResponse{{Message: Message{Content: "OK"}, Done: true}},
			httpStatusCode: http.StatusOK,
			expectedSummary: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := new(MockRoundTripper)

			var responseBody string
			for _, resp := range tt.responses {
				data, _ := json.Marshal(resp)
				responseBody += string(data) + "\n"
			}

			if tt.httpErr != nil {
				mockTransport.On("RoundTrip", mock.Anything).Return(nil, tt.httpErr)
			} else {
				mockTransport.On("RoundTrip", mock.Anything).Return(&http.Response{
					StatusCode: tt.httpStatusCode,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil)
			}

			svc := &LocalLLMService{
				client:  &http.Client{Transport: mockTransport},
				baseURL: "http://localhost:11434/api/chat",
				model:   "qwen2.5:1.5b",
			}

			tokenChan := make(chan string, 100)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			summary, err := svc.GenerateSummaryStream(ctx, tt.transcript, tokenChan)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSummary, summary)
			}

			mockTransport.AssertExpectations(t)
		})
	}
}

func TestLocalLLMService_NewLocalLLMServiceWithConfig(t *testing.T) {
	svc := NewLocalLLMServiceWithConfig("http://custom:8080", "custom-model")

	assert.Equal(t, "http://custom:8080", svc.baseURL)
	assert.Equal(t, "custom-model", svc.model)
}