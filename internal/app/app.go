package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dback/backend/db"
	"dback/backend/ssh"
	"dback/backend/wordpress"
	"dback/internal/debug"
	"dback/internal/store"
	"dback/models"
)

type ProgressFunc func(message string, current int64, total int64)

type App struct {
	store *store.Store

	mu       sync.Mutex
	profiles []models.Profile
	history  []models.ExportRecord
	logs     []models.LogEntry
}

func New(baseDir string) (*App, error) {
	a := &App{store: store.New(baseDir)}
	if err := a.Reload(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) Reload() error {
	profiles, err := a.store.LoadProfiles()
	if err != nil {
		return err
	}
	history, err := a.store.LoadHistory()
	if err != nil {
		return err
	}
	logs, err := a.store.LoadLogs()
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.profiles = profiles
	a.history = history
	a.logs = logs
	return nil
}

func (a *App) Profiles() []models.Profile {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.Profile(nil), a.profiles...)
}

func (a *App) History() []models.ExportRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.ExportRecord(nil), a.history...)
}

func (a *App) Logs() []models.LogEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.LogEntry(nil), a.logs...)
}

func (a *App) SaveProfile(profile models.Profile) error {
	if profile.ID == "" {
		profile.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if profile.Group == "" {
		profile.Group = "Default"
	}
	if profile.ExportSettings == nil {
		settings := models.SettingsFromProfile(profile)
		profile.ExportSettings = &settings
	}
	if profile.ImportSettings == nil {
		settings := models.SettingsFromProfile(profile)
		profile.ImportSettings = &settings
	}

	a.mu.Lock()
	found := false
	for i := range a.profiles {
		if a.profiles[i].ID == profile.ID {
			a.profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		a.profiles = append(a.profiles, profile)
	}
	profiles := append([]models.Profile(nil), a.profiles...)
	a.mu.Unlock()

	return a.store.SaveProfiles(profiles)
}

func (a *App) DeleteProfile(id string) error {
	a.mu.Lock()
	for i := range a.profiles {
		if a.profiles[i].ID == id {
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			break
		}
	}
	profiles := append([]models.Profile(nil), a.profiles...)
	a.mu.Unlock()
	return a.store.SaveProfiles(profiles)
}

func (a *App) ExportProfiles(path string, includeSecrets bool) error {
	return a.store.ExportProfiles(path, a.Profiles(), includeSecrets)
}

func (a *App) ImportProfiles(path string, includeSecrets bool) error {
	profiles, err := a.store.ImportProfiles(path, includeSecrets)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = profiles
	a.mu.Unlock()
	return a.store.SaveProfiles(profiles)
}

func (a *App) Backup(ctx context.Context, profile models.Profile, progress ProgressFunc) (models.ExportRecord, error) {
	operationID := newID()
	source := profile.EffectiveExport()
	if source.Destination == "" {
		source.Destination = "."
	}
	if err := os.MkdirAll(source.Destination, 0755); err != nil {
		return models.ExportRecord{}, err
	}

	a.log(operationID, &profile, "Export", "Starting backup", "", "", "Info", "Started", "")
	if progress != nil {
		progress("Starting backup", 0, 0)
	}

	var fullPath string
	var size int64
	var err error
	if source.ConnectionType == models.ConnectionTypeWordPress {
		fullPath, size, err = a.backupWordPress(ctx, source, progress)
	} else {
		fullPath, size, err = a.backupSSH(ctx, source, progress)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			a.log(operationID, &profile, "Export", "Backup canceled", "", "", "Info", "Canceled", "")
			return models.ExportRecord{}, err
		}
		a.log(operationID, &profile, "Export", "Backup failed", "", "", "Error", "Failed", err.Error())
		return models.ExportRecord{}, err
	}

	record := models.ExportRecord{
		ID:             newID(),
		OperationID:    operationID,
		ProfileID:      profile.ID,
		ProfileName:    profile.Name,
		DatabaseName:   source.TargetDBName,
		ExportDate:     time.Now(),
		FilePath:       fullPath,
		FileSize:       formatSize(size),
		FileSizeBytes:  size,
		ConnectionType: source.ConnectionType,
	}
	a.mu.Lock()
	a.history = append(a.history, record)
	history := append([]models.ExportRecord(nil), a.history...)
	a.mu.Unlock()
	if err := a.store.SaveHistory(history); err != nil {
		return record, err
	}
	a.log(operationID, &profile, "Export", "Backup completed", fullPath, formatSize(size), "Info", "Succeeded", "")
	if progress != nil {
		progress("Backup completed", size, size)
	}
	return record, nil
}

func (a *App) RunImportQuery(ctx context.Context, profile models.Profile, query string, connectDB bool) (db.QueryResult, error) {
	if err := ctx.Err(); err != nil {
		return db.QueryResult{}, err
	}
	if !profile.SupportsImportSQLQuery() {
		return db.QueryResult{}, fmt.Errorf("SQL query is only supported for SSH/Jump Host profiles with MySQL or MariaDB import settings")
	}
	p := profile.EffectiveImport()
	cmd, err := db.BuildQueryCommand(p, query, connectDB)
	if err != nil {
		return db.QueryResult{}, err
	}
	client, err := ssh.NewClient(p)
	if err != nil {
		debug.Errorf("query SSH connect failed: %v", err)
		return db.QueryResult{}, err
	}
	defer client.Close()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-done:
		}
	}()
	defer close(done)

	output, err := client.RunCommand(cmd)
	if ctx.Err() != nil {
		return db.QueryResult{}, ctx.Err()
	}
	output = strings.TrimSpace(output)
	if err != nil {
		return db.QueryResult{
			Columns: []string{"Error"},
			Rows:    [][]string{{output}},
			Message: err.Error(),
		}, fmt.Errorf("%w: %s", err, output)
	}
	result := db.ParseMySQLBatchOutput(output)
	result.Message = output
	return result, nil
}

func (a *App) Restore(ctx context.Context, record models.ExportRecord, destination models.Profile, progress ProgressFunc) error {
	operationID := newID()
	target := destination.EffectiveImport()
	if _, err := os.Stat(record.FilePath); err != nil {
		return err
	}
	a.log(operationID, &destination, "Import", "Starting restore from history", record.FilePath, record.FileSize, "Info", "Started", "")
	if progress != nil {
		progress("Starting restore", 0, record.FileSizeBytes)
	}

	if destination.ImportSettings != nil {
		is := destination.ImportSettings
		if is.RunQueryBeforeImport && strings.TrimSpace(is.PreImportQuery) != "" && destination.SupportsImportSQLQuery() {
			dbName := destination.EffectiveExport().TargetDBName
			query := models.SubstituteQueryDBName(is.PreImportQuery, dbName)
			if progress != nil {
				progress("Running pre-import query", 0, record.FileSizeBytes)
			}
			if result, err := a.RunImportQuery(ctx, destination, query, false); err != nil {
				a.log(operationID, &destination, "Pre-import query", err.Error(), "", "", "Warning", "Failed", err.Error())
			} else {
				details := formatQueryResultSummary(result)
				a.log(operationID, &destination, "Pre-import query", details, "", "", "Info", "Succeeded", "")
			}
		}
	}

	var err error
	if target.ConnectionType == models.ConnectionTypeWordPress {
		err = a.restoreWordPress(ctx, record.FilePath, target, progress)
	} else {
		err = a.restoreSSH(ctx, record.FilePath, target, progress)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			a.log(operationID, &destination, "Import", "Restore canceled", record.FilePath, record.FileSize, "Info", "Canceled", "")
			return err
		}
		a.log(operationID, &destination, "Import", "Restore failed", record.FilePath, record.FileSize, "Error", "Failed", err.Error())
		return err
	}
	a.log(operationID, &destination, "Import", "Restore completed", record.FilePath, record.FileSize, "Info", "Succeeded", "")
	if progress != nil {
		progress("Restore completed", record.FileSizeBytes, record.FileSizeBytes)
	}
	if destination.ImportSettings != nil {
		is := destination.ImportSettings
		if is.RunQueryAfterImport && strings.TrimSpace(is.PostImportQuery) != "" && destination.SupportsImportSQLQuery() {
			query := models.SubstituteQueryDBName(is.PostImportQuery, destination.EffectiveExport().TargetDBName)
			if result, err := a.RunImportQuery(ctx, destination, query, true); err != nil {
				a.log(operationID, &destination, "Post-import query", err.Error(), "", "", "Warning", "Failed", err.Error())
			} else {
				details := formatQueryResultSummary(result)
				a.log(operationID, &destination, "Post-import query", details, "", "", "Info", "Succeeded", "")
			}
		}
	}
	return nil
}

func formatQueryResultSummary(result db.QueryResult) string {
	if result.Message != "" && len(result.Rows) == 0 {
		return result.Message
	}
	if len(result.Rows) == 0 {
		return "Query executed successfully"
	}
	return fmt.Sprintf("%d row(s) returned", len(result.Rows))
}

func (a *App) TestConnection(profile models.Profile, useImport bool) error {
	p := profile.EffectiveExport()
	if useImport {
		p = profile.EffectiveImport()
	}
	if p.ConnectionType == models.ConnectionTypeWordPress {
		_, err := wordpress.NewClient(p.WPUrl, p.WPKey).Ping()
		return err
	}
	client, err := ssh.NewClient(p)
	if err != nil {
		return err
	}
	return client.Close()
}

func (a *App) backupWordPress(ctx context.Context, p models.Profile, progress ProgressFunc) (string, int64, error) {
	fileName := fmt.Sprintf("%s_%s.sql.gz", safeName(p.Name), time.Now().Format("02_01_2006_15_04_05"))
	fullPath := filepath.Join(p.Destination, fileName)
	err := wordpress.NewClient(p.WPUrl, p.WPKey).ExportContext(ctx, fullPath, func(curr int64) {
		if progress != nil {
			progress(fmt.Sprintf("Downloading %.2f MB", float64(curr)/1024/1024), curr, 0)
		}
	})
	if err != nil {
		if ctx.Err() != nil {
			_ = os.Remove(fullPath)
			return "", 0, ctx.Err()
		}
		debug.Errorf("wordpress backup failed for %s: %v", p.Name, err)
		return "", 0, err
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return fullPath, 0, err
	}
	return fullPath, info.Size(), nil
}

func (a *App) backupSSH(ctx context.Context, p models.Profile, progress ProgressFunc) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	client, err := ssh.NewClient(p)
	if err != nil {
		debug.Errorf("backup SSH connect failed for %s: %v", p.Name, err)
		return "", 0, err
	}
	defer client.Close()

	cmd := db.BuildExportCommand(p)
	stdout, stderr, session, err := client.RunCommandStream(cmd)
	if err != nil {
		return "", 0, err
	}
	defer session.Close()
	go func() {
		<-ctx.Done()
		_ = session.Close()
		_ = client.Close()
	}()

	var stderrBuf strings.Builder
	go func() {
		_, _ = io.Copy(&stderrBuf, stderr)
	}()

	fileName := fmt.Sprintf("%s_%s_%s.sql.gz", safeName(p.Name), safeName(p.TargetDBName), time.Now().Format("02_01_2006_15_04_05"))
	fullPath := filepath.Join(p.Destination, fileName)
	out, err := os.Create(fullPath)
	if err != nil {
		return "", 0, err
	}
	defer out.Close()

	written, err := io.Copy(out, &ssh.ProgressReader{
		Reader: stdout,
		Callback: func(current int64, total int64) {
			if progress != nil {
				progress(fmt.Sprintf("Downloading %.2f MB", float64(current)/1024/1024), current, total)
			}
		},
	})
	if ctx.Err() != nil {
		_ = os.Remove(fullPath)
		return "", 0, ctx.Err()
	}
	if err != nil {
		_ = os.Remove(fullPath)
		return "", 0, err
	}
	if err := session.Wait(); err != nil {
		_ = os.Remove(fullPath)
		if ctx.Err() != nil {
			return "", 0, ctx.Err()
		}
		debug.Errorf("backup SSH session failed for %s: %v stderr=%s", p.Name, err, stderrBuf.String())
		return "", 0, fmt.Errorf("%w: %s", err, stderrBuf.String())
	}
	return fullPath, written, nil
}

func (a *App) restoreWordPress(ctx context.Context, path string, p models.Profile, progress ProgressFunc) error {
	var total int64
	if info, err := os.Stat(path); err == nil {
		total = info.Size()
	}
	err := wordpress.NewClient(p.WPUrl, p.WPKey).ImportContext(ctx, path, func(curr int64) {
		if progress != nil {
			progress(fmt.Sprintf("Uploading %.1f%%", percent(curr, total)), curr, total)
		}
	})
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		debug.Errorf("wordpress restore failed for %s: %v", p.Name, err)
	}
	return err
}

func (a *App) restoreSSH(ctx context.Context, path string, p models.Profile, progress ProgressFunc) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	info, _ := in.Stat()
	total := int64(0)
	if info != nil {
		total = info.Size()
	}

	client, err := ssh.NewClient(p)
	if err != nil {
		debug.Errorf("restore SSH connect failed for %s: %v", p.Name, err)
		return err
	}
	defer client.Close()

	importCmd := db.BuildImportCommand(p)
	if db.ImportUsesStreaming(p) {
		if prep := db.BuildImportPrepareCommand(p); prep != "" {
			if out, prepErr := client.RunCommand(prep); prepErr != nil {
				debug.Errorf("restore SSH prepare failed for %s: %v out=%s", p.Name, prepErr, out)
				return fmt.Errorf("%w: %s", prepErr, strings.TrimSpace(out))
			}
		}
		compression, detectErr := detectImportCompression(in)
		if detectErr != nil {
			return detectErr
		}
		importCmd = db.BuildImportStreamCommand(p, compression)
	}
	stdin, stderr, session, err := client.RunCommandPipeInput(importCmd)
	if err != nil {
		return err
	}
	defer session.Close()
	go func() {
		<-ctx.Done()
		_ = session.Close()
		_ = client.Close()
	}()

	var stderrBuf strings.Builder
	go func() {
		_, _ = io.Copy(&stderrBuf, stderr)
	}()

	_, err = io.Copy(stdin, &ssh.ProgressReader{
		Reader: in,
		Total:  total,
		Callback: func(current int64, total int64) {
			if progress != nil {
				progress(fmt.Sprintf("Uploading %.1f%%", percent(current, total)), current, total)
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
		return restoreSSHError(err, &stderrBuf, "upload")
	}
	if err := session.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		debug.Errorf("restore SSH session failed for %s: %v stderr=%s", p.Name, err, stderrBuf.String())
		return restoreSSHError(err, &stderrBuf, "restore")
	}
	return nil
}

func restoreSSHError(err error, stderr *strings.Builder, phase string) error {
	msg := strings.TrimSpace(stderr.String())
	if msg != "" {
		return fmt.Errorf("%s failed: %w: %s", phase, err, msg)
	}
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%s failed: connection closed unexpectedly (EOF); try again or check remote disk, memory, and SSH timeouts", phase)
	}
	return fmt.Errorf("%s failed: %w", phase, err)
}

func detectImportCompression(file *os.File) (string, error) {
	var magic [4]byte
	n, err := io.ReadFull(file, magic[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("detect import compression: %w", err)
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return "", fmt.Errorf("rewind import file: %w", seekErr)
	}
	if n >= 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		return "gzip", nil
	}
	if n >= 4 && magic == [4]byte{0x28, 0xb5, 0x2f, 0xfd} {
		return "zstd", nil
	}
	return "", nil
}

func (a *App) log(operationID string, profile *models.Profile, action, details, filePath, fileSize, level, status, errStr string) {
	entry := models.LogEntry{
		ID:          newID(),
		OperationID: operationID,
		Timestamp:   time.Now(),
		Action:      action,
		Level:       level,
		Details:     details,
		FilePath:    filePath,
		FileSize:    fileSize,
		Status:      status,
		Error:       errStr,
	}
	if profile != nil {
		entry.ProfileID = profile.ID
		entry.ProfileName = profile.Name
	}
	a.mu.Lock()
	a.logs = append(a.logs, entry)
	logs := append([]models.LogEntry(nil), a.logs...)
	a.mu.Unlock()
	_ = a.store.SaveLogs(logs)

	profileName := ""
	if profile != nil {
		profileName = profile.Name
	}
	debug.Log(level, action, status, details, profileName, operationID, errStr)
}

func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "backup"
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	return replacer.Replace(s)
}

func formatSize(size int64) string {
	return fmt.Sprintf("%.2f MB", float64(size)/1024/1024)
}

func percent(current, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(current) / float64(total) * 100
}
