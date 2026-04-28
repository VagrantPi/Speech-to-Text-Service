package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRelayService struct {
	mock.Mock
}

func (m *MockRelayService) ProcessBatch(ctx context.Context, batchSize int) (int, error) {
	args := m.Called(ctx, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockRelayService) PollAndRelay(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestRelayWorker(t *testing.T) {
	t.Run("context cancellation stops worker", func(t *testing.T) {
		mockService := new(MockRelayService)
		mockService.On("PollAndRelay", mock.Anything).Return(nil).Maybe()

		worker := NewRelayWorker(mockService)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan struct{})
		go func() {
			worker.Start(ctx)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not stop within timeout")
		}

		mockService.AssertExpectations(t)
	})
}

func TestNewRelayWorker(t *testing.T) {
	mockService := new(MockRelayService)
	worker := NewRelayWorker(mockService)
	assert.NotNil(t, worker)
}