package remote

import (
	"context"
	"io"
	"time"

	"dback/models"
)

// PutObjectTimeout is the per-destination upload timeout.
const PutObjectTimeout = 2 * time.Minute

// Provider abstracts remote object storage backends.
type Provider interface {
	Type() models.RemoteProviderType
	TestConnection(ctx context.Context) error
	PutObject(ctx context.Context, key string, r io.Reader, size int64, contentType string) (etag string, err error)
}

// NewProvider builds a provider for the given destination.
func NewProvider(dest models.RemoteDestination) (Provider, error) {
	switch dest.Type {
	case models.RemoteProviderS3:
		return NewS3Provider(dest)
	default:
		return nil, models.ErrRemoteDestinationUnsupportedType()
	}
}
