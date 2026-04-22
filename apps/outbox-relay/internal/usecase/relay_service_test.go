package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"

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
				{ID: 1, Topic: "task.created", Payload: datatypes.JSON(`{"id":1}`)},
				{ID: 2, Topic: "task.updated", Payload: datatypes.JSON(`{"id":2}`)},
				{ID: 3, Topic: "task.completed", Payload: datatypes.JSON(`{"id":3}`)},
			},
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {
				for _, e := range events {
					m.On("Publish", ctx, e.Topic, []byte(e.Payload)).Return(nil)
				}
			},
			setupRepo: func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {
				for _, e := range events {
					m.On("MarkAsProcessed", ctx, e.ID).Return(nil)
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
				{ID: 1, Topic: "task.created", Payload: datatypes.JSON(`{"id":1}`)},
				{ID: 2, Topic: "task.updated", Payload: datatypes.JSON(`{"id":2}`)},
				{ID: 3, Topic: "task.completed", Payload: datatypes.JSON(`{"id":3}`)},
			},
			setupPublisher: func(m *MockPublisher, events []*models.OutboxEvent) {
				m.On("Publish", ctx, events[0].Topic, []byte(events[0].Payload)).Return(nil)
				m.On("Publish", ctx, events[1].Topic, []byte(events[1].Payload)).Return(errors.New("publish failed"))
				m.On("Publish", ctx, events[2].Topic, []byte(events[2].Payload)).Return(nil)
			},
			setupRepo: func(m *MockRepoOutboxEvent, events []*models.OutboxEvent) {
				m.On("MarkAsProcessed", ctx, events[0].ID).Return(nil)
				m.On("MarkAsProcessed", ctx, events[2].ID).Return(nil)
			},
			expectedCount: 2,
			expectedErr:   nil,
		},
		{
			name:      "all publish failures - skipped but not marked as failed",
			batchSize: 10,
			pendingEvents: []*models.OutboxEvent{
				{ID: 1, Topic: "task.created", Payload: datatypes.JSON(`{"id":1}`)},
				{ID: 2, Topic: "task.updated", Payload: datatypes.JSON(`{"id":2}`)},
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
			usecase := NewRelayService(mockPublisher, mockRepo)

			if tt.fetchErr == nil && tt.pendingEvents != nil {
				mockRepo.On("FetchPendingEvents", ctx, tt.batchSize).Return(tt.pendingEvents, nil)
			} else if tt.fetchErr != nil {
				mockRepo.On("FetchPendingEvents", ctx, tt.batchSize).Return(nil, tt.fetchErr)
			} else {
				mockRepo.On("FetchPendingEvents", ctx, tt.batchSize).Return([]*models.OutboxEvent{}, nil)
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
