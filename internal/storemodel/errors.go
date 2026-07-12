package storemodel

import "errors"

const CurrentVersion = 7

var (
	ErrVaultLocked                = errors.New("vault is locked")
	ErrVaultExists                = errors.New("vault already exists")
	ErrVaultNotFound              = errors.New("vault not found")
	ErrWrongMasterKey             = errors.New("wrong master key")
	ErrMasterKeyRequired          = errors.New("master key is required")
	ErrIncludeSecretsNoPassphrase = errors.New("passphrase required when including secrets")
	ErrLegacyPlaintextWithVault   = errors.New("legacy plaintext files found alongside encrypted vault")
	ErrSyncNotConfigured          = errors.New("sync settings are not configured")
	ErrRemoteDestinationNotFound  = errors.New("remote destination not found")
	ErrRemoteDestinationInUse     = errors.New("remote destination is in use")
	ErrAppSettingsDestRequired    = errors.New("app settings destination is required")
	ErrTaskNotFound               = errors.New("task not found")
	ErrNotifyChannelNotFound      = errors.New("notify channel not found")
	ErrUserNotFound               = errors.New("user not found")
	ErrUserExists                 = errors.New("user already exists")
	ErrInvalidCredentials         = errors.New("invalid credentials")
	ErrOTPInvalid                 = errors.New("invalid or expired otp")
	ErrSessionInvalid             = errors.New("invalid or expired session")
	ErrSquidProxyNotFound         = errors.New("squid proxy not found")
)
