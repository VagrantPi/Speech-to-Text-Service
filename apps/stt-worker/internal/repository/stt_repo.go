package repository

import "context"

type STTRepo interface {
	Transcribe(ctx context.Context, localFilePath string) (string, error)
}
