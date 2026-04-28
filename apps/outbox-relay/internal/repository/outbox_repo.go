package repository

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

type OutboxRepo interface {
	FetchPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error)
	MarkAsProcessed(ctx context.Context, id uint) error
	MarkAsFailed(ctx context.Context, id uint, errReason string) error
	FetchAndProcess(ctx context.Context, limit int, fn db.ProcessFunc) (int, error)
	MarkAsProcessedTx(ctx context.Context, tx *gorm.DB, ids []uint) error
	MarkAsFailedTx(ctx context.Context, tx *gorm.DB, id uint, errReason string) error
}
