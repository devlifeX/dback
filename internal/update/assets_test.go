package update

import (
	"runtime"
	"testing"
)

func TestPickAssetLinuxDeb(t *testing.T) {
	t.Parallel()

	platformGOOS = "linux"
	platformGOARCH = "amd64"
	t.Cleanup(func() {
		platformGOOS = runtime.GOOS
		platformGOARCH = runtime.GOARCH
	})

	release := Release{
		Version: "3.6.1",
		Assets: []Asset{
			{Name: "dback-linux", URL: "https://example.com/dback-linux"},
			{Name: "dback_3.6.1_amd64.deb", URL: "https://example.com/dback.deb"},
		},
	}

	asset, err := PickAsset(release)
	if err != nil {
		t.Fatalf("PickAsset: %v", err)
	}
	if asset.Name != "dback_3.6.1_amd64.deb" {
		t.Fatalf("got %q", asset.Name)
	}
}

func TestPickAssetWindowsExe(t *testing.T) {
	t.Parallel()

	platformGOOS = "windows"
	platformGOARCH = "amd64"
	t.Cleanup(func() {
		platformGOOS = runtime.GOOS
		platformGOARCH = runtime.GOARCH
	})

	release := Release{
		Version: "3.6.1",
		Assets: []Asset{
			{Name: "dback-windows.exe", URL: "https://example.com/dback-windows.exe"},
		},
	}

	asset, err := PickAsset(release)
	if err != nil {
		t.Fatalf("PickAsset: %v", err)
	}
	if asset.Name != "dback-windows.exe" {
		t.Fatalf("got %q", asset.Name)
	}
}