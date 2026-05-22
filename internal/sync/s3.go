package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"dback/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	ObjectKey     = "dback/app-data.json"
	testObjectKey = "dback/.connection-test"
)

var ErrSyncIncomplete = errors.New("endpoint, bucket, access key, and secret key are required")

// NormalizeEndpoint strips a scheme and path from an endpoint value.
func NormalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
			return u.Host
		}
	}
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func validateSettings(cfg models.SyncSettings) error {
	if NormalizeEndpoint(cfg.Endpoint) == "" ||
		strings.TrimSpace(cfg.Bucket) == "" ||
		strings.TrimSpace(cfg.AccessKeyID) == "" ||
		strings.TrimSpace(cfg.SecretKey) == "" {
		return ErrSyncIncomplete
	}
	return nil
}

func newClient(cfg models.SyncSettings) (*minio.Client, error) {
	if err := validateSettings(cfg); err != nil {
		return nil, err
	}
	endpoint := NormalizeEndpoint(cfg.Endpoint)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: strings.TrimSpace(cfg.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return client, nil
}

// TestConnection verifies bucket access and read/write permissions under dback/.
func TestConnection(ctx context.Context, cfg models.SyncSettings) error {
	client, err := newClient(cfg)
	if err != nil {
		return err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", bucket)
	}
	payload := []byte("dback-connection-test")
	if _, err := client.PutObject(ctx, bucket, testObjectKey, bytes.NewReader(payload), int64(len(payload)), minio.PutObjectOptions{
		ContentType: "text/plain",
	}); err != nil {
		return fmt.Errorf("write test object: %w", err)
	}
	if err := client.RemoveObject(ctx, bucket, testObjectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove test object: %w", err)
	}
	return nil
}

// Push uploads encrypted app data to dback/app-data.json.
func Push(ctx context.Context, cfg models.SyncSettings, data []byte) error {
	client, err := newClient(cfg)
	if err != nil {
		return err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	_, err = client.PutObject(ctx, bucket, ObjectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("upload app data: %w", err)
	}
	return nil
}

// Pull downloads encrypted app data from dback/app-data.json.
func Pull(ctx context.Context, cfg models.SyncSettings) ([]byte, error) {
	client, err := newClient(cfg)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	obj, err := client.GetObject(ctx, bucket, ObjectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download app data: %w", err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read app data: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("remote app data is empty")
	}
	return data, nil
}
