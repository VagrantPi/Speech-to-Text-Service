package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"speech.local/packages/db/models"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) GenerateUploadURL(ctx context.Context, objectKey, contentType string) (string, error) {
	args := m.Called(ctx, objectKey, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) DownloadToTempFile(ctx context.Context, s3Key string) (string, error) {
	args := m.Called(ctx, s3Key)
	return args.String(0), args.Error(1)
}

// MockTaskRepository is a mock implementation of TaskService for testing
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) CreateTaskWithOutbox(ctx context.Context, s3Key string) (uint, error) {
	args := m.Called(ctx, s3Key)
	return args.Get(0).(uint), args.Error(1)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id uint) (*models.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Task), args.Error(1)
}

type MockPubSubRepo struct {
	mock.Mock
}

func (m *MockPubSubRepo) Subscribe(ctx context.Context, channel string) (<-chan string, func() error, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(<-chan string), args.Get(1).(func() error), args.Error(2)
}

func TestGetAudioUploadURL(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		fileExtension string
		setupMock     func(mockStorage *MockStorage)
		expectedURL   string
		expectedKey   string
		expectedErr   error
	}{
		{
			name:          "success - mp3",
			fileExtension: "mp3",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - wav",
			fileExtension: "wav",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - webm",
			fileExtension: "webm",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - m4a",
			fileExtension: "m4a",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - mp4",
			fileExtension: "mp4",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - mpeg",
			fileExtension: "mpeg",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - mpga",
			fileExtension: "mpga",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "invalid extension - txt",
			fileExtension: "txt",
			setupMock:     func(m *MockStorage) {},
			expectedErr:   ErrInvalidFileExtension,
		},
		{
			name:          "invalid extension - pdf",
			fileExtension: "pdf",
			setupMock:     func(m *MockStorage) {},
			expectedErr:   ErrInvalidFileExtension,
		},
		{
			name:          "invalid extension - empty",
			fileExtension: "",
			setupMock:     func(m *MockStorage) {},
			expectedErr:   ErrInvalidFileExtension,
		},
		{
			name:          "invalid extension - with dot",
			fileExtension: ".mp3",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "success - uppercase MP3 normalized to .mp3",
			fileExtension: "MP3",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("https://storage.example.com/signed-url", nil)
			},
			expectedURL: "https://storage.example.com/signed-url",
			expectedErr: nil,
		},
		{
			name:          "storage error",
			fileExtension: "mp3",
			setupMock: func(m *MockStorage) {
				m.On("GenerateUploadURL", ctx, mock.Anything, mock.Anything).Return("", errors.New("storage service unavailable"))
			},
			expectedErr: errors.New("storage service unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorage)
			mockTaskRepo := new(MockTaskRepository)
			mockPubSub := new(MockPubSubRepo)
			usecase := NewTaskUseCase(mockStorage, mockTaskRepo, mockPubSub)

			tt.setupMock(mockStorage)

			uploadURL, s3Key, err := usecase.GetAudioUploadURL(ctx, tt.fileExtension, "audio/mpeg")

			expectedExt := strings.ToLower(tt.fileExtension)
			if len(expectedExt) > 0 && expectedExt[0] == '.' {
				expectedExt = expectedExt[1:]
			}

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, uploadURL)
				assert.Contains(t, s3Key, "uploads/audio/")
				assert.True(t, strings.HasSuffix(s3Key, "."+expectedExt))
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestTaskUseCase_ConfirmTask(t *testing.T) {
	tests := []struct {
		name       string
		s3Key      string
		mockTaskID uint
		mockErr    error
		wantTaskID uint
		wantErr    error
		setupMock  func(*MockTaskRepository)
	}{
		{
			name:      "empty s3Key returns ErrInvalidS3Key",
			s3Key:     "",
			wantErr:   ErrInvalidS3Key,
			setupMock: func(_ *MockTaskRepository) {}, // no call expected
		},
		{
			name:       "valid s3Key calls repository and returns taskID",
			s3Key:      "valid-key",
			mockTaskID: 123,
			wantTaskID: 123,
			setupMock: func(m *MockTaskRepository) {
				m.On("CreateTaskWithOutbox", mock.Anything, "valid-key").
					Return(uint(123), nil)
			},
		},
		{
			name:    "repository error propagates",
			s3Key:   "valid-key",
			mockErr: assert.AnError,
			wantErr: assert.AnError,
			setupMock: func(m *MockTaskRepository) {
				m.On("CreateTaskWithOutbox", mock.Anything, "valid-key").
					Return(uint(0), assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}
			uc := &taskUseCase{
				taskRepo: mockRepo,
			}

			taskID, err := uc.ConfirmTask(context.Background(), tt.s3Key)

			assert.Equal(t, tt.wantTaskID, taskID)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
