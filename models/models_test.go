package models

import "testing"

func TestSubstituteQuery(t *testing.T) {
	query := "USE {databasename}; SELECT '{host}', '{profile}', '{dbuser}';"
	got := SubstituteQuery(query, QueryVars{
		DatabaseName: "app_db",
		Host:         "10.0.0.1",
		Profile:      "Production",
		DBUser:       "root",
	})
	want := "USE app_db; SELECT '10.0.0.1', 'Production', 'root';"
	if got != want {
		t.Fatalf("unexpected substitution:\n got: %q\nwant: %q", got, want)
	}
}

func TestSettingsEqualIgnoresSecrets(t *testing.T) {
	a := &TransferSettings{Host: "a", SSHPassword: "secret"}
	b := &TransferSettings{Host: "a", SSHPassword: "other"}
	if !SettingsEqual(a, b) {
		t.Fatal("settings with different secrets should still be equal")
	}
	if !SettingsEqual(nil, nil) {
		t.Fatal("nil settings should be equal")
	}
	if SettingsEqual(a, nil) {
		t.Fatal("nil mismatch should not be equal")
	}
}

func TestNormalizeDBType(t *testing.T) {
	if got := normalizeDBType(DBTypeMariaDB); got != DBTypeMariaDB {
		t.Fatalf("expected MariaDB, got %q", got)
	}
	if got := normalizeDBType("PostgreSQL"); got != DBTypeMySQL {
		t.Fatalf("non-mysql types should normalize to MySQL, got %q", got)
	}
}
