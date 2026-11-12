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

// S3Storage implements ports.FileStorage against any S3-compatible server.
// minio-go is used only as a generic S3 client; it does not require a MinIO server.
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage creates an S3 client and ensures the bucket exists.
// endpoint: "localhost:9000", useSSL: false for local dev.
func NewS3Storage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
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
		slog.Info("s3 bucket created", "bucket", bucket)
	}

	return &S3Storage{client: client, bucket: bucket}, nil
}

// Upload stores a file under the given key.
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("s3 upload %q: %w", key, err)
	}
	return nil
}

// GetObject retrieves a file by its key.
// Returns ports.ErrFileNotFound if the key doesn't exist.
func (s *S3Storage) GetObject(ctx context.Context, key string) (*ports.FileObject, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3 get %q: %w", key, err)
	}

	// Stat to check existence and get metadata
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		// S3 servers report missing/invalid objects with different codes (including provider-specific X* codes)
		errResp := minio.ErrorResponse{}
		if errors.As(err, &errResp) {
			switch errResp.Code {
			case "NoSuchKey", "NoSuchBucket", "InvalidObjectName",
				"XMinioInvalidObjectName", "XAmzInvalidObjectName",
				"KeyTooLongError", "InvalidArgument":
				return nil, ports.ErrFileNotFound
			}
		}
		return nil, fmt.Errorf("s3 stat %q: %w", key, err)
	}

	return &ports.FileObject{
		Reader:      obj,
		ContentType: info.ContentType,
		Size:        info.Size,
	}, nil
}

// Delete removes a file. Returns nil if the file doesn't exist.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3 delete %q: %w", key, err)
	}
	return nil
}

// Ping checks whether the S3 server is reachable by verifying the bucket exists.
func (s *S3Storage) Ping(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("s3 ping: %w", err)
	}
	return nil
}
