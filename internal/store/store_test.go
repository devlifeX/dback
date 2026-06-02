package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"dback/models"
)

func TestLoadProfilesMigratesLegacyFlatShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	legacy := `{
  "profiles": [{
    "id": "p1",
    "name": "Production",
    "host": "10.0.0.1",
    "port": "22",
    "ssh_user": "root",
    "ssh_password": "secret",
    "auth_type": "Password",
    "db_host": "127.0.0.1",
    "db_port": "3306",
    "db_user": "db",
    "db_password": "db-secret",
    "db_type": "MySQL",
    "target_db_name": "app",
    "destination": "/tmp"
  }]
}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	if err := s.Unlock(testMasterKey); err != nil {
		t.Fatal(err)
	}
	profiles, err := s.LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if profiles[0].Host != "10.0.0.1" {
		t.Fatalf("expected migrated host field: %#v", profiles[0])
	}
	if profiles[0].ExportSettings != nil || profiles[0].ImportSettings != nil {
		t.Fatalf("flattened profile should not keep nested settings: %#v", profiles[0])
	}
	if profiles[0].Group != "Default" {
		t.Fatalf("expected default group, got %q", profiles[0].Group)
	}
}

func TestFlattenProfileSplitsDifferentImportSettings(t *testing.T) {
	export := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "source.example.com",
		Port:           "22",
		DBType:         models.DBTypeMySQL,
		TargetDBName:   "source_db",
	}
	importSettings := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "dest.example.com",
		Port:           "2222",
		DBType:         models.DBTypeMariaDB,
		TargetDBName:   "dest_db",
	}
	profile := models.Profile{
		ID:             "p1",
		Name:           "Production",
		ExportSettings: &export,
		ImportSettings: &importSettings,
	}

	out := flattenProfile(profile)
	if len(out) != 2 {
		t.Fatalf("expected two hosts, got %d: %#v", len(out), out)
	}
	if out[0].Host != "source.example.com" {
		t.Fatalf("unexpected export host: %#v", out[0])
	}
	if out[1].Host != "dest.example.com" || out[1].Name != "Production (import)" {
		t.Fatalf("unexpected import host: %#v", out[1])
	}
}

func TestExportProfilesStripsSecretsByDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	profiles := []models.Profile{{
		ID:          "p1",
		Name:        "Production",
		SSHPassword: "ssh-secret",
		DBPassword:  "db-secret",
		WPKey:       "wp-secret",
	}}

	if err := New(dir).ExportProfiles(path, profiles, false, ""); err != nil {
		t.Fatal(err)
	}
	imported, err := New(dir).ImportProfilesBundle(path, true, "")
	if err != nil {
		t.Fatal(err)
	}
	got := imported[0]
	if got.SSHPassword != "" || got.DBPassword != "" || got.WPKey != "" {
		t.Fatalf("secrets were not stripped: %#v", got)
	}
}

func TestExportProfilesEncryptedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	profiles := []models.Profile{{
		ID:          "p1",
		Name:        "Production",
		SSHPassword: "ssh-secret",
		DBPassword:  "db-secret",
	}}

	s := New(dir)
	if err := s.ExportProfiles(path, profiles, true, "master-pass"); err != nil {
		t.Fatal(err)
	}
	imported, err := s.ImportProfilesBundle(path, true, "master-pass")
	if err != nil {
		t.Fatal(err)
	}
	if imported[0].SSHPassword != "ssh-secret" || imported[0].DBPassword != "db-secret" {
		t.Fatalf("encrypted secrets not restored: %#v", imported[0])
	}
}

func TestLoadTemplatesSeedsDefaults(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)
	templates, err := s.LoadTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) < 2 {
		t.Fatalf("expected seed templates, got %#v", templates)
	}
}

func TestDetectProfileConflicts(t *testing.T) {
	existing := []models.Profile{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
	}
	imported := []models.Profile{
		{ID: "a", Name: "Alpha-new"},
		{ID: "c", Name: "Beta"},
	}
	conflicts := DetectProfileConflicts(existing, imported)
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d: %#v", len(conflicts), conflicts)
	}
}

func TestMergeProfilesByIDAndName(t *testing.T) {
	existing := []models.Profile{{ID: "a", Name: "Alpha", Host: "old"}}
	imported := []models.Profile{
		{ID: "a", Name: "Alpha", Host: "new"},
		{ID: "b", Name: "Beta", Host: "beta"},
	}
	out := MergeProfiles(existing, imported)
	if len(out) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(out))
	}
	if out[0].Host != "new" {
		t.Fatalf("expected ID merge to update host, got %#v", out[0])
	}
}

func TestFlattenProfilesKeepsWordPress(t *testing.T) {
	profiles := []models.Profile{
		{ID: "wp", Name: "WP Site", ConnectionType: models.ConnectionTypeWordPress, WPUrl: "https://example.com", WPKey: "secret"},
		{ID: "ssh", Name: "SSH Host", ConnectionType: models.ConnectionTypeSSH, Host: "10.0.0.1"},
	}
	out := flattenProfiles(profiles)
	if len(out) != 2 {
		t.Fatalf("expected 2 profiles, got %#v", out)
	}
	if out[0].ConnectionType != models.ConnectionTypeWordPress {
		t.Fatalf("expected wordpress host to be kept, got %#v", out[0])
	}
}

func TestExportAppDataPlainAndEncrypted(t *testing.T) {
	dir := t.TempDir()
	pathPlain := filepath.Join(dir, "app-plain.json")
	pathEnc := filepath.Join(dir, "app-enc.json")
	now := time.Now()

	data := AppImportData{
		Profiles: []models.Profile{{
			ID:          "p1",
			Name:        "Production",
			SSHPassword: "ssh-secret",
			DBPassword:  "db-secret",
		}},
		Templates: []models.SQLTemplate{{ID: "t1", Name: "Reset", Body: "SELECT 1", CreatedAt: now, UpdatedAt: now}},
		History:   []models.ExportRecord{{ID: "h1", ProfileID: "p1", ProfileName: "Production", ExportDate: now}},
		Logs:      []models.LogEntry{{ID: "l1", ProfileID: "p1", ProfileName: "Production", Timestamp: now, Details: "test"}},
	}

	s := New(dir)
	if err := s.ExportAppData(pathPlain, data, false, ""); err != nil {
		t.Fatal(err)
	}
	imported, err := s.ImportAppDataBundle(pathPlain, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Profiles) != 1 || imported.Profiles[0].SSHPassword != "" {
		t.Fatalf("plain export should strip secrets: %#v", imported.Profiles[0])
	}
	if len(imported.Templates) != 1 || len(imported.History) != 1 || len(imported.Logs) != 1 {
		t.Fatalf("expected full app payload, got %#v", imported)
	}

	if err := s.ExportAppData(pathEnc, data, true, "master-pass"); err != nil {
		t.Fatal(err)
	}
	importedEnc, err := s.ImportAppDataBundle(pathEnc, true, "master-pass")
	if err != nil {
		t.Fatal(err)
	}
	if importedEnc.Profiles[0].SSHPassword != "ssh-secret" || importedEnc.Profiles[0].DBPassword != "db-secret" {
		t.Fatalf("encrypted secrets not restored: %#v", importedEnc.Profiles[0])
	}
}

func TestImportAppDataLegacyProfileBundle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy-profiles.json")
	profiles := []models.Profile{{ID: "p1", Name: "Legacy", Host: "10.0.0.1"}}
	if err := New(dir).ExportProfiles(path, profiles, false, ""); err != nil {
		t.Fatal(err)
	}
	imported, err := New(dir).ImportAppDataBundle(path, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(imported.Profiles) != 1 || imported.Profiles[0].Name != "Legacy" {
		t.Fatalf("legacy profile bundle not imported: %#v", imported)
	}
}

func TestMergeTemplatesHistoryLogs(t *testing.T) {
	existingTemplates := []models.SQLTemplate{{ID: "t1", Name: "One", Body: "old"}}
	importedTemplates := []models.SQLTemplate{{ID: "t1", Name: "One", Body: "new"}, {ID: "t2", Name: "Two", Body: "two"}}
	outTemplates := MergeTemplates(existingTemplates, importedTemplates)
	if len(outTemplates) != 2 || outTemplates[0].Body != "new" {
		t.Fatalf("template merge failed: %#v", outTemplates)
	}

	existingHistory := []models.ExportRecord{{ID: "h1"}}
	importedHistory := []models.ExportRecord{{ID: "h1"}, {ID: "h2"}}
	outHistory := MergeHistory(existingHistory, importedHistory)
	if len(outHistory) != 2 {
		t.Fatalf("history merge failed: %#v", outHistory)
	}

	existingLogs := []models.LogEntry{{ID: "l1"}}
	importedLogs := []models.LogEntry{{ID: "l1"}, {ID: "l2"}}
	outLogs := MergeLogs(existingLogs, importedLogs)
	if len(outLogs) != 2 {
		t.Fatalf("logs merge failed: %#v", outLogs)
	}
}

func TestSyncSettingsVaultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)

	settings := models.SyncSettings{
		Endpoint:    "minio.local:9000",
		Region:      "us-east-1",
		Bucket:      "dback-sync",
		AccessKeyID: "access",
		SecretKey:   "secret",
		UseSSL:      false,
	}
	if err := s.SaveSyncSettings(settings); err != nil {
		t.Fatal(err)
	}

	s2 := New(dir)
	unlockStore(t, s2)
	loaded, err := s2.LoadSyncSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Bucket != "dback-sync" || loaded.SecretKey != "secret" {
		t.Fatalf("sync settings not restored: %#v", loaded)
	}
}

func TestMarshalAppDataBundleForSync(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)

	syncSettings := models.SyncSettings{
		Endpoint:    "s3.amazonaws.com",
		Bucket:      "bucket",
		AccessKeyID: "key",
		SecretKey:   "secret",
		UseSSL:      true,
	}
	if err := s.SaveSyncSettings(syncSettings); err != nil {
		t.Fatal(err)
	}

	raw, err := s.MarshalAppDataBundleForSync(AppImportData{
		Profiles: []models.Profile{{ID: "p1", Name: "Host", SSHPassword: "pw"}},
		Sync:     syncSettings.Clone(),
	})
	if err != nil {
		t.Fatal(err)
	}
	imported, err := s.ImportAppDataBundleForSync(raw)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Profiles[0].SSHPassword != "pw" {
		t.Fatalf("profile secret not restored: %#v", imported.Profiles[0])
	}
	if imported.Sync == nil || imported.Sync.Bucket != "bucket" {
		t.Fatalf("sync settings not restored: %#v", imported.Sync)
	}
}

func TestValidateMasterPassphrase(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)
	if err := s.ValidateMasterPassphrase(testMasterKey); err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateMasterPassphrase("wrong-key"); err != ErrWrongMasterKey {
		t.Fatalf("expected ErrWrongMasterKey, got %v", err)
	}
}

func TestSyncActivityRecord(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)
	if err := s.RecordSyncPush(); err != nil {
		t.Fatal(err)
	}
	activity, err := s.LoadSyncActivity()
	if err != nil {
		t.Fatal(err)
	}
	if activity.LastPushAt.IsZero() {
		t.Fatal("expected last push time")
	}
	if err := s.RecordSyncPull(); err != nil {
		t.Fatal(err)
	}
	activity, err = s.LoadSyncActivity()
	if err != nil {
		t.Fatal(err)
	}
	if activity.LastPullAt.IsZero() {
		t.Fatal("expected last pull time")
	}
}

func TestSyncActivityVaultPersist(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)
	if err := s.RecordSyncPush(); err != nil {
		t.Fatal(err)
	}
	pushAt, err := s.LoadSyncActivity()
	if err != nil {
		t.Fatal(err)
	}
	if pushAt.LastPushAt.IsZero() {
		t.Fatal("expected last push time in memory")
	}

	s2 := New(dir)
	unlockStore(t, s2)
	activity, err := s2.LoadSyncActivity()
	if err != nil {
		t.Fatal(err)
	}
	if activity.LastPushAt.IsZero() {
		t.Fatal("expected last push time persisted in vault")
	}
	if !activity.LastPushAt.Equal(pushAt.LastPushAt) {
		t.Fatalf("expected %v, got %v", pushAt.LastPushAt, activity.LastPushAt)
	}
}

func TestSyncActivitySurvivesSaveSyncSettings(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)
	if err := s.RecordSyncPush(); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSyncSettings(models.SyncSettings{
		Endpoint:    "s3.example.com",
		Bucket:      "bucket",
		AccessKeyID: "key",
		SecretKey:   "secret",
	}); err != nil {
		t.Fatal(err)
	}
	activity, err := s.LoadSyncActivity()
	if err != nil {
		t.Fatal(err)
	}
	if activity.LastPushAt.IsZero() {
		t.Fatal("expected last push time after saving sync settings")
	}
}
