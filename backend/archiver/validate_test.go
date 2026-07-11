package archiver

import (
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"dback/models"

	"github.com/klauspost/compress/zstd"
)

func TestValidateIntegrityGzip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIntegrity(path, models.ArchiveCompressionGzip); err != nil {
		t.Fatal(err)
	}
}

func TestValidateIntegrityZstd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.tar.zst")
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIntegrity(path, models.ArchiveCompressionZstd); err != nil {
		t.Fatal(err)
	}
}

func TestValidateIntegrityRejectsTruncatedGzip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(path, []byte{0x1f, 0x8b, 0x08}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIntegrity(path, models.ArchiveCompressionGzip); err == nil {
		t.Fatal("expected integrity error")
	}
}

func TestValidateIntegrityUsesZstdBinaryWhenAvailable(t *testing.T) {
	if _, err := exec.LookPath("zstd"); err != nil {
		t.Skip("zstd binary not available")
	}
	path := filepath.Join(t.TempDir(), "sample.tar.zst")
	cmd := exec.Command("sh", "-c", "printf payload | zstd -1 -o "+path)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIntegrity(path, models.ArchiveCompressionZstd); err != nil {
		t.Fatal(err)
	}
}
