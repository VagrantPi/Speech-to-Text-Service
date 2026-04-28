package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"speech.local/packages/db/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.OutboxEvent{}, &models.Task{})
	require.NoError(t, err)

	return db
}

func mustCreate(t *testing.T, db *gorm.DB, value interface{}) {
	result := db.Create(value)
	require.NoError(t, result.Error)
}