package verify

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dback/backend/db"
	"dback/models"
)

const (
	ModeFast  = "fast"
	ModeExact = "exact"
)

// QueryRunner executes SQL against a host profile.
type QueryRunner interface {
	RunQuery(ctx context.Context, profile models.Profile, query string, connectDB bool) (db.QueryResult, error)
}

// TempDBName returns a unique temporary database name for deep verify.
func TempDBName() string {
	return fmt.Sprintf("dback_verify_%d", time.Now().Unix())
}

// BuildFastTableRowsQuery returns SQL that lists approximate row counts per table.
func BuildFastTableRowsQuery(databaseName string, useDatabaseFunc bool) string {
	schema := strings.ReplaceAll(strings.TrimSpace(databaseName), "'", "''")
	where := fmt.Sprintf("table_schema = '%s'", schema)
	if useDatabaseFunc {
		where = "table_schema = DATABASE()"
	}
	return fmt.Sprintf(
		"SELECT table_name, table_rows FROM information_schema.tables WHERE %s AND table_type = 'BASE TABLE' ORDER BY table_name;",
		where,
	)
}

// ParseTableRowsResult parses a two-column table_name/table_rows result set.
func ParseTableRowsResult(result db.QueryResult) (map[string]int64, error) {
	if len(result.Columns) < 2 {
		return nil, fmt.Errorf("unexpected table rows result: %d columns", len(result.Columns))
	}
	counts := make(map[string]int64, len(result.Rows))
	for _, row := range result.Rows {
		if len(row) < 2 {
			continue
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(row[1]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse row count for %q: %w", name, err)
		}
		counts[name] = n
	}
	return counts, nil
}

// CaptureFingerprint queries the source database and builds a backup fingerprint.
func CaptureFingerprint(ctx context.Context, runner QueryRunner, profile models.Profile, databaseName, mode string) (models.BackupFingerprint, error) {
	if mode == "" {
		mode = ModeFast
	}
	if runner == nil {
		return models.BackupFingerprint{}, fmt.Errorf("query runner is required")
	}

	useDatabaseFunc := profile.UsesWordPress() && strings.TrimSpace(databaseName) == ""
	connectDB := profile.UsesWordPress()

	if mode == ModeFast {
		query := BuildFastTableRowsQuery(databaseName, useDatabaseFunc)
		result, err := runner.RunQuery(ctx, profile, query, connectDB)
		if err != nil {
			return models.BackupFingerprint{}, err
		}
		counts, err := ParseTableRowsResult(result)
		if err != nil {
			return models.BackupFingerprint{}, err
		}
		return buildFingerprint(mode, counts), nil
	}

	query := BuildFastTableRowsQuery(databaseName, useDatabaseFunc)
	result, err := runner.RunQuery(ctx, profile, query, connectDB)
	if err != nil {
		return models.BackupFingerprint{}, err
	}
	tables, err := ParseTableRowsResult(result)
	if err != nil {
		return models.BackupFingerprint{}, err
	}
	if len(tables) == 0 {
		return models.BackupFingerprint{}, fmt.Errorf("no tables found for fingerprint")
	}

	counts := make(map[string]int64, len(tables))
	for table := range tables {
		if err := ctx.Err(); err != nil {
			return models.BackupFingerprint{}, err
		}
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s;", db.SQLIdent(table))
		countResult, err := runner.RunQuery(ctx, profile, countQuery, connectDB)
		if err != nil {
			return models.BackupFingerprint{}, fmt.Errorf("count %s: %w", table, err)
		}
		n, err := parseSingleCount(countResult)
		if err != nil {
			return models.BackupFingerprint{}, fmt.Errorf("count %s: %w", table, err)
		}
		counts[table] = n
	}
	return buildFingerprint(mode, counts), nil
}

// CountTablesExact returns exact row counts for the given tables in databaseName.
func CountTablesExact(ctx context.Context, runner QueryRunner, profile models.Profile, databaseName string, tables []string) (map[string]int64, error) {
	if len(tables) == 0 {
		return map[string]int64{}, nil
	}
	p := profile
	if strings.TrimSpace(databaseName) != "" {
		p.TargetDBName = databaseName
	}
	counts := make(map[string]int64, len(tables))
	for _, table := range tables {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", db.SQLIdent(table))
		result, err := runner.RunQuery(ctx, p, query, true)
		if err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		n, err := parseSingleCount(result)
		if err != nil {
			return nil, fmt.Errorf("count %s: %w", table, err)
		}
		counts[table] = n
	}
	return counts, nil
}

func buildFingerprint(mode string, counts map[string]int64) models.BackupFingerprint {
	tables := make(map[string]models.FingerprintTable, len(counts))
	var total int64
	for name, rows := range counts {
		tables[name] = models.FingerprintTable{Rows: rows}
		total += rows
	}
	return models.BackupFingerprint{
		CapturedAt: time.Now().UTC(),
		Mode:       mode,
		Tables:     tables,
		TotalRows:  total,
	}
}

func parseSingleCount(result db.QueryResult) (int64, error) {
	if len(result.Rows) == 0 || len(result.Rows[0]) == 0 {
		return 0, fmt.Errorf("empty count result")
	}
	return strconv.ParseInt(strings.TrimSpace(result.Rows[0][0]), 10, 64)
}
