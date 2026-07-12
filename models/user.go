package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Phone        string    `json:"phone"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"password_hash,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
