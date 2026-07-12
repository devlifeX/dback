package models

import (
	"encoding/json"
	"time"
)

type SMSProvider string

const (
	SMSProviderKavenegar   SMSProvider = "kavenegar"
	SMSProviderMeliPayamak SMSProvider = "melipayamak"
)

type AuthSettings struct {
	TwoFactorEnabled bool            `json:"two_factor_enabled"`
	SMSProvider      SMSProvider     `json:"sms_provider,omitempty"`
	SMSConfig        json.RawMessage `json:"sms_config,omitempty"`
	OTPTTLSeconds    int             `json:"otp_ttl_seconds"`
	OTPLength        int             `json:"otp_length"`
}

func DefaultAuthSettings() AuthSettings {
	return AuthSettings{
		TwoFactorEnabled: false,
		OTPTTLSeconds:    300,
		OTPLength:        6,
	}
}

type Session struct {
	TokenHash string    `json:"-"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type OTPChallenge struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CodeHash  string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Attempts  int       `json:"attempts"`
}
