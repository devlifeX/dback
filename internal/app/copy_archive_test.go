package app

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"dback/backend/shell"
	"dback/models"
)

func writeTestPayload(path string) error {
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i*31 + 17)
	}
	return os.WriteFile(path, payload, 0644)
}

type mockArchiveConnector struct {
	payload []byte
	waitErr error
}

func (m *mockArchiveConnector) Close() error { return nil }

func (m *mockArchiveConnector) Run(ctx context.Context, plan shell.ExecutionPlan) (*shell.StreamResult, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		_, _ = pw.Write(m.payload)
	}()
	wait := connectorOnceWait(m.waitErr)
	return &shell.StreamResult{
		Reader: readNopCloser{Reader: pr, closeFn: wait},
		Wait:   wait,
	}, nil
}

type readNopCloser struct {
	io.Reader
	closeFn func() error
}

func (r readNopCloser) Close() error {
	if r.closeFn != nil {
		return r.closeFn()
	}
	return nil
}

func connectorOnceWait(err error) func() error {
	called := false
	return func() error {
		if called {
			return err
		}
		called = true
		return err
	}
}

func TestCopyArchiveReturnsWaitError(t *testing.T) {
	a := &App{}
	dest := filepath.Join(t.TempDir(), "out.partial")
	conn := &mockArchiveConnector{payload: []byte("archive-bytes"), waitErr: io.ErrUnexpectedEOF}
	_, err := a.copyArchive(context.Background(), conn, shell.ExecutionPlan{}, dest, nil)
	if err == nil {
		t.Fatal("expected wait error")
	}
	if _, statErr := os.Stat(dest); statErr != nil {
		t.Fatalf("expected partial file written before wait failure: %v", statErr)
	}
}

func TestBackupFilesStopOnFirstFailure(t *testing.T) {
	if _, err := execLookPath("tar"); err != nil {
		t.Skip("tar not available")
	}
	if _, err := execLookPath("zstd"); err != nil {
		t.Skip("zstd not available")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeTestPayload(filepath.Join(src, "nested", "file.txt")); err != nil {
		t.Fatal(err)
	}

	a := openApp(t, dir)
	profile := models.Profile{
		ID:                "host-local",
		Name:              "Local",
		Group:             "Default",
		ConnectionType:    models.ConnectionTypeLocalhost,
		FileBackupEnabled: true,
		FileBackupPaths: []models.FileBackupPath{
			{ID: "p1", Name: "Good", RemotePath: src},
			{ID: "p2", Name: "Bad", RemotePath: "/this/path/does/not/exist"},
		},
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	result, err := a.BackupFiles(context.Background(), profile, nil)
	if err == nil {
		t.Fatal("expected backup error")
	}
	if !result.PartialFail {
		t.Fatal("expected partial failure")
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected one successful record before stop, got %d", len(result.Records))
	}
	if result.Records[0].FileBackupPathID != "p1" {
		t.Fatalf("unexpected path id: %q", result.Records[0].FileBackupPathID)
	}
	if result.OperationID == "" {
		t.Fatal("expected operation id")
	}
	for _, rec := range result.Records {
		if rec.OperationID != result.OperationID {
			t.Fatalf("records must share operation id: %#v", result.Records)
		}
	}
}

func TestBackupFilesGroupsRecordsByOperationID(t *testing.T) {
	if _, err := execLookPath("tar"); err != nil {
		t.Skip("tar not available")
	}
	if _, err := execLookPath("zstd"); err != nil {
		t.Skip("zstd not available")
	}

	dir := t.TempDir()
	srcA := filepath.Join(dir, "a")
	srcB := filepath.Join(dir, "b")
	for _, p := range []string{srcA, srcB} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
		if err := writeTestPayload(filepath.Join(p, "file.txt")); err != nil {
			t.Fatal(err)
		}
	}

	a := openApp(t, dir)
	profile := models.Profile{
		ID:                "host-local",
		Name:              "Local",
		Group:             "Default",
		ConnectionType:    models.ConnectionTypeLocalhost,
		FileBackupEnabled: true,
		FileBackupPaths: []models.FileBackupPath{
			{ID: "p1", Name: "Alpha", RemotePath: srcA},
			{ID: "p2", Name: "Beta", RemotePath: srcB},
		},
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	result, err := a.BackupFiles(context.Background(), profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result.Records))
	}
	op := result.OperationID
	if op == "" {
		t.Fatal("missing operation id")
	}
	for _, rec := range result.Records {
		if rec.OperationID != op {
			t.Fatalf("operation id mismatch: %#v", rec)
		}
		if rec.ExportType != models.ExportTypeFiles {
			t.Fatalf("expected files export type: %#v", rec)
		}
		if rec.FileBackupPathID == "" || rec.SourceLabel == "" {
			t.Fatalf("missing snapshot fields: %#v", rec)
		}
	}
}

func execLookPath(name string) (string, error) {
	return osExecLookPath(name)
}

var osExecLookPath = func(name string) (string, error) {
	return "", nil
}

func init() {
	osExecLookPath = func(name string) (string, error) {
		return exec.LookPath(name)
	}
}
