package app

import (
	"testing"
)

func TestValidatePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"09121234567", "09121234567", true},
		{"+98 912 123 4567", "09121234567", true},
		{"9121234567", "09121234567", true},
		{"123", "", false},
	}
	for _, tc := range cases {
		got, err := ValidatePhone(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestUserCRUDAndAuth(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	user, err := a.CreateUser("09121234567", "secret12", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	if user.PasswordHash != "" {
		t.Fatal("password hash should be redacted")
	}
	_, err = a.Authenticate("09121234567", "wrong")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	authUser, err := a.Authenticate("09121234567", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := a.Login(t.Context(), "09121234567", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token == "" {
		t.Fatal("expected session token")
	}
	userID, err := a.ValidateSession(result.Token)
	if err != nil || userID != authUser.ID {
		t.Fatalf("session user=%q err=%v", userID, err)
	}
	if err := a.Logout(result.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ValidateSession(result.Token); !IsSessionInvalid(err) {
		t.Fatalf("expected invalid session after logout, got %v", err)
	}
}

func TestEnsureDefaultAdmin(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	user, created, err := a.EnsureDefaultAdmin("09359922324", "09359922324", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected default admin to be created")
	}
	if user.Phone != "09359922324" {
		t.Fatalf("phone=%q", user.Phone)
	}
	_, createdAgain, err := a.EnsureDefaultAdmin("09359922324", "09359922324", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	if createdAgain {
		t.Fatal("expected no second admin creation")
	}
}
