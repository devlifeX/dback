package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"dback/models"
)

// DeriveKey derives a 32-byte AES key from passphrase and salt using Argon2id.
func DeriveKey(passphrase string, salt []byte) []byte {
	return deriveKey(passphrase, salt)
}

// EncryptWithKey encrypts plaintext with AES-GCM using the provided key.
func EncryptWithKey(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, plaintext, nil), nil
}

// DecryptWithKey decrypts AES-GCM ciphertext with the provided key.
func DecryptWithKey(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong key or corrupted data")
	}
	return plain, nil
}

// MarshalEncryptVault serializes payload and encrypts it with key.
func MarshalEncryptVault(key []byte, payload models.AppVaultPayload) (nonce, ciphertext []byte, err error) {
	plain, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	return EncryptWithKey(key, plain)
}

// DecryptUnmarshalVault decrypts ciphertext and unmarshals into payload.
func DecryptUnmarshalVault(key, nonce, ciphertext []byte) (models.AppVaultPayload, error) {
	plain, err := DecryptWithKey(key, nonce, ciphertext)
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	var payload models.AppVaultPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return models.AppVaultPayload{}, fmt.Errorf("invalid vault payload: %w", err)
	}
	return payload, nil
}
