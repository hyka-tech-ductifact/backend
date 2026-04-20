package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"ductifact/internal/application/ports"

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

// GetObject retrieves a file from MinIO by its key.
// Returns ports.ErrFileNotFound if the key doesn't exist.
func (s *MinIOStorage) GetObject(ctx context.Context, key string) (*ports.FileObject, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio get %q: %w", key, err)
	}

	// Stat to check existence and get metadata
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		// MinIO returns an ErrorResponse with Code "NoSuchKey" for missing objects
		errResp := minio.ErrorResponse{}
		if errors.As(err, &errResp) && errResp.Code == "NoSuchKey" {
			return nil, ports.ErrFileNotFound
		}
		return nil, fmt.Errorf("minio stat %q: %w", key, err)
	}

	return &ports.FileObject{
		Reader:      obj,
		ContentType: info.ContentType,
		Size:        info.Size,
	}, nil
}

// Delete removes a file from MinIO. Returns nil if the file doesn't exist.
func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio delete %q: %w", key, err)
	}
	return nil
}
