package app

import (
	"context"
	"net/http"
	"time"

	"dback/internal/update"
)

const updateHTTPTimeout = 2 * time.Minute

// UpdateInfo describes whether a newer release is available.
type UpdateInfo = update.Info

func (a *App) updateHTTPClient() *http.Client {
	return &http.Client{Timeout: updateHTTPTimeout}
}

// CheckForUpdate queries GitHub Releases for a newer version.
func (a *App) CheckForUpdate(ctx context.Context, currentVersion string) (UpdateInfo, error) {
	return update.Check(ctx, a.updateHTTPClient(), updateUserAgent(currentVersion), currentVersion)
}

// ApplyUpdate downloads and installs the selected release asset for this OS.
func (a *App) ApplyUpdate(ctx context.Context, info UpdateInfo, progress func(string)) error {
	return update.Apply(ctx, info, progress, nil)
}

func updateUserAgent(currentVersion string) string {
	return "DBack/" + update.NormalizeVersion(currentVersion)
}
