package sqlstore

import (
	"dback/models"

	"github.com/google/uuid"
)

const maxTaskRuns = 500

func newID() string {
	return uuid.NewString()
}

func cloneRemoteDestinations(src []models.RemoteDestination) []models.RemoteDestination {
	if len(src) == 0 {
		return nil
	}
	out := make([]models.RemoteDestination, len(src))
	for i, d := range src {
		out[i] = d.Clone()
	}
	return out
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
