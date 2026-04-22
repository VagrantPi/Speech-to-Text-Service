package models

import (
	"time"

	"gorm.io/datatypes"
)

// 定義業務實體的類型 Enum，用於多型關聯
const (
	AggregateTypeTask uint16 = 1
)

// OutboxEvent 發件匣中繼表
type OutboxEvent struct {
	// 事件專屬的遞增 ID
	ID uint `gorm:"primaryKey;autoIncrement"`

	// 複合索引：加速特定業務實體的事件追蹤
	AggregateTypeID uint16 `gorm:"index:idx_outbox_aggregate,priority:1;not null"`
	AggregateID     uint   `gorm:"index:idx_outbox_aggregate,priority:2;not null"`

	Topic string `gorm:"type:varchar(100);not null"`

	// 使用 gorm.io/datatypes 來支援 PostgreSQL 的 JSONB 格式
	Payload datatypes.JSON `gorm:"type:jsonb;not null"`

	// 部分索引 (Partial Index)：只針對 PENDING 狀態建 B-Tree，讓 Relay 輪詢效能極大化
	Status      string `gorm:"type:varchar(20);default:'PENDING';index:idx_outbox_pending,where:status='PENDING'"`
	RetryCount  int    // 記錄重試次數
	ErrorReason string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
}
