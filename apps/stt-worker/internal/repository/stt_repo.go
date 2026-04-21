package repository

import "context"

// STTService 是 stt-worker 專屬的微型介面
type STTService interface {
	Transcribe(ctx context.Context, localFilePath string) (string, error)
}
