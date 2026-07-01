package remote

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"dback/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	AppDataObjectKey = "dback/app-data.json"
	testObjectKey    = "dback/.connection-test"
)

var ErrIncomplete = fmt.Errorf("endpoint, bucket, access key, and secret key are required")

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

type S3Provider struct {
	dest   models.RemoteDestination
	cfg    models.S3DestinationConfig
	client *minio.Client
}

func NewS3Provider(dest models.RemoteDestination) (*S3Provider, error) {
	if dest.S3 == nil {
		return nil, ErrIncomplete
	}
	cfg := *dest.S3
	if NormalizeEndpoint(cfg.Endpoint) == "" ||
		strings.TrimSpace(cfg.Bucket) == "" ||
		strings.TrimSpace(cfg.AccessKeyID) == "" ||
		strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, ErrIncomplete
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
	return &S3Provider{dest: dest, cfg: cfg, client: client}, nil
}

func (p *S3Provider) Type() models.RemoteProviderType {
	return models.RemoteProviderS3
}

func (p *S3Provider) bucket() string {
	return strings.TrimSpace(p.cfg.Bucket)
}

// TestConnection verifies bucket access and read/write permissions under dback/.
func (p *S3Provider) TestConnection(ctx context.Context) error {
	bucket := p.bucket()
	exists, err := p.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", bucket)
	}
	payload := []byte("dback-connection-test")
	if _, err := p.client.PutObject(ctx, bucket, testObjectKey, bytes.NewReader(payload), int64(len(payload)), minio.PutObjectOptions{
		ContentType: "text/plain",
	}); err != nil {
		return fmt.Errorf("write test object: %w", err)
	}
	if err := p.client.RemoveObject(ctx, bucket, testObjectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove test object: %w", err)
	}
	return nil
}

func (p *S3Provider) PutObject(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	info, err := p.client.PutObject(ctx, p.bucket(), key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload object: %w", err)
	}
	return info.ETag, nil
}

// PushAppData uploads encrypted app settings bundle.
func PushAppData(ctx context.Context, dest models.RemoteDestination, data []byte) error {
	provider, err := NewProvider(dest)
	if err != nil {
		return err
	}
	_, err = provider.PutObject(ctx, AppDataObjectKey, bytes.NewReader(data), int64(len(data)), "application/json")
	return err
}

// PullAppData downloads encrypted app settings bundle.
func PullAppData(ctx context.Context, dest models.RemoteDestination) ([]byte, error) {
	p, err := NewS3Provider(dest)
	if err != nil {
		return nil, err
	}
	obj, err := p.client.GetObject(ctx, p.bucket(), AppDataObjectKey, minio.GetObjectOptions{})
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

// SyncSettingsToDestination converts legacy sync settings to a destination value.
func SyncSettingsToDestination(cfg models.SyncSettings) models.RemoteDestination {
	return models.RemoteDestinationFromSyncSettings("legacy", "Legacy", cfg)
}
