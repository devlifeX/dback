package dbackdbtools

import "embed"

// Files contains the WordPress plugin template embedded in the Go binary.
//
//go:embed dback-db-tools.php index.php assets includes vendor
var Files embed.FS

const (
	PluginSlug        = "dback-db-tools"
	APIKeyPlaceholder = "{{DBACK_API_KEY}}"
	VersionConstant   = "DBACK_DB_TOOLS_VERSION"
)
