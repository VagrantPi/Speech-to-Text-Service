package domain

import (
	"time"

	"gorm.io/datatypes"
)

type OutboxEvent struct {
	ID              uint
	AggregateTypeID uint16
	AggregateID     uint
	Topic           string
	Payload         datatypes.JSON
	Status          string
	CreatedAt       time.Time
}
