package sqlstore

import (
	"encoding/json"
	"fmt"

	"dback/internal/secrets"
	"dback/models"
)

func encryptProfile(p models.Profile, enc *secrets.FieldEncryptor) (models.Profile, error) {
	var err error
	cp := p
	if cp.SSHPassword, err = enc.Encrypt(cp.SSHPassword); err != nil {
		return models.Profile{}, err
	}
	if cp.JumpPassword, err = enc.Encrypt(cp.JumpPassword); err != nil {
		return models.Profile{}, err
	}
	if cp.DBPassword, err = enc.Encrypt(cp.DBPassword); err != nil {
		return models.Profile{}, err
	}
	if cp.AuthKeyPEM, err = enc.Encrypt(cp.AuthKeyPEM); err != nil {
		return models.Profile{}, err
	}
	if cp.JumpAuthKeyPEM, err = enc.Encrypt(cp.JumpAuthKeyPEM); err != nil {
		return models.Profile{}, err
	}
	if cp.WPKey, err = enc.Encrypt(cp.WPKey); err != nil {
		return models.Profile{}, err
	}
	return cp, nil
}

func decryptProfile(p models.Profile, enc *secrets.FieldEncryptor) (models.Profile, error) {
	var err error
	cp := p
	if cp.SSHPassword, err = enc.Decrypt(cp.SSHPassword); err != nil {
		return models.Profile{}, err
	}
	if cp.JumpPassword, err = enc.Decrypt(cp.JumpPassword); err != nil {
		return models.Profile{}, err
	}
	if cp.DBPassword, err = enc.Decrypt(cp.DBPassword); err != nil {
		return models.Profile{}, err
	}
	if cp.AuthKeyPEM, err = enc.Decrypt(cp.AuthKeyPEM); err != nil {
		return models.Profile{}, err
	}
	if cp.JumpAuthKeyPEM, err = enc.Decrypt(cp.JumpAuthKeyPEM); err != nil {
		return models.Profile{}, err
	}
	if cp.WPKey, err = enc.Decrypt(cp.WPKey); err != nil {
		return models.Profile{}, err
	}
	return cp, nil
}

func encryptDestination(d models.RemoteDestination, enc *secrets.FieldEncryptor) (models.RemoteDestination, error) {
	cp := d.Clone()
	if cp.S3 == nil {
		return cp, nil
	}
	var err error
	cp.S3.SecretKey, err = enc.Encrypt(cp.S3.SecretKey)
	if err != nil {
		return models.RemoteDestination{}, err
	}
	return cp, nil
}

func decryptDestination(d models.RemoteDestination, enc *secrets.FieldEncryptor) (models.RemoteDestination, error) {
	cp := d.Clone()
	if cp.S3 == nil {
		return cp, nil
	}
	var err error
	cp.S3.SecretKey, err = enc.Decrypt(cp.S3.SecretKey)
	if err != nil {
		return models.RemoteDestination{}, err
	}
	return cp, nil
}

func encryptSync(sync *models.SyncSettings, enc *secrets.FieldEncryptor) (*models.SyncSettings, error) {
	if sync == nil {
		return nil, nil
	}
	cp := sync.Clone()
	var err error
	cp.SecretKey, err = enc.Encrypt(cp.SecretKey)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

func decryptSync(sync *models.SyncSettings, enc *secrets.FieldEncryptor) (*models.SyncSettings, error) {
	if sync == nil {
		return nil, nil
	}
	cp := sync.Clone()
	var err error
	cp.SecretKey, err = enc.Decrypt(cp.SecretKey)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

func encryptNotifyConfig(raw json.RawMessage, enc *secrets.FieldEncryptor) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw, nil
	}
	for _, key := range []string{"token", "webhook_url", "url"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil || s == "" {
			continue
		}
		encVal, err := enc.Encrypt(s)
		if err != nil {
			return nil, err
		}
		b, _ := json.Marshal(encVal)
		m[key] = b
	}
	out, err := json.Marshal(m)
	return out, err
}

func decryptNotifyConfig(raw json.RawMessage, enc *secrets.FieldEncryptor) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw, nil
	}
	for _, key := range []string{"token", "webhook_url", "url"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil || s == "" {
			continue
		}
		decVal, err := enc.Decrypt(s)
		if err != nil {
			return nil, fmt.Errorf("notify config %s: %w", key, err)
		}
		b, _ := json.Marshal(decVal)
		m[key] = b
	}
	out, err := json.Marshal(m)
	return out, err
}

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalJSON[T any](data string, dst *T) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), dst)
}
