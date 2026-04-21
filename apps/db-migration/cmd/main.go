package main

import (
	"log"

	"speech.local/packages/config"
	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

func main() {
	log.Println("🚀 開始執行資料庫遷移...")

	// 1. 載入統一的 AppConfig
	appConfig, err := config.NewAppConfig()
	if err != nil {
		log.Fatalf("無法載入設定檔: %v", err)
	}

	// 2. 取得 DB 連線
	gormDB, err := db.NewPostgresConn(appConfig.DBConfig)
	if err != nil {
		log.Fatalf("無法連線至資料庫: %v", err)
	}

	// 3. 執行 AutoMigrate
	// [修改點] 由於 gormDB 本身就是 *gorm.DB，不再需要呼叫 .GetDB()
	err = gormDB.AutoMigrate(
		&models.Task{},
		&models.OutboxEvent{},
	)

	if err != nil {
		log.Fatalf("❌ 資料庫遷移失敗: %v", err)
	}

	log.Println("✅ 資料庫 Schema 遷移與同步完成！")
}
