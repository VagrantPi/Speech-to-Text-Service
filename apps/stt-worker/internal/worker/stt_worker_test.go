package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMQConsumer struct {
	mock.Mock
}

func (m *MockMQConsumer) Consume(ctx context.Context, queueName string, handler func(ctx context.Context, payload []byte) error) error {
	args := m.Called(ctx, queueName, handler)
	return args.Error(0)
}

type MockSTTUseCase struct {
	mock.Mock
}

func (m *MockSTTUseCase) ProcessTask(ctx context.Context, taskID uint, s3Key string) error {
	args := m.Called(ctx, taskID, s3Key)
	return args.Error(0)
}

func TestSTTWorker_Start(t *testing.T) {
	tests := []struct {
		name        string
		consumeErr  error
		expectedErr error
	}{
		{
			name:        "success",
			consumeErr:  nil,
			expectedErr: nil,
		},
		{
			name:        "consume error",
			consumeErr:  errors.New("consume failed"),
			expectedErr: errors.New("consume failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConsumer := new(MockMQConsumer)
			mockUC := new(MockSTTUseCase)

			worker := NewSTTWorker(mockConsumer, mockUC)

			ctx := context.Background()
			mockConsumer.On("Consume", ctx, "stt-queue", mock.AnythingOfType("func(context.Context, []uint8) error")).Return(tt.consumeErr)

			err := worker.Start(ctx)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			mockConsumer.AssertExpectations(t)
		})
	}
}

func TestSTTWorker_handleMessage(t *testing.T) {
	tests := []struct {
		name          string
		payload       []byte
		processTaskErr error
		expectedErr   error
	}{
		{
			name:    "success",
			payload: mustMarshal(taskPayload{TaskID: 1, S3Key: "audio/task1.wav"}),
			processTaskErr: nil,
			expectedErr:   nil,
		},
		{
			name:          "invalid json",
			payload:       []byte("invalid json"),
			processTaskErr: nil,
			expectedErr:   errors.New("expected JSON unmarshal error"),
		},
		{
			name:    "process task error",
			payload: mustMarshal(taskPayload{TaskID: 1, S3Key: "audio/task1.wav"}),
			processTaskErr: errors.New("process task failed"),
			expectedErr:   errors.New("process task failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConsumer := new(MockMQConsumer)
			mockUC := new(MockSTTUseCase)

			worker := NewSTTWorker(mockConsumer, mockUC)

			ctx := context.Background()

			if tt.expectedErr == nil || (tt.expectedErr != nil && tt.expectedErr.Error() == "process task failed") {
				var task taskPayload
				if err := json.Unmarshal(tt.payload, &task); err == nil {
					mockUC.On("ProcessTask", ctx, task.TaskID, task.S3Key).Return(tt.processTaskErr)
				}
			}

			err := worker.handleMessage(ctx, tt.payload)

			if tt.expectedErr != nil {
				if tt.expectedErr.Error() == "expected JSON unmarshal error" {
					assert.Error(t, err)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedErr == nil || (tt.expectedErr != nil && tt.expectedErr.Error() == "process task failed") {
				mockUC.AssertExpectations(t)
			}
		})
	}
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}