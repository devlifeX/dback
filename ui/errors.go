package ui

import (
	"dback/internal/debug"
	"dback/internal/store"
	"errors"
	"strings"
)

const maxErrorMessageLen = 240

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	if debug.Enabled {
		return truncateError(err.Error(), maxErrorMessageLen*4)
	}
	switch {
	case errors.Is(err, store.ErrWrongMasterKey):
		return "Wrong master key."
	case errors.Is(err, store.ErrMasterKeyRequired):
		return "Master key is required."
	case errors.Is(err, store.ErrVaultLocked):
		return "Vault is locked."
	case errors.Is(err, store.ErrIncludeSecretsNoPassphrase):
		return "Passphrase is required when exporting secrets."
	case errors.Is(err, store.ErrLegacyPlaintextWithVault):
		return "Legacy plaintext files were found alongside the encrypted vault and could not be removed safely."
	case errors.Is(err, store.ErrSyncNotConfigured):
		return "Configure and save S3 sync settings first."
	default:
		msg := err.Error()
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "decryption failed") {
			return "Wrong master key or corrupted remote data."
		}
		if strings.Contains(lower, "create host backup folder") ||
			strings.Contains(lower, "backup folder") && strings.Contains(lower, "permission denied") {
			return "Cannot write to the backup folder. Check the Destination Folder path and permissions."
		}
		if strings.Contains(lower, "password") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "key") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "denied") ||
			strings.Contains(lower, "authentication") {
			return "Operation failed due to an authentication or credential error."
		}
		return truncateError(msg, maxErrorMessageLen)
	}
}

func truncateError(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if max <= 0 || len(msg) <= max {
		return msg
	}
	return msg[:max-3] + "..."
}
