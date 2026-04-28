package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

type MockOutboxRepo struct {
	mock.Mock
}

func (m *MockOutboxRepo) FetchPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OutboxEvent), args.Error(1)
}

func (m *MockOutboxRepo) MarkAsProcessed(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOutboxRepo) MarkAsFailed(ctx context.Context, id uint, errReason string) error {
	args := m.Called(ctx, id, errReason)
	return args.Error(0)
}

func (m *MockOutboxRepo) FetchAndProcess(ctx context.Context, limit int, fn db.ProcessFunc) (int, error) {
	args := m.Called(ctx, limit, fn)
	return args.Int(0), args.Error(1)
}

func (m *MockOutboxRepo) MarkAsProcessedTx(ctx context.Context, tx *gorm.DB, ids []uint) error {
	args := m.Called(ctx, tx, ids)
	return args.Error(0)
}

func (m *MockOutboxRepo) MarkAsFailedTx(ctx context.Context, tx *gorm.DB, id uint, errReason string) error {
	args := m.Called(ctx, tx, id, errReason)
	return args.Error(0)
}

func TestMockOutboxRepo(t *testing.T) {
	t.Run("MarkAsProcessedTx calls expected arguments", func(t *testing.T) {
		mockRepo := new(MockOutboxRepo)
		var tx *gorm.DB
		ids := []uint{1, 2, 3}

		mockRepo.On("MarkAsProcessedTx", mock.Anything, tx, ids).Return(nil)

		err := mockRepo.MarkAsProcessedTx(context.Background(), tx, ids)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("FetchPendingEvents returns events", func(t *testing.T) {
		mockRepo := new(MockOutboxRepo)
		events := []*models.OutboxEvent{
			{ID: 1, Topic: "test"},
			{ID: 2, Topic: "test"},
		}

		mockRepo.On("FetchPendingEvents", mock.Anything, 10).Return(events, nil)

		result, err := mockRepo.FetchPendingEvents(context.Background(), 10)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockRepo.AssertExpectations(t)
	})
}

func TestOutboxRepoInterface(t *testing.T) {
	var repo OutboxRepo = new(MockOutboxRepo)
	assert.NotNil(t, repo)
}