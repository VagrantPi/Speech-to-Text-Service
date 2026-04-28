package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"speech.local/packages/db/models"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	require.NoError(t, err)

	closeFn := func() { db.Close() }
	return gormDB, mock, closeFn
}

func TestTaskRepo_GetByID(t *testing.T) {
	gormDB, mock, closeFn := setupMockDB(t)
	defer closeFn()
	repo := NewTaskRepo(gormDB)

	tests := []struct {
		name      string
		taskID    uint
		setupMock func(sqlmock.Sqlmock)
		wantTask  *models.Task
		wantErr  error
	}{
		{
			name:   "success",
			taskID: 1,
			setupMock: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "status", "s3_key", "transcript", "summary"}).
					AddRow(1, "COMPLETED", "uploads/audio/test.mp3", "Hello", "Summary")
				m.ExpectQuery("SELECT \\* FROM `task` WHERE `task`\\.`id` = .+").
					WillReturnRows(rows)
			},
			wantTask: &models.Task{
				ID:         1,
				Status:     "COMPLETED",
				S3Key:      "uploads/audio/test.mp3",
				Transcript: "Hello",
				Summary:    "Summary",
			},
		},
		{
			name:   "not found",
			taskID: 999,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT \\* FROM `task` WHERE `task`\\.`id` = .+").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			task, err := repo.GetByID(context.Background(), tt.taskID)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTask.ID, task.ID)
				assert.Equal(t, tt.wantTask.Status, task.Status)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}