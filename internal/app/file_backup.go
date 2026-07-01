package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dback/backend/builder"
	"dback/backend/shell"
	"dback/backend/verify"
	"dback/internal/capability"
	"dback/internal/connector"
	"dback/internal/paths"
	"dback/models"
)

// FileBackupProgress reports multi-path file backup status.
type FileBackupProgress struct {
	PathIndex  int
	PathTotal  int
	PathName   string
	BytesDone  int64
	BytesTotal int64
	SpeedBps   int64
	Phase      string
}

// FileBackupProgressFunc receives structured progress updates.
type FileBackupProgressFunc func(p FileBackupProgress)

// FileBackupResult summarizes one file backup job.
type FileBackupResult struct {
	OperationID string
	Records     []models.ExportRecord
	PartialFail bool
}

// BackupFiles archives configured paths sequentially for a host.
func (a *App) BackupFiles(ctx context.Context, profile models.Profile, progress FileBackupProgressFunc) (FileBackupResult, error) {
	if !profile.SupportsFileBackup() {
		return FileBackupResult{}, fmt.Errorf("file backup is not supported for this connection type")
	}
	if !profile.FileBackupEnabled {
		return FileBackupResult{}, fmt.Errorf("file backup is disabled for this host")
	}
	if len(profile.FileBackupPaths) == 0 {
		return FileBackupResult{}, fmt.Errorf("no file backup paths configured")
	}
	if err := models.ValidateFileBackupPaths(profile.FileBackupPaths); err != nil {
		return FileBackupResult{}, err
	}

	operationID := newID()
	started := time.Now()
	dest := paths.EffectiveBackupDestination(profile.EffectiveFileBackupDestination(paths.DefaultBackupDestination()))
	if err := os.MkdirAll(dest, 0755); err != nil {
		return FileBackupResult{}, err
	}

	logger := a.newOpLogger(operationID, &profile)
	a.logPhase(operationID, &profile, "FileExport", "start", "", 0, "Starting file backup", "Info", "Started", "")

	conn, err := connector.NewConnector(profile)
	if err != nil {
		a.logPhase(operationID, &profile, "FileExport", "failure", "", 0, "Connector failed", "Error", "Failed", err.Error())
		return FileBackupResult{}, err
	}
	defer conn.Close()

	planner := capability.DefaultFilesystemProvider()
	mode := shell.ModeRemoteToLocalPipe
	if profile.IsLocalhost() {
		mode = shell.ModeLocalPipe
	}
	compression := models.NormalizeArchiveCompression(profile.FileBackupCompression)
	ext, err := builder.ArchiveExtension(compression)
	if err != nil {
		return FileBackupResult{}, err
	}

	hostDir := filepath.Join(dest, safeName(profile.Name), "files")
	if err := os.MkdirAll(hostDir, 0755); err != nil {
		return FileBackupResult{}, err
	}

	var records []models.ExportRecord
	total := len(profile.FileBackupPaths)
	var lastErr error

	for i, pathCfg := range profile.FileBackupPaths {
		if err := ctx.Err(); err != nil {
			return FileBackupResult{OperationID: operationID, Records: records, PartialFail: len(records) > 0}, err
		}

		seq := i + 1
		reportProgress(progress, FileBackupProgress{
			PathIndex: seq,
			PathTotal: total,
			PathName:  pathCfg.Name,
			Phase:     "archiving",
		})

		plan, err := planner.PlanArchive(capability.ArchiveOptions{
			RootPath:    pathCfg.RemotePath,
			Excludes:    profile.FileBackupExclude,
			Compression: compression,
			Mode:        mode,
		})
		if err != nil {
			lastErr = err
			break
		}

		label := models.SafeLabel(pathCfg.CanonicalKey)
		ts := time.Now().Format("02_01_2006_15_04_05")
		finalPath := filepath.Join(hostDir, fmt.Sprintf("%s_%s%s", label, ts, ext))
		partialPath := finalPath + ".partial"

		size, copyErr := a.copyArchive(ctx, conn, plan, partialPath, func(bytesDone int64, speed int64) {
			reportProgress(progress, FileBackupProgress{
				PathIndex: seq,
				PathTotal: total,
				PathName:  pathCfg.Name,
				BytesDone: bytesDone,
				SpeedBps:  speed,
				Phase:     "transferring",
			})
		})
		if copyErr != nil {
			_ = os.Remove(partialPath)
			lastErr = copyErr
			a.logPhase(operationID, &profile, "FileExport", "failure", "", seq, fmt.Sprintf("Path %q failed: %v", pathCfg.Name, copyErr), "Error", "Failed", copyErr.Error())
			break
		}
		if err := os.Rename(partialPath, finalPath); err != nil {
			_ = os.Remove(partialPath)
			lastErr = err
			break
		}
		if size < 128 {
			_ = os.Remove(finalPath)
			lastErr = fmt.Errorf("backup file too small for %q (%d bytes)", pathCfg.Name, size)
			break
		}

		sha256, err := verify.ChecksumFile(finalPath)
		if err != nil {
			lastErr = err
			break
		}

		record := models.ExportRecord{
			ID:                 newID(),
			OperationID:        operationID,
			ProfileID:          profile.ID,
			ProfileName:        profile.Name,
			ExportType:         models.ExportTypeFiles,
			FileBackupPathID:   pathCfg.ID,
			SourceLabel:        pathCfg.Name,
			SourcePath:         pathCfg.RemotePath,
			CanonicalKey:       pathCfg.CanonicalKey,
			JobSequence:        seq,
			VerificationMethod: models.VerifySHA256,
			ExportDate:         time.Now(),
			FilePath:           finalPath,
			FileSize:           formatSize(size),
			FileSizeBytes:      size,
			ConnectionType:     profile.ConnectionType,
			Sha256:             sha256,
		}

		a.mu.Lock()
		a.history = append(a.history, record)
		history := append([]models.ExportRecord(nil), a.history...)
		a.mu.Unlock()
		if err := a.store.SaveHistory(history); err != nil {
			return FileBackupResult{OperationID: operationID, Records: append(records, record), PartialFail: true}, err
		}
		records = append(records, record)

		a.logPhaseWithFile(operationID, profile, "FileExport", "path_complete", "", seq, fmt.Sprintf("Archived %q", pathCfg.Name), "Info", "Succeeded", "", finalPath, size)
	}

	result := FileBackupResult{
		OperationID: operationID,
		Records:     records,
		PartialFail: lastErr != nil && len(records) > 0,
	}
	if lastErr != nil {
		if errors.Is(lastErr, context.Canceled) {
			a.logPhase(operationID, &profile, "FileExport", "cancel", "", 0, "File backup canceled", "Info", "Canceled", "")
			return result, lastErr
		}
		return result, lastErr
	}

	a.logPhaseWithFile(operationID, profile, "FileExport", "complete", "", 0, fmt.Sprintf("File backup completed in %s (%d paths)", time.Since(started).Round(time.Millisecond), len(records)), "Info", "Succeeded", "", "", 0)
	_ = logger
	return result, nil
}

func (a *App) copyArchive(ctx context.Context, conn connector.Connector, plan shell.ExecutionPlan, destPath string, onBytes func(done int64, speedBps int64)) (int64, error) {
	result, err := conn.Run(ctx, plan)
	if err != nil {
		return 0, err
	}
	defer result.Reader.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}

	var written int64
	lastTick := time.Now()
	lastBytes := int64(0)
	pr := &transferProgressReader{
		Reader: result.Reader,
		onRead: func(n int64) {
			written += n
			now := time.Now()
			if onBytes != nil && now.Sub(lastTick) >= 200*time.Millisecond {
				elapsed := now.Sub(lastTick).Seconds()
				if elapsed > 0 {
					speed := int64(float64(written-lastBytes) / elapsed)
					onBytes(written, speed)
				}
				lastTick = now
				lastBytes = written
			}
		},
	}
	_, copyErr := io.Copy(out, pr)
	closeErr := out.Close()
	waitErr := result.Wait()

	if copyErr != nil {
		return written, copyErr
	}
	if closeErr != nil {
		return written, closeErr
	}
	if waitErr != nil {
		return written, waitErr
	}
	if err := ctx.Err(); err != nil {
		return written, err
	}
	if onBytes != nil {
		onBytes(written, 0)
	}
	return written, nil
}

type transferProgressReader struct {
	io.Reader
	onRead func(n int64)
}

func (r *transferProgressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 && r.onRead != nil {
		r.onRead(int64(n))
	}
	return n, err
}

func reportProgress(fn FileBackupProgressFunc, p FileBackupProgress) {
	if fn != nil {
		fn(p)
	}
}

func normalizeProfileFileBackup(p *models.Profile) error {
	p.FileBackupCompression = models.NormalizeArchiveCompression(p.FileBackupCompression)
	for i, ex := range p.FileBackupExclude {
		p.FileBackupExclude[i] = strings.TrimSpace(ex)
		if err := models.ValidateExcludePattern(p.FileBackupExclude[i]); err != nil {
			return err
		}
	}
	if len(p.FileBackupPaths) == 0 {
		return nil
	}
	return models.ValidateFileBackupPaths(p.FileBackupPaths)
}
