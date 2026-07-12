package sqlstore

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

var migrationFiles = []struct {
	version int
	file    string
}{
	{1, "migrations/001_initial.sql"},
	{2, "migrations/002_users.sql"},
	{3, "migrations/003_url_check_samples.sql"},
	{4, "migrations/004_squid_proxies.sql"},
}

func runMigrations(db *sql.DB, driver string) error {
	for _, m := range migrationFiles {
		applied, err := migrationApplied(db, driver, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile(m.file)
		if err != nil {
			return err
		}
		sqlText := string(body)
		if driver == "mysql" {
			sqlText = strings.ReplaceAll(sqlText, "IF NOT EXISTS ", "")
		}
		for _, stmt := range splitSQL(sqlText) {
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				if driver == "mysql" && (strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "already exists")) {
					continue
				}
				return fmt.Errorf("migration v%d: %w\nstmt: %s", m.version, err, stmt)
			}
		}
		_, err = db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, m.version, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	return nil
}

func migrationApplied(db *sql.DB, driver string, version int) (bool, error) {
	if !tableExists(db, driver, "schema_migrations") {
		return false, nil
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func tableExists(db *sql.DB, driver, name string) bool {
	var count int
	var err error
	if driver == "mysql" {
		err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, name).Scan(&count)
	} else {
		err = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&count)
	}
	return err == nil && count > 0
}

func splitSQL(text string) []string {
	var out []string
	var b strings.Builder
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			out = append(out, strings.TrimSpace(b.String()))
			b.Reset()
		}
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		out = append(out, s)
	}
	return out
}
