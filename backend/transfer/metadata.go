package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const metaSuffix = ".dback-meta.json"

type FileMeta struct {
	OperationID string `json:"operation_id"`
	RemotePath  string `json:"remote_path"`
	LocalPath   string `json:"local_path"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum,omitempty"`
	Compression string `json:"compression,omitempty"`
	Offset      int64  `json:"offset,omitempty"`
}

func metaPathFor(localPath string) string {
	return localPath + metaSuffix
}

func loadMeta(localPath string) (FileMeta, bool) {
	data, err := os.ReadFile(metaPathFor(localPath))
	if err != nil {
		return FileMeta{}, false
	}
	var meta FileMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return FileMeta{}, false
	}
	return meta, true
}

func saveMeta(meta FileMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPathFor(meta.LocalPath), data, 0600)
}

func removeMeta(localPath string) {
	_ = os.Remove(metaPathFor(localPath))
}

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validateLocalFile(path string, expectedSize int64, expectedChecksum string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if expectedSize > 0 && info.Size() != expectedSize {
		return fmt.Errorf("size mismatch: got %d want %d", info.Size(), expectedSize)
	}
	if expectedChecksum == "" {
		return nil
	}
	sum, err := checksumFile(path)
	if err != nil {
		return err
	}
	if sum != expectedChecksum {
		return fmt.Errorf("checksum mismatch")
	}
	return nil
}

func localResumeOffset(localPath string, meta FileMeta, remoteSize int64) int64 {
	if st, err := os.Stat(localPath); err == nil && st.Size() > 0 {
		if remoteSize > 0 && st.Size() < remoteSize {
			return st.Size()
		}
		if meta.Offset > 0 && st.Size() == meta.Offset {
			return meta.Offset
		}
	}
	return 0
}

func openLocalAppend(path string, offset int64) (*os.File, error) {
	if offset <= 0 {
		return os.Create(path)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	if st, err := f.Stat(); err != nil || st.Size() != offset {
		_ = f.Close()
		return os.Create(path)
	}
	return f, nil
}

func remoteMetaPath(tmpDir string) string {
	return filepath.Join(tmpDir, "meta.json")
}
