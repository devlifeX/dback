package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dback/backend/db"
	"dback/backend/preflight"
	"dback/backend/ssh"
	"dback/models"

	cryptossh "golang.org/x/crypto/ssh"
)

type Strategy string

const (
	StrategyStreaming Strategy = "streaming"
	StrategyTmpFile   Strategy = "tmp-file"
)

type Logger interface {
	Phase(action, phase, strategy string, attempt int, details, status, errStr string)
}

type ProgressFunc func(message string, current int64, total int64)

type BackupRequest struct {
	Profile     models.Profile
	OperationID string
	Destination string
	Logger      Logger
	Progress    ProgressFunc
}

type BackupResult struct {
	Path string
	Size int64
}

// BackupSSH performs backup with streaming first, tmp-file fallback on retryable errors.
func BackupSSH(ctx context.Context, req BackupRequest) (BackupResult, error) {
	p := req.Profile
	if err := ctx.Err(); err != nil {
		return BackupResult{}, err
	}
	client, err := ssh.NewClient(p)
	if err != nil {
		return BackupResult{}, err
	}
	defer client.Close()

	pf, pfErr := preflight.Run(client, p, 0, req.OperationID)
	if pfErr != nil {
		logReq(req, "preflight", "", 0, preflight.Summary(pf), "Failed", pfErr.Error())
		return BackupResult{}, pfErr
	}
	logReq(req, "preflight", "", 0, preflight.Summary(pf), "Succeeded", "")

	fileName := fmt.Sprintf("%s_%s_%s.sql.gz", safeName(p.Name), safeName(p.TargetDBName), time.Now().Format("02_01_2006_15_04_05"))
	fullPath := filepath.Join(req.Destination, fileName)

	exportCmd := db.BuildExportCommand(p)
	logReq(req, "command", string(StrategyStreaming), 0, db.MaskCommand(exportCmd), "Built", "")

	strategies := []Strategy{StrategyStreaming, StrategyTmpFile}
	var lastErr error
	for attempt, strategy := range strategies {
		if err := ctx.Err(); err != nil {
			return BackupResult{}, err
		}
		logReq(req, "backup", string(strategy), attempt+1, "Starting backup attempt", "Started", "")
		var size int64
		var err error
		switch strategy {
		case StrategyStreaming:
			size, err = backupStream(ctx, client, p, fullPath, req.Progress)
		case StrategyTmpFile:
			size, err = backupTmpFile(ctx, client, p, pf.SelectedTmpDir, fullPath, req.OperationID, req.Progress)
		}
		if err == nil {
			if validateErr := validateLocalFile(fullPath, size, ""); validateErr != nil {
				err = validateErr
			} else if sum, sumErr := checksumFile(fullPath); sumErr == nil {
				logReq(req, "checksum", string(strategy), attempt+1, "sha256="+sum, "Succeeded", "")
			}
		}
		if err == nil {
			removeMeta(fullPath)
			logReq(req, "backup", string(strategy), attempt+1, "Backup completed", "Succeeded", "")
			return BackupResult{Path: fullPath, Size: size}, nil
		}
		lastErr = err
		logReq(req, "backup", string(strategy), attempt+1, err.Error(), "Failed", err.Error())
		if strategy == StrategyTmpFile {
			logReq(req, "cleanup", string(strategy), attempt+1, "remote tmp kept at "+pf.SelectedTmpDir, "Warning", err.Error())
		}
		if !isRetryable(err) || attempt == len(strategies)-1 {
			break
		}
		logReq(req, "backup", string(strategy), attempt+1, "Retrying with tmp-file strategy", "Retry", "")
		_ = os.Remove(fullPath)
		removeMeta(fullPath)
	}
	return BackupResult{}, lastErr
}

func backupStream(ctx context.Context, client *ssh.Client, p models.Profile, fullPath string, progress ProgressFunc) (int64, error) {
	cmd := db.BuildExportCommand(p)
	stdout, stderr, session, err := client.RunCommandStream(cmd)
	if err != nil {
		return 0, err
	}
	defer session.Close()
	go cancelOnContext(ctx, session, client)

	var stderrBuf strings.Builder
	go func() { _, _ = io.Copy(&stderrBuf, stderr) }()

	out, err := os.Create(fullPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	written, err := io.Copy(out, &ssh.ProgressReader{
		Reader: stdout,
		Callback: func(current int64, total int64) {
			if progress != nil {
				progress(fmt.Sprintf("Streaming backup %.2f MB", float64(current)/1024/1024), current, total)
			}
		},
	})
	if ctx.Err() != nil {
		_ = os.Remove(fullPath)
		return 0, ctx.Err()
	}
	if err != nil {
		_ = os.Remove(fullPath)
		return 0, err
	}
	if err := session.Wait(); err != nil {
		_ = os.Remove(fullPath)
		return 0, fmt.Errorf("%w: %s", err, stderrBuf.String())
	}
	if written < 128 {
		_ = os.Remove(fullPath)
		return 0, fmt.Errorf("backup file too small (%d bytes)", written)
	}
	return written, nil
}

func backupTmpFile(ctx context.Context, client *ssh.Client, p models.Profile, tmpDir, localPath, operationID string, progress ProgressFunc) (int64, error) {
	remotePath := tmpDir + "/dump.sql.gz"
	mkdir := shellMkdir(tmpDir)
	if _, err := client.RunCommand(mkdir); err != nil {
		return 0, fmt.Errorf("create tmp dir: %w", err)
	}

	if meta, ok := loadMeta(localPath); ok && meta.RemotePath == remotePath {
		if progress != nil {
			progress("Resuming tmp-file download", meta.Offset, meta.Size)
		}
	} else {
		exportCmd := db.BuildExportToFileCommand(p, remotePath)
		if out, err := client.RunCommand(exportCmd); err != nil {
			return 0, fmt.Errorf("remote dump: %w: %s", err, strings.TrimSpace(out))
		}
	}

	sizeOut, err := client.RunCommand(db.BuildFileSizeCommand(remotePath))
	if err != nil {
		return 0, fmt.Errorf("remote file size: %w", err)
	}
	var remoteSize int64
	fmt.Sscanf(strings.TrimSpace(sizeOut), "%d", &remoteSize)
	if remoteSize < 128 {
		return 0, fmt.Errorf("remote dump too small (%d bytes)", remoteSize)
	}

	checksumOut, _ := client.RunCommand(db.BuildChecksumCommand(remotePath))
	remoteChecksum := strings.TrimSpace(checksumOut)

	meta := FileMeta{
		OperationID: operationID,
		RemotePath:  remotePath,
		LocalPath:   localPath,
		Size:        remoteSize,
		Checksum:    remoteChecksum,
	}
	offset := localResumeOffset(localPath, meta, remoteSize)
	meta.Offset = offset
	_ = saveMeta(meta)

	downloadCmd := db.BuildDownloadChunkCommand(remotePath, offset)
	stdout, stderr, session, err := client.RunCommandStream(downloadCmd)
	if err != nil {
		return 0, err
	}
	defer session.Close()
	go cancelOnContext(ctx, session, client)

	var stderrBuf strings.Builder
	go func() { _, _ = io.Copy(&stderrBuf, stderr) }()

	out, err := openLocalAppend(localPath, offset)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	written, err := io.Copy(out, &ssh.ProgressReader{
		Reader: stdout,
		Total:  remoteSize - offset,
		Callback: func(current int64, total int64) {
			meta.Offset = offset + current
			_ = saveMeta(meta)
			if progress != nil {
				progress(fmt.Sprintf("Downloading tmp file %.1f%%", percent(offset+current, remoteSize)), offset+current, remoteSize)
			}
		},
	})
	if ctx.Err() != nil {
		return offset + written, ctx.Err()
	}
	if err != nil {
		return offset + written, err
	}
	if err := session.Wait(); err != nil {
		return offset + written, fmt.Errorf("%w: %s", err, stderrBuf.String())
	}

	finalSize := offset + written
	if remoteSize > 0 && finalSize != remoteSize {
		return finalSize, fmt.Errorf("incomplete download: got %d want %d", finalSize, remoteSize)
	}
	if remoteChecksum != "" {
		if err := validateLocalFile(localPath, remoteSize, remoteChecksum); err != nil {
			return finalSize, err
		}
	}
	_, _ = client.RunCommand(shellCleanup(tmpDir))
	logReq(BackupRequest{OperationID: operationID}, "cleanup", string(StrategyTmpFile), 0, tmpDir, "Succeeded", "")
	return finalSize, nil
}

type RestoreRequest struct {
	Profile     models.Profile
	OperationID string
	LocalPath   string
	FileSize    int64
	Logger      Logger
	Progress    ProgressFunc
}

// RestoreSSH restores with streaming first, tmp-file fallback.
func RestoreSSH(ctx context.Context, req RestoreRequest) error {
	p := req.Profile
	in, err := os.Open(req.LocalPath)
	if err != nil {
		return err
	}
	defer in.Close()

	if req.FileSize <= 0 {
		if info, statErr := os.Stat(req.LocalPath); statErr == nil {
			req.FileSize = info.Size()
		}
	}
	if sum, sumErr := checksumFile(req.LocalPath); sumErr == nil {
		logRestore(req, "checksum", "", 0, "local sha256="+sum, "Info", "")
	}

	client, err := ssh.NewClient(p)
	if err != nil {
		return err
	}
	defer client.Close()

	pf, pfErr := preflight.Run(client, p, req.FileSize, req.OperationID)
	if pfErr != nil {
		logRestore(req, "preflight", "", 0, preflight.Summary(pf), "Failed", pfErr.Error())
		return pfErr
	}
	logRestore(req, "preflight", "", 0, preflight.Summary(pf), "Succeeded", "")

	compression, err := detectCompression(in)
	if err != nil {
		return err
	}

	importCmd := db.BuildImportStreamCommand(p, compression)
	logRestore(req, "command", string(StrategyStreaming), 0, db.MaskCommand(importCmd), "Built", "")

	strategies := []Strategy{StrategyStreaming, StrategyTmpFile}
	var lastErr error
	for attempt, strategy := range strategies {
		if err := ctx.Err(); err != nil {
			return err
		}
		logRestore(req, "restore", string(strategy), attempt+1, "Starting restore attempt", "Started", "")

		if prep := db.BuildImportPrepareCommand(p); prep != "" {
			if out, prepErr := client.RunCommand(prep); prepErr != nil {
				lastErr = fmt.Errorf("prepare database: %w: %s", prepErr, strings.TrimSpace(out))
				logRestore(req, "prepare", string(strategy), attempt+1, lastErr.Error(), "Failed", lastErr.Error())
				if !isRetryable(lastErr) {
					return lastErr
				}
				continue
			}
			logRestore(req, "prepare", string(strategy), attempt+1, "DROP/CREATE completed", "Succeeded", "")
		}

		var restoreErr error
		switch strategy {
		case StrategyStreaming:
			if _, seekErr := in.Seek(0, io.SeekStart); seekErr != nil {
				return seekErr
			}
			restoreErr = restoreStream(ctx, client, p, in, req.FileSize, compression, req.Progress)
		case StrategyTmpFile:
			if _, seekErr := in.Seek(0, io.SeekStart); seekErr != nil {
				return seekErr
			}
			restoreErr = restoreTmpFile(ctx, client, p, pf.SelectedTmpDir, req.LocalPath, in, req.FileSize, compression, req.OperationID, req.Progress)
		}
		if restoreErr == nil {
			logRestore(req, "restore", string(strategy), attempt+1, "Restore completed", "Succeeded", "")
			return nil
		}
		lastErr = restoreErr
		logRestore(req, "restore", string(strategy), attempt+1, restoreErr.Error(), "Failed", restoreErr.Error())
		if strategy == StrategyTmpFile {
			logRestore(req, "cleanup", string(strategy), attempt+1, "remote tmp kept at "+pf.SelectedTmpDir, "Warning", restoreErr.Error())
		}
		if !isRetryable(restoreErr) || attempt == len(strategies)-1 {
			break
		}
		logRestore(req, "restore", string(strategy), attempt+1, "Retrying with tmp-file strategy", "Retry", "")
	}
	return lastErr
}

func restoreStream(ctx context.Context, client *ssh.Client, p models.Profile, in *os.File, total int64, compression string, progress ProgressFunc) error {
	importCmd := db.BuildImportStreamCommand(p, compression)
	stdin, stderr, session, err := client.RunCommandPipeInput(importCmd)
	if err != nil {
		return err
	}
	defer session.Close()
	go cancelOnContext(ctx, session, client)

	var stderrBuf strings.Builder
	go func() { _, _ = io.Copy(&stderrBuf, stderr) }()

	_, err = io.Copy(stdin, &ssh.ProgressReader{
		Reader: in,
		Total:  total,
		Callback: func(current int64, total int64) {
			if progress != nil {
				progress(fmt.Sprintf("Streaming restore %.1f%%", percent(current, total)), current, total)
			}
		},
	})
	if ctx.Err() != nil {
		_ = stdin.Close()
		return ctx.Err()
	}
	if closeErr := stdin.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	if err := session.Wait(); err != nil {
		return fmt.Errorf("%w: %s", err, stderrBuf.String())
	}
	return nil
}

func restoreTmpFile(ctx context.Context, client *ssh.Client, p models.Profile, tmpDir, localPath string, in *os.File, total int64, compression, operationID string, progress ProgressFunc) error {
	remotePath := tmpDir + "/import.sql.gz"
	mkdir := shellMkdir(tmpDir)
	if _, err := client.RunCommand(mkdir); err != nil {
		return err
	}

	meta, hasMeta := loadMeta(localPath)
	offset := int64(0)
	if hasMeta && meta.RemotePath == remotePath {
		offset = meta.Offset
	} else {
		sizeOut, _ := client.RunCommand(db.BuildFileSizeCommand(remotePath))
		fmt.Sscanf(strings.TrimSpace(sizeOut), "%d", &offset)
	}

	uploadCmd := db.BuildUploadCommand(remotePath, offset > 0)
	stdin, stderr, session, err := client.RunCommandPipeInput(uploadCmd)
	if err != nil {
		return err
	}

	if offset > 0 {
		if _, err := in.Seek(offset, io.SeekStart); err != nil {
			_ = session.Close()
			return err
		}
	}

	_, copyErr := io.Copy(stdin, &ssh.ProgressReader{
		Reader: in,
		Total:  total - offset,
		Callback: func(current int64, total int64) {
			meta = FileMeta{
				OperationID: operationID,
				RemotePath:  remotePath,
				LocalPath:   localPath,
				Size:        total,
				Offset:      offset + current,
				Compression: compression,
			}
			_ = saveMeta(meta)
			if progress != nil {
				progress(fmt.Sprintf("Uploading tmp file %.1f%%", percent(offset+current, total)), offset+current, total)
			}
		},
	})
	_ = stdin.Close()
	if copyErr != nil {
		_ = session.Close()
		return copyErr
	}
	var stderrBuf strings.Builder
	go func() { _, _ = io.Copy(&stderrBuf, stderr) }()
	if err := session.Wait(); err != nil {
		return fmt.Errorf("upload tmp: %w: %s", err, stderrBuf.String())
	}

	remoteSizeOut, _ := client.RunCommand(db.BuildFileSizeCommand(remotePath))
	var remoteSize int64
	fmt.Sscanf(strings.TrimSpace(remoteSizeOut), "%d", &remoteSize)
	if total > 0 && remoteSize != total {
		return fmt.Errorf("upload size mismatch: remote %d local %d", remoteSize, total)
	}

	importCmd := db.BuildImportFromFileCommand(p, remotePath, compression)
	if out, err := client.RunCommand(importCmd); err != nil {
		return fmt.Errorf("import from tmp: %w: %s", err, strings.TrimSpace(out))
	}
	_, _ = client.RunCommand(shellCleanup(tmpDir))
	removeMeta(localPath)
	return nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"eof", "broken pipe", "connection reset", "timeout", "temporarily unavailable",
		"connection closed", "i/o timeout", "unexpectedly", "incomplete download", "upload size mismatch",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func detectCompression(file *os.File) (string, error) {
	var magic [4]byte
	n, err := io.ReadFull(file, magic[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return "", seekErr
	}
	if n >= 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		return "gzip", nil
	}
	if n >= 4 && magic == [4]byte{0x28, 0xb5, 0x2f, 0xfd} {
		return "zstd", nil
	}
	return "", nil
}

func cancelOnContext(ctx context.Context, session *cryptossh.Session, client *ssh.Client) {
	<-ctx.Done()
	_ = session.Close()
	_ = client.Close()
}

func logReq(req BackupRequest, phase, strategy string, attempt int, details, status, errStr string) {
	if req.Logger != nil {
		req.Logger.Phase("Export", phase, strategy, attempt, details, status, errStr)
	}
}

func logRestore(req RestoreRequest, phase, strategy string, attempt int, details, status, errStr string) {
	if req.Logger != nil {
		req.Logger.Phase("Import", phase, strategy, attempt, details, status, errStr)
	}
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "backup"
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	return replacer.Replace(s)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func shellMkdir(dir string) string {
	if strings.Contains(dir, "$HOME") {
		return fmt.Sprintf("bash -lc %s", shellQuote("mkdir -p "+dir))
	}
	return fmt.Sprintf("mkdir -p %s", shellQuote(dir))
}

func shellCleanup(dir string) string {
	if strings.Contains(dir, "$HOME") {
		return fmt.Sprintf("bash -lc %s", shellQuote("rm -rf "+dir))
	}
	return db.BuildCleanupCommand(dir)
}

func percent(current, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(current) / float64(total) * 100
}
