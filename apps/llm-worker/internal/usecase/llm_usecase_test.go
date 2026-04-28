package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockTaskRepoLLM struct {
	mock.Mock
}

func (m *MockTaskRepoLLM) GetTranscript(ctx context.Context, taskID uint) (string, error) {
	args := m.Called(ctx, taskID)
	return args.String(0), args.Error(1)
}

func (m *MockTaskRepoLLM) UpdateSummary(ctx context.Context, taskID uint, summary string) error {
	args := m.Called(ctx, taskID, summary)
	return args.Error(0)
}

type MockLLMRepo struct {
	mock.Mock
}

func (m *MockLLMRepo) GenerateSummaryStream(ctx context.Context, transcript string, tokenChan chan<- string) (string, error) {
	args := m.Called(ctx, transcript, tokenChan)
	return args.String(0), args.Error(1)
}

type MockPubSubRepo struct {
	mock.Mock
}

func (m *MockPubSubRepo) Publish(ctx context.Context, channel string, payload string) error {
	args := m.Called(ctx, channel, payload)
	return args.Error(0)
}

func TestLLMUseCase_ProcessTask(t *testing.T) {
	tests := []struct {
		name          string
		taskID        uint
		transcript    string
		summary       string
		generateErr   error
		updateErr     error
		publishErr    error
		expectedErr   error
	}{
		{
			name:        "success",
			taskID:      1,
			transcript:  "Hello world",
			summary:     "Summary of hello world",
			generateErr: nil,
			updateErr:   nil,
			publishErr:  nil,
			expectedErr: nil,
		},
		{
			name:        "generate summary fails",
			taskID:      1,
			transcript:  "Hello world",
			summary:     "",
			generateErr: errors.New("generate failed"),
			updateErr:   nil,
			publishErr:  nil,
			expectedErr: errors.New("failed to generate summary: generate failed"),
		},
		{
			name:        "update summary fails",
			taskID:      1,
			transcript:  "Hello world",
			summary:     "Summary of hello world",
			generateErr: nil,
			updateErr:   errors.New("update failed"),
			publishErr:  nil,
			expectedErr: errors.New("failed to update summary: update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTask := new(MockTaskRepoLLM)
			mockLLM := new(MockLLMRepo)
			mockPubSub := new(MockPubSubRepo)

			logger := zap.NewNop()

			uc, err := NewLLMUseCase(mockTask, mockLLM, mockPubSub, logger)
			assert.NoError(t, err)

			ctx := context.Background()

			if tt.generateErr == nil {
				mockLLM.On("GenerateSummaryStream", ctx, tt.transcript, mock.Anything).Return(tt.summary, tt.generateErr)
			} else {
				mockLLM.On("GenerateSummaryStream", ctx, tt.transcript, mock.Anything).Return(tt.summary, tt.generateErr)
			}

			if tt.generateErr == nil {
				mockTask.On("UpdateSummary", ctx, tt.taskID, tt.summary).Return(tt.updateErr)
			}

			mockPubSub.On("Publish", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(tt.publishErr).Maybe()

			err = uc.ProcessTask(ctx, tt.taskID, tt.transcript)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			mockTask.AssertExpectations(t)
			mockLLM.AssertExpectations(t)
			mockPubSub.AssertExpectations(t)
		})
	}
}