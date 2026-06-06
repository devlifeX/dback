package transfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"dback/backend/db"
	"dback/backend/wordpress"
)

// BackupWordPress downloads a gzip SQL dump via the WordPress REST plugin.
func BackupWordPress(ctx context.Context, req BackupRequest) (BackupResult, error) {
	p := req.Profile
	if err := ctx.Err(); err != nil {
		return BackupResult{}, err
	}
	if err := db.ValidateProfileForWordPress(p); err != nil {
		return BackupResult{}, err
	}

	client, err := wordpress.NewClient(p)
	if err != nil {
		return BackupResult{}, err
	}

	pf, pfErr := client.Preflight(ctx)
	if pfErr != nil {
		logReq(req, "preflight", "", 0, pfErr.Error(), "Failed", pfErr.Error())
		return BackupResult{}, pfErr
	}
	if err := pf.FailureError(); err != nil {
		logReq(req, "preflight", "", 0, pf.Summary, "Failed", err.Error())
		return BackupResult{}, err
	}
	logReq(req, "preflight", "", 0, pf.Summary, "Succeeded", "")

	hostDir := filepath.Join(req.Destination, safeName(p.Name))
	if err := os.MkdirAll(hostDir, 0755); err != nil {
		return BackupResult{}, fmt.Errorf("create host backup folder: %w", err)
	}
	dbLabel := safeName(p.TargetDBName)
	if dbLabel == "" || dbLabel == "backup" {
		dbLabel = "wordpress"
	}
	fileName := fmt.Sprintf("%s_%s.sql.gz", dbLabel, time.Now().Format("02_01_2006_15_04_05"))
	fullPath := filepath.Join(hostDir, fileName)

	if req.Progress != nil {
		req.Progress("Starting WordPress export...", 0, 0)
	}

	body, err := client.Export(ctx)
	if err != nil {
		return BackupResult{}, err
	}
	defer body.Close()

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return BackupResult{}, err
	}
	out, err := os.Create(fullPath)
	if err != nil {
		return BackupResult{}, err
	}
	defer out.Close()

	written, err := fastCopy(out, &progressReader{
		reader: body,
		callback: func(current int64) {
			if req.Progress != nil {
				req.Progress(fmt.Sprintf("Streaming backup %.2f MB", float64(current)/1024/1024), current, 0)
			}
		},
	})
	if ctx.Err() != nil {
		_ = os.Remove(fullPath)
		return BackupResult{}, ctx.Err()
	}
	if err != nil {
		_ = os.Remove(fullPath)
		return BackupResult{}, err
	}
	if validateErr := validateBackupIntegrity(fullPath); validateErr != nil {
		_ = os.Remove(fullPath)
		return BackupResult{}, validateErr
	}
	if written < 128 {
		_ = os.Remove(fullPath)
		return BackupResult{}, fmt.Errorf("backup file too small (%d bytes)", written)
	}
	if sum, sumErr := checksumFile(fullPath); sumErr == nil {
		logReq(req, "checksum", string(StrategyStreaming), 0, "sha256="+sum, "Succeeded", "")
	}
	logReq(req, "backup", string(StrategyStreaming), 1, "Backup completed", "Succeeded", "")
	return BackupResult{Path: fullPath, Size: written}, nil
}

// RestoreWordPress uploads a gzip SQL dump to the WordPress REST plugin.
func RestoreWordPress(ctx context.Context, req RestoreRequest) error {
	p := restoreProfile(req)
	if err := db.ValidateProfileForWordPress(p); err != nil {
		return err
	}
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

	client, err := wordpress.NewClient(p)
	if err != nil {
		return err
	}

	pf, pfErr := client.Preflight(ctx)
	if pfErr != nil {
		logRestore(req, "preflight", "", 0, pfErr.Error(), "Failed", pfErr.Error())
		return pfErr
	}
	if err := pf.FailureError(); err != nil {
		logRestore(req, "preflight", "", 0, pf.Summary, "Failed", err.Error())
		return err
	}
	logRestore(req, "preflight", "", 0, pf.Summary, "Succeeded", "")

	if req.Progress != nil {
		req.Progress("Uploading backup to WordPress...", 0, req.FileSize)
	}

	uploadReader := io.Reader(in)
	if req.FileSize > 0 {
		uploadReader = &progressReader{
			reader: in,
			callback: func(current int64) {
				if req.Progress != nil {
					req.Progress(fmt.Sprintf("Uploading restore %.1f%%", percent(current, req.FileSize)), current, req.FileSize)
				}
			},
		}
	}

	if err := client.Import(ctx, uploadReader, req.FileSize, db.WordPressImportDatabase(p)); err != nil {
		logRestore(req, "restore", string(StrategyStreaming), 1, err.Error(), "Failed", err.Error())
		return err
	}
	logRestore(req, "restore", string(StrategyStreaming), 1, "Restore completed", "Succeeded", "")
	if req.Progress != nil {
		req.Progress("Restore completed", req.FileSize, req.FileSize)
	}
	return nil
}

type progressReader struct {
	reader   io.Reader
	callback func(int64)
	total    int64
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	if n > 0 {
		p.total += int64(n)
		if p.callback != nil {
			p.callback(p.total)
		}
	}
	return n, err
}
