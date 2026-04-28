package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"speech.local/apps/api-server/internal/usecase"
	"speech.local/packages/db/models"
)

type MockTaskUseCase struct {
	mock.Mock
}

func (m *MockTaskUseCase) GetAudioUploadURL(ctx context.Context, fileExtension, contentType string) (string, string, error) {
	args := m.Called(ctx, fileExtension, contentType)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTaskUseCase) ConfirmTask(ctx context.Context, s3Key string) (uint, error) {
	args := m.Called(ctx, s3Key)
	return args.Get(0).(uint), args.Error(1)
}

func (m *MockTaskUseCase) GetTaskDetail(ctx context.Context, id uint) (*models.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskUseCase) StreamTaskSummary(ctx context.Context, taskID uint) (<-chan string, func() error, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).(<-chan string), args.Get(1).(func() error), args.Error(2)
}

func setupTestRouter(handler *TaskHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/upload-url", handler.HandleGetUploadURL)
	r.POST("/tasks/confirm", handler.HandleConfirmTask)
	r.GET("/tasks/:id", handler.HandleGetTask)
	r.GET("/tasks/:id/stream", handler.HandleStreamSummary)
	return r
}

func TestHandleGetUploadURL(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMock      func(*MockTaskUseCase)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "success",
			queryParams: map[string]string{
				"ext":          "mp3",
				"content_type": "audio/mpeg",
			},
			setupMock: func(m *MockTaskUseCase) {
				m.On("GetAudioUploadURL", mock.Anything, "mp3", "audio/mpeg").
					Return("https://signed-url.com", "uploads/audio/uuid.mp3", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"upload_url": "https://signed-url.com",
				"s3_key":     "uploads/audio/uuid.mp3",
			},
		},
		{
			name: "missing ext",
			queryParams: map[string]string{
				"content_type": "audio/mpeg",
			},
			setupMock:      func(m *MockTaskUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "query parameter 'ext' is required"},
		},
		{
			name: "missing content_type",
			queryParams: map[string]string{
				"ext": "mp3",
			},
			setupMock:      func(m *MockTaskUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "query parameter 'content_type' is required"},
		},
		{
			name: "invalid file extension",
			queryParams: map[string]string{
				"ext":          "pdf",
				"content_type": "application/pdf",
			},
			setupMock: func(m *MockTaskUseCase) {
				m.On("GetAudioUploadURL", mock.Anything, "pdf", "application/pdf").
					Return("", "", usecase.ErrInvalidFileExtension)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid file extension"},
		},
		{
			name: "storage error",
			queryParams: map[string]string{
				"ext":          "mp3",
				"content_type": "audio/mpeg",
			},
			setupMock: func(m *MockTaskUseCase) {
				m.On("GetAudioUploadURL", mock.Anything, "mp3", "audio/mpeg").
					Return("", "", errors.New("storage unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "storage unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockTaskUseCase)
			tt.setupMock(mockUC)

			handler := NewTaskHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req, _ := http.NewRequest("GET", "/upload-url", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			for k, v := range tt.expectedBody {
				assert.Equal(t, v, resp[k])
			}

			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandleConfirmTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name           string
		requestBody    map[string]string
		setupMock      func(*MockTaskUseCase)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "success",
			requestBody: map[string]string{"s3_key": "uploads/audio/test.mp3"},
			setupMock: func(m *MockTaskUseCase) {
				m.On("ConfirmTask", mock.Anything, "uploads/audio/test.mp3").
					Return(uint(123), nil)
			},
			expectedStatus: http.StatusAccepted,
			expectedBody: map[string]interface{}{
				"task_id": float64(123),
				"status":  "PENDING",
			},
		},
		{
			name:        "missing s3_key",
			requestBody: map[string]string{},
			setupMock:   func(m *MockTaskUseCase) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "invalid request body: s3_key is required",
			},
		},
		{
			name:        "confirm task error",
			requestBody: map[string]string{"s3_key": "invalid-key"},
			setupMock: func(m *MockTaskUseCase) {
				m.On("ConfirmTask", mock.Anything, "invalid-key").
					Return(uint(0), errors.New("invalid s3 key"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "invalid s3 key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockTaskUseCase)
			tt.setupMock(mockUC)

			handler := NewTaskHandler(mockUC, logger)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/tasks/confirm", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			for k, v := range tt.expectedBody {
				assert.Equal(t, v, resp[k])
			}

			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandleGetTask(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name           string
		taskID         string
		setupMock      func(*MockTaskUseCase)
		expectedStatus int
	}{
		{
			name:   "success",
			taskID: "123",
			setupMock: func(m *MockTaskUseCase) {
				m.On("GetTaskDetail", mock.Anything, uint(123)).
					Return(&models.Task{
						ID:     123,
						Status: "COMPLETED",
						S3Key:  "uploads/audio/test.mp3",
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalid id",
			taskID: "abc",
			setupMock: func(m *MockTaskUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "task not found",
			taskID: "999",
			setupMock: func(m *MockTaskUseCase) {
				m.On("GetTaskDetail", mock.Anything, uint(999)).
					Return(nil, errors.New("record not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockTaskUseCase)
			tt.setupMock(mockUC)

			handler := NewTaskHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req, _ := http.NewRequest("GET", "/tasks/"+tt.taskID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestHandleStreamSummary(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name           string
		taskID         string
		setupMock      func(*MockTaskUseCase)
		expectedStatus int
	}{
		{
			name:   "invalid id",
			taskID: "abc",
			setupMock: func(m *MockTaskUseCase) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "pubsub error",
			taskID: "123",
			setupMock: func(m *MockTaskUseCase) {
				var nilCh <-chan string
				m.On("StreamTaskSummary", mock.Anything, uint(123)).
					Return(nilCh, func() error { return nil }, errors.New("redis connection error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := new(MockTaskUseCase)
			tt.setupMock(mockUC)

			handler := NewTaskHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req, _ := http.NewRequest("GET", "/tasks/"+tt.taskID+"/stream", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockUC.AssertExpectations(t)
		})
	}
}