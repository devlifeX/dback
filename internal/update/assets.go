package update

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	platformGOOS   = runtime.GOOS
	platformGOARCH = runtime.GOARCH
)

func PickAsset(release Release) (Asset, error) {
	switch platformGOOS {
	case "linux":
		if platformGOARCH != "amd64" {
			return Asset{}, fmt.Errorf("automatic updates are only supported on linux/amd64 (got %s/%s)", platformGOOS, platformGOARCH)
		}
		want := fmt.Sprintf("dback_%s_amd64.deb", release.Version)
		for _, asset := range release.Assets {
			if asset.Name == want {
				return asset, nil
			}
		}
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasPrefix(name, "dback_") && strings.HasSuffix(name, "_amd64.deb") {
				return asset, nil
			}
		}
	case "windows":
		if platformGOARCH != "amd64" {
			return Asset{}, fmt.Errorf("automatic updates are only supported on windows/amd64 (got %s/%s)", platformGOOS, platformGOARCH)
		}
		for _, asset := range release.Assets {
			if asset.Name == "dback-windows.exe" {
				return asset, nil
			}
		}
	default:
		return Asset{}, fmt.Errorf("automatic updates are not supported on %s/%s", platformGOOS, platformGOARCH)
	}

	return Asset{}, fmt.Errorf("no update asset found for %s/%s in release %s", platformGOOS, platformGOARCH, release.Version)
}
