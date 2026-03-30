package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"backend/internal/config"
)

// Client abstracts object storage operations used by the app.
type Client struct {
	minio  *minio.Client
	bucket string
}

// NewClient creates a MinIO-backed storage client.
func NewClient(cfg config.StorageConfig) (*Client, error) {
	credentialProvider := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentialProvider,
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &Client{minio: minioClient, bucket: cfg.Bucket}, nil
}

// Upload stores the provided object and returns its storage path.
func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	if objectName == "" {
		objectName = generateObjectName()
	}

	_, err := c.minio.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload object: %w", err)
	}

	return fmt.Sprintf("%s/%s", c.bucket, objectName), nil
}

// Presign returns a temporary URL to access the object.
func (c *Client) Presign(ctx context.Context, objectPath string) (*url.URL, error) {
	bucket, object := splitPath(objectPath)
	reqParams := make(url.Values)
	return c.minio.PresignedGetObject(ctx, bucket, object, presignDuration, reqParams)
}

// Delete removes the object if exists.
func (c *Client) Delete(ctx context.Context, objectPath string) error {
	bucket, object := splitPath(objectPath)
	return c.minio.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
}

const presignDuration = 5 * time.Minute

func generateObjectName() string {
	return path.Join("uploads", uuid.NewString())
}

func splitPath(fullPath string) (bucket, object string) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
