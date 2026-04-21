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
		Where("status = ?", "PENDING").
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (d *OutboxDAO) Delete(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&models.OutboxEvent{}, id).Error
}
