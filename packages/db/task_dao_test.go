package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"speech.local/packages/db/models"
)

func setupTaskDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Task{})
	require.NoError(t, err)

	return db
}

func TestTaskDAO_Create(t *testing.T) {
	db := setupTaskDB(t)
	dao := NewTaskDAO(db)

	task := &models.Task{S3Key: "uploads/audio/test.mp3", Status: "CREATED"}

	t.Run("success", func(t *testing.T) {
		err := dao.Create(db, task)
		assert.NoError(t, err)
		assert.NotZero(t, task.ID)
	})
}

func TestTaskDAO_FindByID(t *testing.T) {
	db := setupTaskDB(t)
	dao := NewTaskDAO(db)

	mustCreate(t, db, &models.Task{S3Key: "uploads/audio/test.mp3", Status: "CREATED"})

	t.Run("success", func(t *testing.T) {
		task, err := dao.FindByID(context.Background(), 1)
		assert.NoError(t, err)
		assert.Equal(t, "uploads/audio/test.mp3", task.S3Key)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := dao.FindByID(context.Background(), 999)
		assert.Error(t, err)
	})
}

func TestTaskDAO_Update(t *testing.T) {
	db := setupTaskDB(t)
	dao := NewTaskDAO(db)

	mustCreate(t, db, &models.Task{S3Key: "uploads/audio/test.mp3", Status: "CREATED"})

	t.Run("success", func(t *testing.T) {
		task := &models.Task{ID: 1, Status: "PROCESSING"}
		err := dao.Update(context.Background(), task)
		assert.NoError(t, err)

		var updated models.Task
		db.First(&updated, 1)
		assert.Equal(t, "PROCESSING", updated.Status)
	})
}