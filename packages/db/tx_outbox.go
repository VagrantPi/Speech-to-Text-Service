// packages/db/tx_outbox.go
package db

import (
	"encoding/json"

	"gorm.io/gorm"
	"speech.local/packages/db/models"
)

// ExecuteWithOutbox 封裝了 Transaction 與 Outbox 機制
// 高層只要傳入 dbOp (真正的業務邏輯) 即可，不用再手動 Commit/Rollback
func ExecuteWithOutbox(
	db *gorm.DB,
	aggregateType uint16,
	topic string,
	payload map[string]interface{},
	dbOp func(tx *gorm.DB) (uint, error),
) (uint, error) {
	var aggregateID uint

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 執行外部傳入的業務邏輯 (例如：寫入 Task 表)
		id, err := dbOp(tx)
		if err != nil {
			return err // 發生錯誤，Gorm 會自動 Rollback
		}
		aggregateID = id

		// 2. 準備 Outbox Payload
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		// 3. 寫入 Outbox 紀錄
		outboxEvent := &models.OutboxEvent{
			AggregateTypeID: aggregateType,
			AggregateID:     aggregateID,
			Topic:           topic,
			Payload:         payloadBytes,
			Status:          "PENDING",
		}

		if err := tx.Create(outboxEvent).Error; err != nil {
			return err // Outbox 寫入失敗也會自動 Rollback
		}

		return nil // 一切順利，Gorm 會自動 Commit
	})

	return aggregateID, err
}
