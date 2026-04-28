package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"speech.local/packages/db/models"
)

func setupOutboxDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.OutboxEvent{})
	require.NoError(t, err)

	return db
}

func TestOutboxDAO_FetchPendingEvents(t *testing.T) {
	db := setupOutboxDB(t)
	dao := NewOutboxDAO(db)

	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":1}`), Status: "PENDING"})
	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":2}`), Status: "PENDING"})
	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":3}`), Status: "PROCESSED"})

	tests := []struct {
		name       string
		limit     int
		wantCount int
	}{
		{"success - fetch pending events", 10, 2},
		{"limit 1", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := dao.FetchPendingEvents(context.Background(), tt.limit)
			assert.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
		})
	}
}

func TestOutboxDAO_MarkAsProcessed(t *testing.T) {
	db := setupOutboxDB(t)
	dao := NewOutboxDAO(db)

	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":1}`), Status: "PENDING"})

	t.Run("success", func(t *testing.T) {
		err := dao.MarkAsProcessed(context.Background(), 1)
		assert.NoError(t, err)

		var event models.OutboxEvent
		db.First(&event, 1)
		assert.Equal(t, "PROCESSED", event.Status)
	})
}

func TestOutboxDAO_MarkAsProcessedTx(t *testing.T) {
	db := setupOutboxDB(t)
	dao := NewOutboxDAO(db)

	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":1}`), Status: "PENDING"})
	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":2}`), Status: "PENDING"})

	t.Run("success - single id", func(t *testing.T) {
		err := dao.MarkAsProcessedTx(context.Background(), db, []uint{1})
		assert.NoError(t, err)

		var event models.OutboxEvent
		db.First(&event, 1)
		assert.Equal(t, "PROCESSED", event.Status)
	})

	t.Run("success - empty ids", func(t *testing.T) {
		err := dao.MarkAsProcessedTx(context.Background(), db, []uint{})
		assert.NoError(t, err)
	})
}

func TestOutboxDAO_MarkAsFailed(t *testing.T) {
	db := setupOutboxDB(t)
	dao := NewOutboxDAO(db)

	mustCreate(t, db, &models.OutboxEvent{Topic: "stt-queue", Payload: []byte(`{"id":1}`), Status: "PENDING", RetryCount: 0})

	t.Run("success - increment retry count", func(t *testing.T) {
		err := dao.MarkAsFailed(context.Background(), 1, "error reason")
		assert.NoError(t, err)

		var event models.OutboxEvent
		db.First(&event, 1)
		assert.Equal(t, 1, event.RetryCount)
		assert.Equal(t, "error reason", event.ErrorReason)
		assert.Equal(t, "PENDING", event.Status)
	})

	t.Run("exceeds max retries - mark as FAILED", func(t *testing.T) {
		db.Model(&models.OutboxEvent{}).Where("id = ?", 1).Update("retry_count", 4)

		err := dao.MarkAsFailed(context.Background(), 1, "error reason")
		assert.NoError(t, err)

		var event models.OutboxEvent
		db.First(&event, 1)
		assert.Equal(t, 5, event.RetryCount)
		assert.Equal(t, "FAILED", event.Status)
	})
}