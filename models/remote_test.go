package models

import "testing"

func TestValidateRemoteDestination(t *testing.T) {
	err := ValidateRemoteDestination(RemoteDestination{
		Name: "Prod",
		Type: RemoteProviderS3,
		S3: &S3DestinationConfig{
			Endpoint:    "s3.amazonaws.com",
			Bucket:      "bucket",
			AccessKeyID: "key",
			SecretKey:   "secret",
		},
	})
	if err != nil {
		t.Fatalf("expected valid destination: %v", err)
	}

	err = ValidateRemoteDestination(RemoteDestination{Name: "Missing S3", Type: RemoteProviderS3})
	if err == nil {
		t.Fatal("expected error for missing s3 config")
	}
}

func TestRemoteUploadDoneForDestination(t *testing.T) {
	uploads := []RemoteUploadState{
		{DestinationID: "d1", Status: RemoteUploadDone},
	}
	if !RemoteUploadDoneForDestination(uploads, "d1") {
		t.Fatal("expected done for d1")
	}
	if RemoteUploadDoneForDestination(uploads, "d2") {
		t.Fatal("expected not done for d2")
	}
}

func TestRemoteDestinationClone(t *testing.T) {
	src := RemoteDestination{
		ID:   "id",
		Name: "Name",
		Type: RemoteProviderS3,
		S3:   &S3DestinationConfig{SecretKey: "secret"},
	}
	cp := src.Clone()
	if cp.S3 == src.S3 {
		t.Fatal("expected deep clone of s3 config")
	}
	if cp.S3.SecretKey != "secret" {
		t.Fatal("expected secret preserved in clone")
	}
}
