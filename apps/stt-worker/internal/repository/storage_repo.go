package repository

import "context"

type StorageRepo interface {
	DownloadToTempFile(ctx context.Context, s3Key string) (string, error)
}
