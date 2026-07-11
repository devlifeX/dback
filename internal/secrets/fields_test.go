package secrets

import "testing"

func TestFieldEncryptorRoundTrip(t *testing.T) {
	key := DeriveKey("test-passphrase", []byte("1234567890123456"))
	enc := NewFieldEncryptor(key)
	cipher, err := enc.Encrypt("super-secret")
	if err != nil {
		t.Fatal(err)
	}
	if cipher == "super-secret" {
		t.Fatal("expected encrypted value")
	}
	plain, err := enc.Decrypt(cipher)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "super-secret" {
		t.Fatalf("got %q", plain)
	}
}

func TestFieldEncryptorWrongKey(t *testing.T) {
	key1 := DeriveKey("pass-one", []byte("1234567890123456"))
	key2 := DeriveKey("pass-two", []byte("1234567890123456"))
	cipher, err := NewFieldEncryptor(key1).Encrypt("token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewFieldEncryptor(key2).Decrypt(cipher); err == nil {
		t.Fatal("expected decrypt failure with wrong key")
	}
}
