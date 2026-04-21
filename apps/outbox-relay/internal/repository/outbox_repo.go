package repository

import (
	"context"

	"speech.local/packages/db/models"
)

type OutboxRepo interface {
	FetchPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error)
	Delete(ctx context.Context, id uint) error
}
