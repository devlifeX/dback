package app

import (
	"fmt"
	"time"

	"dback/internal/debug"
	"dback/internal/remote"
	"dback/models"
)

func remoteUploadLog(action, status, details, profileName, errText string) {
	debug.Log("DEBUG", "RemoteUpload."+action, status, details, profileName, "", errText)
}

func destinationDebugDetails(dest models.RemoteDestination) string {
	if dest.S3 == nil {
		return fmt.Sprintf("dest=%q id=%s type=%s", dest.Name, dest.ID, dest.Type)
	}
	return fmt.Sprintf(
		"dest=%q id=%s endpoint=%s bucket=%q ssl=%v",
		dest.Name,
		dest.ID,
		remote.NormalizeEndpoint(dest.S3.Endpoint),
		dest.S3.Bucket,
		dest.S3.UseSSL,
	)
}

func remoteUploadLogTimed(profileName, action, status, details string, start time.Time, err error) {
	details = fmt.Sprintf("%s elapsed=%s", details, time.Since(start).Round(time.Millisecond))
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	remoteUploadLog(action, status, details, profileName, errText)
}
