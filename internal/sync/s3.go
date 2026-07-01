package sync

import (
	"context"

	"dback/models"

	"dback/internal/remote"
)

const ObjectKey = remote.AppDataObjectKey

var ErrSyncIncomplete = remote.ErrIncomplete

// NormalizeEndpoint strips a scheme and path from an endpoint value.
func NormalizeEndpoint(endpoint string) string {
	return remote.NormalizeEndpoint(endpoint)
}

func destinationFromSettings(cfg models.SyncSettings) models.RemoteDestination {
	return remote.SyncSettingsToDestination(cfg)
}

// TestConnection verifies bucket access and read/write permissions under dback/.
func TestConnection(ctx context.Context, cfg models.SyncSettings) error {
	dest := destinationFromSettings(cfg)
	provider, err := remote.NewProvider(dest)
	if err != nil {
		return err
	}
	return provider.TestConnection(ctx)
}

// TestDestinationConnection verifies access for a remote destination.
func TestDestinationConnection(ctx context.Context, dest models.RemoteDestination) error {
	provider, err := remote.NewProvider(dest)
	if err != nil {
		return err
	}
	return provider.TestConnection(ctx)
}

// Push uploads encrypted app data to dback/app-data.json.
func Push(ctx context.Context, cfg models.SyncSettings, data []byte) error {
	dest := destinationFromSettings(cfg)
	return remote.PushAppData(ctx, dest, data)
}

// PushDestination uploads encrypted app data using a remote destination.
func PushDestination(ctx context.Context, dest models.RemoteDestination, data []byte) error {
	return remote.PushAppData(ctx, dest, data)
}

// Pull downloads encrypted app data from dback/app-data.json.
func Pull(ctx context.Context, cfg models.SyncSettings) ([]byte, error) {
	dest := destinationFromSettings(cfg)
	return remote.PullAppData(ctx, dest)
}

// PullDestination downloads encrypted app data using a remote destination.
func PullDestination(ctx context.Context, dest models.RemoteDestination) ([]byte, error) {
	return remote.PullAppData(ctx, dest)
}
