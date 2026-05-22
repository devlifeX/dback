package secrets

import (
	"testing"

	"dback/models"
)

func TestEncryptAppBundleWithSyncSettings(t *testing.T) {
	syncSettings := &models.SyncSettings{
		Endpoint:    "s3.amazonaws.com",
		Region:      "us-east-1",
		Bucket:      "my-bucket",
		AccessKeyID: "access-key",
		SecretKey:   "secret-key",
		UseSSL:      true,
	}
	bundle, err := EncryptAppBundle(nil, nil, nil, nil, syncSettings, "master-key-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.Encrypted {
		t.Fatal("expected encrypted bundle")
	}
	decoded, err := DecryptAppBundle(bundle, "master-key-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Sync == nil || decoded.Sync.Bucket != "my-bucket" {
		t.Fatalf("sync settings not restored: %#v", decoded.Sync)
	}
}

func TestDecryptAppBundleWithoutSyncBackwardCompatible(t *testing.T) {
	bundle, err := EncryptAppBundle(
		[]models.Profile{{ID: "p1", Name: "host", SSHPassword: "secret"}},
		nil, nil, nil, nil, "master-key-12345678",
	)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecryptAppBundle(bundle, "master-key-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Sync != nil {
		t.Fatalf("expected nil sync settings, got %#v", decoded.Sync)
	}
	if decoded.Profiles[0].SSHPassword != "secret" {
		t.Fatal("profile secret not restored")
	}
}
