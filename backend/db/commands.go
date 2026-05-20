package db

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"dback/models"
)

func shellEscape(s string) string {
	// Replace ' with '\''
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func sqlIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func mysqlOrMariaDB(p models.Profile) bool {
	return p.DBType == models.DBTypeMySQL || p.DBType == models.DBTypeMariaDB
}

// ImportUsesStreaming returns true when restore should run prepare + stream (no /tmp buffer).
func ImportUsesStreaming(p models.Profile) bool {
	return mysqlOrMariaDB(p)
}

// mysqlClientExec builds: if mariadb; then mariadb <args>; else mysql <args>; fi
// extraArgs is appended inside each branch (e.g. -e 'SQL' or empty for stdin import).
func mysqlClientExec(p models.Profile, database, extraArgs string) string {
	authArgs := fmt.Sprintf("-u %s -p%s", p.DBUser, shellEscape(p.DBPassword))
	hostArgs := ""
	if !p.IsDocker {
		hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
	}
	dbArg := ""
	if database != "" {
		dbArg = " " + shellEscape(database)
	}
	if extraArgs != "" {
		extraArgs = " " + extraArgs
	}
	return fmt.Sprintf(
		"if command -v mariadb >/dev/null 2>&1; then mariadb %s%s%s%s; else mysql %s%s%s%s; fi",
		hostArgs, authArgs, dbArg, extraArgs,
		hostArgs, authArgs, dbArg, extraArgs,
	)
}

func importDecompressStream(compression string) string {
	switch compression {
	case "gzip":
		return "gzip -dc"
	case "zstd":
		return "zstd -d -c"
	default:
		return "cat"
	}
}

func shellWithPipefail(script string) string {
	return fmt.Sprintf(
		"if command -v bash >/dev/null 2>&1; then bash -o pipefail -c %s; else sh -c %s; fi",
		shellEscape(script),
		shellEscape(script),
	)
}

// BuildExportCommand constructs the shell command to dump the database.
func BuildExportCommand(p models.Profile) string {
	var cmd string

	if p.DBType == models.DBTypeCouchDB {
		// CouchDB Logic - backup the data directory
		if p.IsDocker {
			// For Docker, backup /opt/couchdb/data from inside the container
			cmd = fmt.Sprintf("docker exec %s tar cf - /opt/couchdb/data", p.ContainerID)
		} else {
			// For native install, find and backup the data directory
			cmd = `sh -c 'DATA_DIR=$(grep -r "database_dir" /opt/couchdb/etc/local.ini 2>/dev/null | awk "{print \$3}"); if [ -z "$DATA_DIR" ]; then DATA_DIR="/var/lib/couchdb"; fi; tar cf - "$DATA_DIR"'`
		}
	} else if p.DBType == models.DBTypePostgreSQL {
		// PostgreSQL Logic
		// Escape inputs
		pwd := shellEscape(p.DBPassword)
		user := shellEscape(p.DBUser)
		dbName := shellEscape(p.TargetDBName)

		// authEnv includes PGPASSWORD='...' which is quoted by shellEscape
		// Wait, shellEscape returns '...'.
		// So PGPASSWORD='...' is PGPASSWORD='pass'.
		// If pass is pass'word, it becomes PGPASSWORD='pass'\''word'. Correct.
		authEnv := fmt.Sprintf("PGPASSWORD=%s", pwd)
		args := fmt.Sprintf("-U %s %s", user, dbName)

		if p.IsDocker {
			cmd = fmt.Sprintf("docker exec -e %s %s pg_dump %s",
				authEnv, p.ContainerID, args)
		} else {
			hostArgs := fmt.Sprintf("-h %s -p %s", p.DBHost, p.DBPort)
			cmd = fmt.Sprintf("%s pg_dump %s %s", authEnv, hostArgs, args)
		}

	} else {
		// MySQL/MariaDB Logic
		// mysql -p'pass'
		// If pass is 'pass', -p'pass'.
		// If pass is pass'word, -p'pass'\''word'.
		// But we need to be careful about -pFLAG format.
		// -p%s.
		pwd := shellEscape(p.DBPassword)
		// shellEscape adds outer quotes.
		// So -p'pass' becomes -p'pass'.
		// Wait, shellEscape returns 'pass'.
		// fmt.Sprintf("-p%s", pwd) -> -p'pass'. Correct.

		authArgs := fmt.Sprintf("-u %s -p%s", p.DBUser, pwd)
		hostArgs := ""
		if !p.IsDocker {
			hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
		}

		if p.IsDocker {
			cmd = fmt.Sprintf("docker exec -i %s mysqldump %s %s",
				p.ContainerID, authArgs, p.TargetDBName)
		} else {
			cmd = fmt.Sprintf("mysqldump %s %s %s",
				hostArgs, authArgs, p.TargetDBName)
		}
	}

	// Compression Logic
	compressCmd := "if command -v zstd >/dev/null 2>&1; then zstd; else gzip; fi"

	// Use set -o pipefail to catch errors.
	// We wrap in bash to ensure pipefail support, but NOT using -c '...' because of quoting hell.
	// We try to run: bash -c "set -o pipefail; CMD | COMPRESS"
	// But CMD has single quotes.
	// Double quotes "..." allow $ expansion.
	// We must escape $ and " and \ inside CMD.
	// This is hard.
	// Alternative: Use { set -o pipefail; cmd; } | compress?
	// No, pipefail must be set in the shell executing the pipeline.
	// If we just send the string `set -o pipefail; cmd | compress` to SSH, it runs in user shell.
	// If user shell is bash, it works.
	// If user shell is sh, it might fail.
	// But wrapping in `bash -c` caused the error.
	// So we simply send it raw and hope for bash/zsh.
	return fmt.Sprintf("set -o pipefail; %s | { %s; }", cmd, compressCmd)
}

// BuildImportCommand constructs the shell command to restore the database.
func BuildImportCommand(p models.Profile) string {
	var cmd string

	if p.DBType == models.DBTypeCouchDB {
		// CouchDB Logic
		if p.IsDocker {
			// Docker: Untar then restart
			cmd = fmt.Sprintf(`sh -c 'docker exec -i %s tar xf - -C /; docker restart %s >&2'`, p.ContainerID, p.ContainerID)
		} else {
			// Native: Stop, Untar, Start
			cmd = `sh -c 'sudo systemctl stop couchdb >&2; tar xf - -C /; sudo systemctl start couchdb >&2'`
		}
	} else if p.DBType == models.DBTypePostgreSQL {
		// PostgreSQL Logic - drop and recreate database before import
		authEnv := fmt.Sprintf("PGPASSWORD='%s'", p.DBPassword)
		
		if p.IsDocker {
			// First drop/create, then pipe stdin to psql for import
			cmd = fmt.Sprintf("sh -c '%s docker exec %s psql -U %s postgres -c \"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s;\" && docker exec -i -e %s %s psql -U %s %s'",
				authEnv, p.ContainerID, p.DBUser, p.TargetDBName, p.TargetDBName,
				authEnv, p.ContainerID, p.DBUser, p.TargetDBName)
		} else {
			hostArgs := fmt.Sprintf("-h %s -p %s", p.DBHost, p.DBPort)
			cmd = fmt.Sprintf("sh -c '%s psql %s -U %s postgres -c \"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s;\" && %s psql %s -U %s %s'",
				authEnv, hostArgs, p.DBUser, p.TargetDBName, p.TargetDBName,
				authEnv, hostArgs, p.DBUser, p.TargetDBName)
		}

	} else if mysqlOrMariaDB(p) {
		return BuildImportStreamCommand(p, "")
	} else {
		return ""
	}

	return cmd
}

// BuildImportPrepareCommand runs DROP/CREATE DATABASE before streaming a MySQL/MariaDB dump.
func BuildImportPrepareCommand(p models.Profile) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	db := sqlIdent(p.TargetDBName)
	sql := fmt.Sprintf(
		"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
		db, db,
	)
	inner := fmt.Sprintf("set -e; %s", mysqlClientExec(p, "", "-e "+shellEscape(sql)))
	if p.IsDocker {
		return fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(inner))
	}
	return fmt.Sprintf("sh -c %s", shellEscape(inner))
}

// BuildImportStreamCommand streams a dump from stdin into mysql/mariadb (no /tmp buffer).
func BuildImportStreamCommand(p models.Profile, compression string) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	client := mysqlClientExec(p, p.TargetDBName, "")
	sessionSetup := `printf "SET SESSION sql_mode=''; SET FOREIGN_KEY_CHECKS=0;\n"`
	pipe := fmt.Sprintf(
		"{ %s; %s; } | %s",
		sessionSetup, importDecompressStream(compression), client,
	)
	cmd := shellWithPipefail(pipe)
	if p.IsDocker {
		return fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(cmd))
	}
	return cmd
}

// BuildQueryCommand constructs a shell command to run SQL via mysql/mariadb CLI.
// When connectDB is false, SQL runs without selecting a database (for DROP/CREATE DATABASE).
func BuildQueryCommand(p models.Profile, query string, connectDB bool) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("query is empty")
	}
	if p.DBType != models.DBTypeMySQL && p.DBType != models.DBTypeMariaDB {
		return "", errors.New("query only supported for MySQL/MariaDB")
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(query))
	b64Esc := shellEscape(b64)

	authArgs := fmt.Sprintf("-u %s -p%s", p.DBUser, shellEscape(p.DBPassword))
	hostArgs := ""
	if !p.IsDocker {
		hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
	}
	batchFlags := "--batch --raw"
	var clientInner string
	if connectDB {
		dbName := shellEscape(p.TargetDBName)
		clientInner = fmt.Sprintf(
			"if command -v mariadb >/dev/null 2>&1; then mariadb %s %s %s %s; else mysql %s %s %s %s; fi",
			hostArgs, authArgs, batchFlags, dbName, hostArgs, authArgs, batchFlags, dbName,
		)
	} else {
		clientInner = fmt.Sprintf(
			"if command -v mariadb >/dev/null 2>&1; then mariadb %s %s %s; else mysql %s %s %s; fi",
			hostArgs, authArgs, batchFlags, hostArgs, authArgs, batchFlags,
		)
	}
	pipe := fmt.Sprintf("echo %s | base64 -d | %s", b64Esc, clientInner)

	if p.IsDocker {
		return fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(pipe)), nil
	}
	return fmt.Sprintf("sh -c %s", shellEscape(pipe)), nil
}
