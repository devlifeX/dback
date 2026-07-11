package secrets

import (
	"encoding/base64"
	"errors"
	"strings"
)

const fieldEncPrefix = "enc:"

var ErrFieldNotEncrypted = errors.New("encryption key not available")

// FieldEncryptor encrypts individual secret strings at rest using AES-GCM.
type FieldEncryptor struct {
	key []byte
}

func NewFieldEncryptor(key []byte) *FieldEncryptor {
	return &FieldEncryptor{key: append([]byte(nil), key...)}
}

func (f *FieldEncryptor) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if f == nil || len(f.key) == 0 {
		return "", ErrFieldNotEncrypted
	}
	nonce, ciphertext, err := EncryptWithKey(f.key, []byte(plain))
	if err != nil {
		return "", err
	}
	buf := append(nonce, ciphertext...)
	return fieldEncPrefix + base64.StdEncoding.EncodeToString(buf), nil
}

func (f *FieldEncryptor) Decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, fieldEncPrefix) {
		return stored, nil
	}
	if f == nil || len(f.key) == 0 {
		return "", ErrFieldNotEncrypted
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, fieldEncPrefix))
	if err != nil {
		return "", err
	}
	nonceSize := 12 // AES-GCM standard nonce size used by EncryptWithKey
	if len(raw) < nonceSize {
		return "", errors.New("invalid encrypted field")
	}
	plain, err := DecryptWithKey(f.key, raw[:nonceSize], raw[nonceSize:])
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
