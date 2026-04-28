package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockStorageRepo struct {
	mock.Mock
}

func (m *MockStorageRepo) DownloadToTempFile(ctx context.Context, s3Key string) (string, error) {
	args := m.Called(ctx, s3Key)
	return args.String(0), args.Error(1)
}

type MockSTTRepo struct {
	mock.Mock
}

func (m *MockSTTRepo) Transcribe(ctx context.Context, localFilePath string) (string, error) {
	args := m.Called(ctx, localFilePath)
	return args.String(0), args.Error(1)
}

type MockTaskRepo struct {
	mock.Mock
}

func (m *MockTaskRepo) UpdateTranscript(ctx context.Context, taskID uint, transcript string) error {
	args := m.Called(ctx, taskID, transcript)
	return args.Error(0)
}

func TestSTTUseCase_ProcessTask(t *testing.T) {
	tests := []struct {
		name          string
		taskID        uint
		s3Key         string
		env           string
		downloadPath  string
		downloadErr   error
		transcript    string
		transcribeErr error
		updateErr     error
		expectedErr   error
	}{
		{
			name:         "success with local env",
			taskID:       1,
			s3Key:        "audio/task1.wav",
			env:          "local",
			downloadPath: "/tmp/task1.wav",
			transcript:   "Hello world",
			expectedErr:  nil,
		},
		{
			name:         "success with mock env",
			taskID:       1,
			s3Key:        "audio/task1.wav",
			env:          "mock",
			downloadPath: "/tmp/mock-audio.wav",
			transcript:   "Hello world",
			expectedErr:  nil,
		},
		{
			name:        "download fails",
			taskID:      1,
			s3Key:       "audio/task1.wav",
			env:         "local",
			downloadErr: errors.New("download failed"),
			expectedErr: errors.New("download failed"),
		},
		{
			name:          "transcribe fails",
			taskID:        1,
			s3Key:         "audio/task1.wav",
			env:           "local",
			downloadPath:  "/tmp/task1.wav",
			transcribeErr: errors.New("transcribe failed"),
			expectedErr:   errors.New("transcribe failed"),
		},
		{
			name:         "update transcript fails",
			taskID:       1,
			s3Key:        "audio/task1.wav",
			env:          "local",
			downloadPath: "/tmp/task1.wav",
			transcript:   "Hello world",
			updateErr:    errors.New("update failed"),
			expectedErr:  errors.New("update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(MockStorageRepo)
			mockSTT := new(MockSTTRepo)
			mockTask := new(MockTaskRepo)

			logger := zap.NewNop()

			uc, err := NewSTTUseCase(mockStorage, mockSTT, mockTask, logger, tt.env)
			assert.NoError(t, err)

			ctx := context.Background()

			if tt.env != "mock" {
				mockStorage.On("DownloadToTempFile", ctx, tt.s3Key).Return(tt.downloadPath, tt.downloadErr)
			}
			if tt.downloadErr == nil {
				expectedPath := tt.downloadPath
				if tt.env == "mock" {
					expectedPath = "/tmp/mock-audio.wav"
				}
				mockSTT.On("Transcribe", ctx, expectedPath).Return(tt.transcript, tt.transcribeErr)
			}
			if tt.downloadErr == nil && tt.transcribeErr == nil {
				mockTask.On("UpdateTranscript", ctx, tt.taskID, tt.transcript).Return(tt.updateErr)
			}

			err = uc.ProcessTask(ctx, tt.taskID, tt.s3Key)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
			mockSTT.AssertExpectations(t)
			mockTask.AssertExpectations(t)
		})
	}
}