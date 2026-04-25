package repository

import "context"

type StorageRepo interface {
	GenerateUploadURL(ctx context.Context, objectKey, contentType string) (string, error)
	EnsureBucket(ctx context.Context) error
}
