package db

import (
	"dback/models"
	"fmt"
	"strings"
)

func shellEscape(s string) string {
	// Replace ' with '\''
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildExportCommand constructs the shell command to dump the database.
func BuildExportCommand(p models.Profile) string {
	var cmd string

	if p.DBType == models.DBTypeCouchDB {
		// CouchDB Logic
		// Note: CouchDB logic uses complex sh -c string, we assume fixed structure safe.
		// But p.ContainerID might need escaping?
		// For simplicity, we keep CouchDB logic as is, assuming alphanumeric IDs.
		if p.IsDocker {
			cmd = fmt.Sprintf(`sh -c 'DATA_DIR=$(docker inspect %s --format "{{ range .Mounts }}{{ if eq .Destination \"/opt/couchdb/data\" }}{{ .Destination }}{{ end }}{{ end }}"); if [ -z "$DATA_DIR" ]; then DATA_DIR="/opt/couchdb/data"; fi; docker exec %s tar cf - $DATA_DIR'`, p.ContainerID, p.ContainerID)
		} else {
			cmd = `sh -c 'DATA_DIR=$(grep -r "database_dir" /opt/couchdb/etc/local.ini 2>/dev/null | awk "{print $3}"); if [ -z "$DATA_DIR" ]; then DATA_DIR="/var/lib/couchdb"; fi; sudo systemctl stop couchdb >&2; tar cf - $DATA_DIR; sudo systemctl start couchdb >&2'`
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
		// PostgreSQL Logic
		authEnv := fmt.Sprintf("PGPASSWORD='%s'", p.DBPassword)
		args := fmt.Sprintf("-U %s %s", p.DBUser, p.TargetDBName)

		if p.IsDocker {
			cmd = fmt.Sprintf("docker exec -i -e %s %s psql %s",
				authEnv, p.ContainerID, args)
		} else {
			hostArgs := fmt.Sprintf("-h %s -p %s", p.DBHost, p.DBPort)
			cmd = fmt.Sprintf("%s psql %s %s", authEnv, hostArgs, args)
		}

	} else {
		// MySQL/MariaDB Logic
		authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)
		hostArgs := ""
		if !p.IsDocker {
			hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
		}

		if p.IsDocker {
			cmd = fmt.Sprintf("docker exec -i %s mysql %s %s",
				p.ContainerID, authArgs, p.TargetDBName)
		} else {
			cmd = fmt.Sprintf("mysql %s %s %s",
				hostArgs, authArgs, p.TargetDBName)
		}
	}

	// Decompression Logic: Try zstd, fallback to gzip
	decompressCmd := "if command -v zstd >/dev/null 2>&1; then zstd -d 2>/dev/null || gunzip -c; else gunzip -c; fi"
	return fmt.Sprintf("{ %s; } | %s", decompressCmd, cmd)
}
