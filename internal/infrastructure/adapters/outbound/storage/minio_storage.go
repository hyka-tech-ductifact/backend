package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements ports.FileStorage using MinIO (S3-compatible).
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIOStorage creates a new MinIO client and ensures the bucket exists.
// endpoint: "localhost:9000", useSSL: false for local dev.
func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	// Ensure bucket exists (idempotent)
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("checking bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("creating bucket %q: %w", bucket, err)
		}
		slog.Info("minio bucket created", "bucket", bucket)
	}

	return &MinIOStorage{client: client, bucket: bucket}, nil
}

// Upload stores a file in MinIO under the given key.
func (s *MinIOStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio upload %q: %w", key, err)
	}
	return nil
}

// Delete removes a file from MinIO. Returns nil if the file doesn't exist.
func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio delete %q: %w", key, err)
	}
	return nil
}
