package db

import (
	"context"
	"log"

	"gorm.io/gorm/clause"
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
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND retry_count < ?", "PENDING", 5).
		Order("created_at ASC").
		Limit(limit).
		Find(&events)
	log.Printf("[DEBUG] Fetched %d pending events", len(events))
	return events, err.Error
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

type ProcessFunc func(ctx context.Context, tx *gorm.DB, events []*models.OutboxEvent) error

func (d *OutboxDAO) FetchAndProcess(ctx context.Context, limit int, process ProcessFunc) (int, error) {
	var events []*models.OutboxEvent
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND retry_count < ?", "PENDING", 5).
			Order("created_at ASC").
			Limit(limit).
			Find(&events).Error; err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		return process(ctx, tx, events)
	})
	return len(events), err
}

func (d *OutboxDAO) MarkAsProcessedTx(ctx context.Context, tx *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(&models.OutboxEvent{}).
		Where("id IN ?", ids).
		Update("status", "PROCESSED").Error
}

func (d *OutboxDAO) MarkAsFailedTx(ctx context.Context, tx *gorm.DB, id uint, errReason string) error {
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
}
