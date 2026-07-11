package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dback/internal/paths"
	"dback/internal/remote"
	"dback/models"
)

const remoteBackupPrefix = remote.BackupPrefix

// StorageRoot is a browsable local backup directory.
type StorageRoot struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// StorageEntry is one row in a storage listing.
type StorageEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir"`
	Modified string `json:"modified,omitempty"`
}

// StorageListing is the response for browse endpoints.
type StorageListing struct {
	Path    string         `json:"path"`
	Parent  string         `json:"parent,omitempty"`
	Roots   []StorageRoot  `json:"roots,omitempty"`
	Entries []StorageEntry `json:"entries"`
}

// StorageSummary holds aggregate storage stats.
type StorageSummary struct {
	Bytes int64 `json:"bytes"`
	Count int   `json:"count"`
}

// LocalStorageSummary aggregates on-disk backup storage.
type LocalStorageSummary struct {
	Bytes int64 `json:"bytes"`
	Files int   `json:"files"`
	Roots int   `json:"roots"`
}

// RemoteStorageSummary aggregates remote backup objects from history.
type RemoteStorageSummary struct {
	Bytes        int64 `json:"bytes"`
	Objects      int   `json:"objects"`
	Destinations int   `json:"destinations"`
}

func (a *App) LocalStorageRoots() []StorageRoot {
	byPath := map[string][]string{}
	for _, profile := range a.Profiles() {
		dest := paths.EffectiveBackupDestination(profile.Destination)
		if dest != "" {
			byPath[dest] = append(byPath[dest], profile.Name+" (DB)")
		}
		if profile.FileBackupEnabled {
			fileDest := paths.EffectiveBackupDestination(profile.FileBackupDestination)
			if fileDest != "" {
				byPath[fileDest] = append(byPath[fileDest], profile.Name+" (Files)")
			}
		}
	}
	roots := make([]StorageRoot, 0, len(byPath))
	for path, labels := range byPath {
		roots = append(roots, StorageRoot{
			Path:  path,
			Label: strings.Join(uniqueStrings(labels), ", "),
		})
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Path < roots[j].Path })
	return roots
}

func (a *App) ListLocalStorage(path string) (StorageListing, error) {
	roots := a.LocalStorageRoots()
	path = strings.TrimSpace(path)
	if path == "" {
		return StorageListing{Roots: roots, Entries: rootsToEntries(roots)}, nil
	}
	if err := a.validateLocalPath(path); err != nil {
		return StorageListing{}, err
	}
	entries, err := listLocalDir(path)
	if err != nil {
		return StorageListing{}, err
	}
	parent := localParentPath(path, roots)
	return StorageListing{Path: path, Parent: parent, Entries: entries}, nil
}

func (a *App) OpenLocalStorageFile(path string) (*os.File, StorageEntry, error) {
	if err := a.validateLocalPath(path); err != nil {
		return nil, StorageEntry{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, StorageEntry{}, fmt.Errorf("file not found")
	}
	if info.IsDir() {
		return nil, StorageEntry{}, fmt.Errorf("path is a directory")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, StorageEntry{}, err
	}
	entry := StorageEntry{
		Name:     info.Name(),
		Path:     path,
		Size:     info.Size(),
		IsDir:    false,
		Modified: info.ModTime().UTC().Format(time.RFC3339),
	}
	return f, entry, nil
}

func (a *App) LocalStorageSummary() LocalStorageSummary {
	roots := a.LocalStorageRoots()
	var totalBytes int64
	var fileCount int
	seen := map[string]struct{}{}
	for _, root := range roots {
		_ = filepath.WalkDir(root.Path, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if _, ok := seen[p]; ok {
				return nil
			}
			seen[p] = struct{}{}
			info, statErr := d.Info()
			if statErr != nil {
				return nil
			}
			fileCount++
			totalBytes += info.Size()
			return nil
		})
	}
	return LocalStorageSummary{Bytes: totalBytes, Files: fileCount, Roots: len(roots)}
}

func (a *App) ListRemoteStorage(ctx context.Context, destinationID, prefix string) (StorageListing, error) {
	dest, err := a.store.RemoteDestinationByID(destinationID)
	if err != nil {
		return StorageListing{}, err
	}
	provider, err := remote.NewProvider(dest)
	if err != nil {
		return StorageListing{}, err
	}
	prefix = normalizeRemotePrefix(prefix)
	listCtx, cancel := context.WithTimeout(ctx, remote.ListObjectsTimeout)
	defer cancel()
	objects, err := provider.ListObjects(listCtx, prefix)
	if err != nil {
		return StorageListing{}, err
	}
	entries := make([]StorageEntry, 0, len(objects))
	for _, obj := range objects {
		entries = append(entries, StorageEntry{
			Name:     obj.Name,
			Path:     obj.Key,
			Size:     obj.Size,
			IsDir:    obj.IsDir,
			Modified: formatTime(obj.LastModified),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	parent := remoteParentPrefix(prefix)
	return StorageListing{
		Path:    prefix,
		Parent:  parent,
		Entries: entries,
	}, nil
}

func (a *App) OpenRemoteStorageObject(ctx context.Context, destinationID, key string) (io.ReadCloser, StorageEntry, error) {
	dest, err := a.store.RemoteDestinationByID(destinationID)
	if err != nil {
		return nil, StorageEntry{}, err
	}
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" || !strings.HasPrefix(key, remoteBackupPrefix) {
		return nil, StorageEntry{}, fmt.Errorf("invalid remote key")
	}
	provider, err := remote.NewProvider(dest)
	if err != nil {
		return nil, StorageEntry{}, err
	}
	getCtx, cancel := context.WithTimeout(ctx, remote.GetObjectTimeout)
	defer cancel()
	reader, meta, err := provider.GetObject(getCtx, key)
	if err != nil {
		return nil, StorageEntry{}, err
	}
	entry := StorageEntry{
		Name:     filepath.Base(key),
		Path:     key,
		Size:     meta.Size,
		IsDir:    false,
		Modified: formatTime(meta.LastModified),
	}
	return reader, entry, nil
}

func (a *App) RemoteStorageSummary() RemoteStorageSummary {
	destinations, _ := a.store.LoadRemoteDestinations()
	destSet := map[string]struct{}{}
	var bytes int64
	var count int
	for _, rec := range a.History() {
		size := rec.FileSizeBytes
		for _, upload := range rec.RemoteUploads {
			if upload.Status != models.RemoteUploadDone {
				continue
			}
			if upload.SizeBytes > 0 {
				bytes += upload.SizeBytes
			} else if size > 0 {
				bytes += size
			}
			count++
			destSet[upload.DestinationID] = struct{}{}
		}
	}
	return RemoteStorageSummary{
		Bytes:        bytes,
		Objects:      count,
		Destinations: max(len(destSet), len(destinations)),
	}
}

func (a *App) validateLocalPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path")
	}
	abs = filepath.Clean(abs)
	for _, root := range a.LocalStorageRoots() {
		rootAbs, rootErr := filepath.Abs(root.Path)
		if rootErr != nil {
			continue
		}
		rootAbs = filepath.Clean(rootAbs)
		if pathWithinRoot(abs, rootAbs) {
			return nil
		}
	}
	return fmt.Errorf("path not allowed")
}

func pathWithinRoot(path, root string) bool {
	if path == root {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(path, root+sep)
}

func listLocalDir(path string) ([]StorageEntry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	entries := make([]StorageEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		info, infoErr := de.Info()
		if infoErr != nil {
			continue
		}
		entry := StorageEntry{
			Name:  name,
			Path:  full,
			Size:  info.Size(),
			IsDir: info.IsDir(),
		}
		if !info.IsDir() {
			entry.Modified = info.ModTime().UTC().Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func rootsToEntries(roots []StorageRoot) []StorageEntry {
	entries := make([]StorageEntry, 0, len(roots))
	for _, root := range roots {
		name := root.Label
		if name == "" {
			name = filepath.Base(root.Path)
		}
		entries = append(entries, StorageEntry{
			Name:  name,
			Path:  root.Path,
			IsDir: true,
		})
	}
	return entries
}

func localParentPath(path string, roots []StorageRoot) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	abs = filepath.Clean(abs)
	for _, root := range roots {
		rootAbs, rootErr := filepath.Abs(root.Path)
		if rootErr != nil {
			continue
		}
		rootAbs = filepath.Clean(rootAbs)
		if abs == rootAbs {
			return ""
		}
		if pathWithinRoot(abs, rootAbs) {
			parent := filepath.Dir(abs)
			if pathWithinRoot(parent, rootAbs) || parent == rootAbs {
				return parent
			}
		}
	}
	return ""
}

func normalizeRemotePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return remoteBackupPrefix
	}
	prefix = strings.TrimPrefix(prefix, "/")
	base := strings.TrimSuffix(remoteBackupPrefix, "/")
	if prefix == base {
		return remoteBackupPrefix
	}
	if !strings.HasPrefix(prefix, base+"/") && prefix != base {
		prefix = remoteBackupPrefix + strings.TrimPrefix(prefix, remoteBackupPrefix)
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

func remoteParentPrefix(prefix string) string {
	prefix = normalizeRemotePrefix(prefix)
	if prefix == remoteBackupPrefix {
		return ""
	}
	trimmed := strings.TrimSuffix(prefix, "/")
	if trimmed == "" || trimmed == remoteBackupPrefix[:len(remoteBackupPrefix)-1] {
		return remoteBackupPrefix
	}
	parent := filepath.ToSlash(filepath.Dir(trimmed))
	if parent == "." {
		return remoteBackupPrefix
	}
	if !strings.HasSuffix(parent, "/") {
		parent += "/"
	}
	if !strings.HasPrefix(parent, remoteBackupPrefix) {
		return remoteBackupPrefix
	}
	return parent
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
