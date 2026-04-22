package db

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db/models"
)

type OutboxDAO struct {
	db *gorm.DB
}

func NewOutboxDAO(db *gorm.DB) *OutboxDAO {
	return &OutboxDAO{db: db}
}

func (d *OutboxDAO) FetchPendingEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error) {
	var events []*models.OutboxEvent
	err := d.db.WithContext(ctx).
		Where("status = ? AND retry_count < ?", "PENDING", 5).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (d *OutboxDAO) MarkAsProcessed(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Model(&models.OutboxEvent{}).
		Where("id = ?", id).
		Update("status", "PROCESSED").Error
}

func (d *OutboxDAO) MarkAsFailed(ctx context.Context, id uint, errReason string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event models.OutboxEvent
		if err := tx.WithContext(ctx).First(&event, id).Error; err != nil {
			return err
		}

		newRetryCount := event.RetryCount + 1
		updates := map[string]interface{}{
			"retry_count":  newRetryCount,
			"error_reason": errReason,
		}

		if newRetryCount >= 5 {
			updates["status"] = "FAILED"
		}

		return tx.Model(&models.OutboxEvent{}).Where("id = ?", id).Updates(updates).Error
	})
}
