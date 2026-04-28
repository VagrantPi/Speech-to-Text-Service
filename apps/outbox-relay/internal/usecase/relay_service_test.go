package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	args := m.Called(ctx, topic, payload)
	return args.Error(0)
}

type MockRepoOutboxEvent struct {
	mock.Mock
}

func (m *MockRepoOutboxEvent) FetchPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OutboxEvent), args.Error(1)
}

func (m *MockRepoOutboxEvent) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepoOutboxEvent) MarkAsProcessed(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepoOutboxEvent) MarkAsFailed(ctx context.Context, id uint, errReason string) error {
	args := m.Called(ctx, id, errReason)
	return args.Error(0)
}

func (m *MockRepoOutboxEvent) FetchAndProcess(ctx context.Context, limit int, fn db.ProcessFunc) (int, error) {
	args := m.Called(ctx, limit, fn)
	return args.Int(0), args.Error(1)
}

func (m *MockRepoOutboxEvent) MarkAsProcessedTx(ctx context.Context, tx *gorm.DB, ids []uint) error {
	args := m.Called(ctx, tx, ids)
	return args.Error(0)
}

func (m *MockRepoOutboxEvent) MarkAsFailedTx(ctx context.Context, tx *gorm.DB, id uint, errReason string) error {
	args := m.Called(ctx, tx, id, errReason)
	return args.Error(0)
}

func TestProcessBatch(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		batchSize      int
		pendingEvents  []*models.OutboxEvent
		fetchErr       error
		setupPublisher func(m *MockPublisher, events []*models.OutboxEvent)
		setupRepo      func(m *MockRepoOutboxEvent, events []*models.OutboxEvent)
		expectedCount  int
		expectedErr    error
	}{
		{
			name:      "success - process all events",
			batchSize: 10,
			pendingEvents: []*models.OutboxEvent{
				{ID: 1, Topic: "stt-queue", Payload: datatypes.JSON(`{"id":1}`)},
				{ID: 2, Topic: "stt-queue", Payload: datatypes.JSON(`{"id":2}`)},
				{ID: 3, Topic: "llm-queue", Payload: datatypes.JSON(`{"id":3}`)},
			},
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {
				for _, e := range events {
					m.On("Publish", ctx, e.Topic, []byte(e.Payload)).Return(nil)
				}
			},
			setupRepo: func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {
				for _, e := range events {
					m.On("MarkAsProcessedTx", mock.Anything, mock.Anything, []uint{e.ID}).Return(nil)
				}
			},
			expectedCount: 3,
			expectedErr:   nil,
		},
		{
			name:           "no pending events",
			batchSize:      10,
			pendingEvents:  []*models.OutboxEvent{},
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {},
			setupRepo:      func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {},
			expectedCount:  0,
			expectedErr:    nil,
		},
		{
			name:      "publish failure - should not mark as processed",
			batchSize: 10,
			pendingEvents: []*models.OutboxEvent{
				{ID: 1, Topic: "stt-queue", Payload: datatypes.JSON(`{"id":1}`)},
				{ID: 2, Topic: "stt-queue", Payload: datatypes.JSON(`{"id":2}`)},
				{ID: 3, Topic: "llm-queue", Payload: datatypes.JSON(`{"id":3}`)},
			},
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {
				m.On("Publish", ctx, events[0].Topic, []byte(events[0].Payload)).Return(nil)
				m.On("Publish", ctx, events[1].Topic, []byte(events[1].Payload)).Return(errors.New("publish failed"))
				m.On("Publish", ctx, events[2].Topic, []byte(events[2].Payload)).Return(nil)
			},
			setupRepo: func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {
				m.On("MarkAsProcessedTx", mock.Anything, mock.Anything, []uint{events[0].ID}).Return(nil)
				m.On("MarkAsProcessedTx", mock.Anything, mock.Anything, []uint{events[2].ID}).Return(nil)
			},
			expectedCount: 2,
			expectedErr:   nil,
		},
		{
			name:      "all publish failures - skipped but not marked as failed",
			batchSize: 10,
			pendingEvents: []*models.OutboxEvent{
				{ID: 1, Topic: "stt-queue", Payload: datatypes.JSON(`{"id":1}`)},
				{ID: 2, Topic: "stt-queue", Payload: datatypes.JSON(`{"id":2}`)},
			},
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {
				for _, e := range events {
					m.On("Publish", ctx, e.Topic, []byte(e.Payload)).Return(errors.New("mq connection failed"))
				}
			},
			setupRepo:     func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {},
			expectedCount: 0,
			expectedErr:   nil,
		},
		{
			name:           "fetch error",
			batchSize:      10,
			fetchErr:       errors.New("database connection failed"),
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {},
			setupRepo:      func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {},
			expectedCount:  0,
			expectedErr:    errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPublisher := new(MockPublisher)
			mockRepo := new(MockRepoOutboxEvent)
			logger, _ := zap.NewDevelopment()
			usecase, _ := NewRelayService(mockPublisher, mockRepo, logger)

			if tt.fetchErr != nil {
				mockRepo.On("FetchAndProcess", mock.Anything, mock.Anything, mock.Anything).Return(0, tt.fetchErr)
			} else {
				mockRepo.On("FetchAndProcess", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					fn := args.Get(2)
					if f, ok := fn.(db.ProcessFunc); ok {
						_ = f(context.Background(), nil, tt.pendingEvents)
					}
				}).Return(len(tt.pendingEvents), nil)
			}

			tt.setupPublisher(mockPublisher, tt.pendingEvents)
			tt.setupRepo(mockRepo, tt.pendingEvents)

			count, err := usecase.ProcessBatch(ctx, tt.batchSize)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCount, count)

			mockPublisher.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}
