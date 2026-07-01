package connector

import (
	"fmt"

	"dback/models"
)

// NewConnector is the only place that maps profile connection type to transport.
func NewConnector(profile models.Profile) (Connector, error) {
	switch profile.ConnectionType {
	case models.ConnectionTypeLocalhost:
		return newLocalConnector(), nil
	case models.ConnectionTypeSSH, models.ConnectionTypeJumpHost:
		return newSSHConnector(profile)
	default:
		return nil, fmt.Errorf("file backup is not supported for connection type %q", profile.ConnectionType)
	}
}
