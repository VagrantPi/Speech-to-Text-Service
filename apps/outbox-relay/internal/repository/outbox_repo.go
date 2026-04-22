package repository

import (
	"context"

	"speech.local/packages/db/models"
)

type OutboxRepo interface {
	// 撈取待處理事件
	FetchPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error)
	// 標記為已處理
	MarkAsProcessed(ctx context.Context, id uint) error
	// 標記為失敗並增加重試計數
	MarkAsFailed(ctx context.Context, id uint, errReason string) error
}
