package db

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"dback/models"
)

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func sqlIdent(name string) string {
	return SQLIdent(name)
}

// SQLIdent quotes a MySQL identifier.
func SQLIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func mysqlOrMariaDB(p models.Profile) bool {
	return p.DBType == models.DBTypeMySQL || p.DBType == models.DBTypeMariaDB
}

func ImportUsesStreaming(p models.Profile) bool {
	return mysqlOrMariaDB(p)
}

func MaskCommand(cmd string) string {
	out := maskMySQLPasswordArgs(cmd)
	out = strings.ReplaceAll(out, "PGPASSWORD=", "PGPASSWORD=***")
	return out
}

func maskMySQLPasswordArgs(cmd string) string {
	var b strings.Builder
	searchFrom := 0
	for {
		rel := strings.Index(cmd[searchFrom:], "-p")
		if rel < 0 {
			b.WriteString(cmd[searchFrom:])
			break
		}
		idx := searchFrom + rel
		b.WriteString(cmd[searchFrom:idx])
		end := idx + 2
		for end < len(cmd) && !strings.ContainsRune(" \t\r\n|;)", rune(cmd[end])) {
			end++
		}
		b.WriteString("-p'***'")
		searchFrom = end
	}
	return b.String()
}

func mysqlClientExec(p models.Profile, database, extraArgs string) string {
	authArgs := fmt.Sprintf("-u %s -p%s", shellEscape(p.DBUser), shellEscape(p.DBPassword))
	hostArgs := ""
	if p.DBHost != "" {
		hostArgs = fmt.Sprintf("-h %s -P %s", shellEscape(p.DBHost), shellEscape(p.DBPort))
	}
	dbArg := ""
	if database != "" {
		dbArg = " " + shellEscape(database)
	}
	if extraArgs != "" {
		extraArgs = " " + extraArgs
	}
	return fmt.Sprintf(
		"if command -v mariadb >/dev/null 2>&1; then mariadb %s %s%s%s; else mysql %s %s%s%s; fi",
		hostArgs, authArgs, dbArg, extraArgs,
		hostArgs, authArgs, dbArg, extraArgs,
	)
}

func mysqlDumpArgs(p models.Profile) string {
	flags := []string{
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"--routines",
		"--triggers",
		"--events",
		"--hex-blob",
		"--default-character-set=utf8mb4",
		"--skip-comments",
	}
	if p.DBType == models.DBTypeMySQL {
		flags = append(flags, "--set-gtid-purged=OFF")
	}
	return strings.Join(flags, " ")
}

func mysqlDumpMySQL8FlagSetup() string {
	// Oracle/MySQL Community prints "Ver 8.x"; MariaDB-based builds often use "Distrib 8.x".
	return `_mf=""; _mx=$(mysqldump --version 2>&1); case "$_mx" in *"Distrib 8."*|*"Distrib 9."*|*"Ver 8."*|*"Ver 9."*) _mf="--column-statistics=0 --no-tablespaces";; esac;`
}

func mysqlDumpExec(p models.Profile) string {
	dumpArgs := mysqlDumpArgs(p)
	authArgs := fmt.Sprintf("-u %s -p%s", shellEscape(p.DBUser), shellEscape(p.DBPassword))
	hostArgs := ""
	if p.DBHost != "" {
		hostArgs = fmt.Sprintf("-h %s -P %s", shellEscape(p.DBHost), shellEscape(p.DBPort))
	}
	dbArg := shellEscape(p.TargetDBName)
	mysqlDump := fmt.Sprintf("mysqldump %s %s %s %s", hostArgs, authArgs, dumpArgs, dbArg)
	if p.DBType == models.DBTypeMySQL {
		mysqlDump = fmt.Sprintf("%s mysqldump %s %s %s $_mf %s", mysqlDumpMySQL8FlagSetup(), hostArgs, authArgs, dumpArgs, dbArg)
	}
	return fmt.Sprintf(
		"if command -v mariadb-dump >/dev/null 2>&1; then mariadb-dump %s %s %s %s; elif command -v mysqldump >/dev/null 2>&1; then %s; else echo 'no dump tool' >&2; exit 127; fi",
		hostArgs, authArgs, dumpArgs, dbArg,
		mysqlDump,
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

func compressCmd() string {
	// pigz/gzip -1: fast compression (phpMyAdmin-like). zstd -1 as fallback.
	return "if command -v pigz >/dev/null 2>&1; then pigz -1; elif command -v gzip >/dev/null 2>&1; then gzip -1; elif command -v zstd >/dev/null 2>&1; then zstd -1; else gzip -1; fi"
}

// BuildNativeExportCommand streams dump from native host tools.
func BuildNativeExportCommand(p models.Profile) string {
	return BuildExportCommand(cloneDockerMode(p, false))
}

// BuildDockerExportCommand streams dump from a docker container.
func BuildDockerExportCommand(p models.Profile) string {
	return BuildExportCommand(cloneDockerMode(p, true))
}

// BuildNativeExportToFileCommand writes compressed dump on native host.
func BuildNativeExportToFileCommand(p models.Profile, remotePath string) string {
	return BuildExportToFileCommand(cloneDockerMode(p, false), remotePath)
}

// BuildDockerExportToFileCommand writes compressed dump from container to host file.
func BuildDockerExportToFileCommand(p models.Profile, remotePath string) string {
	return BuildExportToFileCommand(cloneDockerMode(p, true), remotePath)
}

func cloneDockerMode(p models.Profile, docker bool) models.Profile {
	p.IsDocker = docker
	return p
}

// BuildExportCommand constructs the shell command to dump the database (streaming).
func BuildExportCommand(p models.Profile) string {
	dump := mysqlDumpExec(p)
	inner := fmt.Sprintf("%s | { %s; }", dump, compressCmd())
	if p.IsDocker {
		cmd, err := dockerExecCommand(p.ContainerID, inner)
		if err != nil {
			return ""
		}
		return cmd
	}
	return shellWithPipefail(inner)
}

// BuildExportToFileCommand writes compressed dump to remotePath on host.
func BuildExportToFileCommand(p models.Profile, remotePath string) string {
	dump := mysqlDumpExec(p)
	inner := fmt.Sprintf("%s | { %s; } > %s", dump, compressCmd(), shellEscape(remotePath))
	if p.IsDocker {
		containerDump, err := dockerExecCommand(p.ContainerID, fmt.Sprintf("%s | { %s; }", dump, compressCmd()))
		if err != nil {
			return ""
		}
		return shellWithPipefail(fmt.Sprintf("%s > %s", containerDump, shellEscape(remotePath)))
	}
	return shellWithPipefail(inner)
}

// BuildImportCommand constructs restore command (streaming default).
func BuildImportCommand(p models.Profile) string {
	if mysqlOrMariaDB(p) {
		return BuildImportStreamCommand(p, "")
	}
	return ""
}

// BuildDropDatabaseCommand returns SQL that drops a database by name.
func BuildDropDatabaseCommand(databaseName string) string {
	db := sqlIdent(databaseName)
	return fmt.Sprintf("DROP DATABASE IF EXISTS %s;", db)
}

// BuildImportPrepareTempCommand runs DROP/CREATE for a temporary verify database.
func BuildImportPrepareTempCommand(p models.Profile, tempDBName string) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	db := sqlIdent(tempDBName)
	sql := fmt.Sprintf(
		"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
		db, db,
	)
	inner := fmt.Sprintf("set -e; %s", mysqlClientExec(p, "", "-e "+shellEscape(sql)))
	if p.IsDocker {
		cmd, err := dockerExecCommand(p.ContainerID, inner)
		if err != nil {
			return ""
		}
		return cmd
	}
	return fmt.Sprintf("sh -c %s", shellEscape(inner))
}

// BuildImportPrepareCommand runs DROP/CREATE DATABASE before streaming import.
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
		cmd, err := dockerExecCommand(p.ContainerID, inner)
		if err != nil {
			return ""
		}
		return cmd
	}
	return fmt.Sprintf("sh -c %s", shellEscape(inner))
}

// BuildImportStreamCommand streams dump from stdin into mysql/mariadb.
func BuildImportStreamCommand(p models.Profile, compression string) string {
	return buildImportStreamCommand(p, compression, "")
}

// BuildImportStreamCommandForVerify streams a dump into a temp verify database without
// allowing USE/CREATE/DROP statements in the SQL to touch other databases.
func BuildImportStreamCommandForVerify(p models.Profile, compression, tempDBName string) string {
	return buildImportStreamCommand(p, compression, tempDBName)
}

func importVerifySanitizeFilter(tempDBName string) string {
	ident := sqlIdent(tempDBName)
	return fmt.Sprintf(
		`sed -e '/^CREATE DATABASE/IId' -e '/^DROP DATABASE/IId' -e 's/^USE `+"`"+`[^`+"`"+`]*`+"`"+`/USE %s/I'`,
		ident,
	)
}

func buildImportStreamCommand(p models.Profile, compression, verifyTempDB string) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	client := mysqlClientExec(p, p.TargetDBName, "")
	sessionSetup := `printf "SET SESSION sql_mode=''; SET FOREIGN_KEY_CHECKS=0;\n"`
	stream := importDecompressStream(compression)
	if verifyTempDB != "" {
		stream = fmt.Sprintf("%s | %s", stream, importVerifySanitizeFilter(verifyTempDB))
	}
	pipe := fmt.Sprintf("{ %s; %s; } | %s", sessionSetup, stream, client)
	cmd := shellWithPipefail(pipe)
	if p.IsDocker {
		dockerCmd, err := dockerExecCommand(p.ContainerID, cmd)
		if err != nil {
			return ""
		}
		return dockerCmd
	}
	return cmd
}

// BuildImportFromFileCommand imports from remote compressed file on host.
func BuildImportFromFileCommand(p models.Profile, remotePath, compression string) string {
	return buildImportFromFileCommand(p, remotePath, compression, "")
}

// BuildImportFromFileCommandForVerify imports into a temp verify database safely.
func BuildImportFromFileCommandForVerify(p models.Profile, remotePath, compression, tempDBName string) string {
	return buildImportFromFileCommand(p, remotePath, compression, tempDBName)
}

func buildImportFromFileCommand(p models.Profile, remotePath, compression, verifyTempDB string) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	client := mysqlClientExec(p, p.TargetDBName, "")
	sessionSetup := `printf "SET SESSION sql_mode=''; SET FOREIGN_KEY_CHECKS=0;\n"`
	stream := fmt.Sprintf("%s %s", importDecompressStream(compression), shellEscape(remotePath))
	if verifyTempDB != "" {
		stream = fmt.Sprintf("%s | %s", stream, importVerifySanitizeFilter(verifyTempDB))
	}
	pipe := fmt.Sprintf(
		"{ %s; %s; } | %s",
		sessionSetup, stream, client,
	)
	if p.IsDocker {
		containerClient, err := dockerExecCommand(p.ContainerID, shellWithPipefail(fmt.Sprintf("{ %s; cat; } | %s", sessionSetup, mysqlClientExec(p, p.TargetDBName, ""))))
		if err != nil {
			return ""
		}
		hostStream := importDecompressStream(compression) + " " + shellEscape(remotePath)
		if verifyTempDB != "" {
			hostStream = fmt.Sprintf("%s | %s", hostStream, importVerifySanitizeFilter(verifyTempDB))
		}
		hostPipe := fmt.Sprintf("%s | %s", hostStream, containerClient)
		return shellWithPipefail(hostPipe)
	}
	return shellWithPipefail(pipe)
}

// BuildQueryCommand runs SQL via mysql/mariadb CLI.
func BuildQueryCommand(p models.Profile, query string, connectDB bool) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("query is empty")
	}
	if !mysqlOrMariaDB(p) {
		return "", errors.New("query only supported for MySQL/MariaDB")
	}
	if err := ValidateProfileForRemoteOps(p); err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(query))
	b64Esc := shellEscape(b64)

	authArgs := fmt.Sprintf("-u %s -p%s", shellEscape(p.DBUser), shellEscape(p.DBPassword))
	hostArgs := ""
	if p.DBHost != "" {
		hostArgs = fmt.Sprintf("-h %s -P %s", shellEscape(p.DBHost), shellEscape(p.DBPort))
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
		cmd, err := dockerExecCommand(p.ContainerID, pipe)
		if err != nil {
			return "", err
		}
		return cmd, nil
	}
	return fmt.Sprintf("sh -c %s", shellEscape(pipe)), nil
}

func containerDumpProbe() string {
	return "(command -v mysqldump >/dev/null 2>&1 && mysqldump --version) || (command -v mariadb-dump >/dev/null 2>&1 && mariadb-dump --version) || echo no-dump-tool"
}

func containerClientProbe() string {
	return "(command -v mysql >/dev/null 2>&1 && mysql --version) || (command -v mariadb >/dev/null 2>&1 && mariadb --version) || echo no-mysql-client"
}

func recordPreflightCheck(name, cmd string) string {
	displayCmd := strings.ReplaceAll(cmd, "|", "/")
	return fmt.Sprintf(`
echo "check|%s|cmd|%s"
__rc=0
__out=$(%s 2>&1) || __rc=$?
echo "check|%s|exit|$__rc"
echo "check|%s|out|$(printf '%%s' "$__out" | tr '\n' ' ' | sed 's/  */ /g' | sed 's/^ *//;s/ *$//')"
`, name, displayCmd, cmd, name, name)
}

// BuildPreflightScript returns a shell script that gathers server info and disk space.
func BuildPreflightScript(p models.Profile, requiredBytes int64, candidatePaths []string) string {
	paths := strings.Join(candidatePaths, " ")
	requiredKB := requiredBytes / 1024
	if requiredKB < 512*1024 {
		requiredKB = 512 * 1024 // 512MB minimum safe threshold
	}
	dockerBlock := ""
	checksBlock := ""
	preflightChecks := `if ! uname -s 2>/dev/null | grep -qi linux; then fail=1; msg="$msg not-linux;"; fi
command -v sh >/dev/null 2>&1 || { fail=1; msg="$msg missing:sh;"; }
command -v gzip >/dev/null 2>&1 || command -v zstd >/dev/null 2>&1 || { fail=1; msg="$msg missing:compression;"; }`
	if p.IsDocker {
		cid := shellEscape(p.ContainerID)
		dumpProbe := shellEscape(containerDumpProbe())
		clientProbe := shellEscape(containerClientProbe())
		dumpExec := fmt.Sprintf("docker exec %s sh -c %s", cid, dumpProbe)
		clientExec := fmt.Sprintf("docker exec %s sh -c %s", cid, clientProbe)
		dumpPathExec := fmt.Sprintf("docker exec %s sh -c %s", cid, shellEscape("command -v mysqldump 2>/dev/null || command -v mariadb-dump 2>/dev/null"))
		clientPathExec := fmt.Sprintf("docker exec %s sh -c %s", cid, shellEscape("command -v mysql 2>/dev/null || command -v mariadb 2>/dev/null"))
		dockerBlock = fmt.Sprintf(`
echo "===DOCKER==="
command -v docker >/dev/null 2>&1 && docker --version 2>/dev/null || echo "docker missing"
docker inspect -f '{{.State.Status}}' %s 2>/dev/null || echo "container not found"
%s 2>/dev/null || true
%s 2>/dev/null || true
`, cid, dumpExec, clientExec)
		checksBlock = strings.Join([]string{
			recordPreflightCheck("container_dump_version", dumpExec),
			recordPreflightCheck("container_client_version", clientExec),
			recordPreflightCheck("container_dump_path", dumpPathExec),
			recordPreflightCheck("container_client_path", clientPathExec),
		}, "")
		preflightChecks += fmt.Sprintf(`
command -v docker >/dev/null 2>&1 || { fail=1; msg="$msg missing:docker;"; }
docker inspect %s >/dev/null 2>&1 || { fail=1; msg="$msg container-not-found;"; }
[ "$(docker inspect -f '{{.State.Status}}' %s 2>/dev/null)" = "running" ] || { fail=1; msg="$msg container-not-running;"; }
%s >/dev/null 2>&1 || { fail=1; msg="$msg missing:container-dump-tool;"; }
%s >/dev/null 2>&1 || { fail=1; msg="$msg missing:container-mysql-client;"; }`,
			cid,
			cid,
			dumpPathExec,
			clientPathExec,
		)
	} else {
		checksBlock = strings.Join([]string{
			recordPreflightCheck("dump_version", "(command -v mysqldump >/dev/null 2>&1 && mysqldump --version) || (command -v mariadb-dump >/dev/null 2>&1 && mariadb-dump --version) || echo no-dump-tool"),
			recordPreflightCheck("client_version", "(command -v mysql >/dev/null 2>&1 && mysql --version) || (command -v mariadb >/dev/null 2>&1 && mariadb --version) || echo no-mysql-client"),
			recordPreflightCheck("dump_path", "command -v mysqldump 2>/dev/null || command -v mariadb-dump 2>/dev/null"),
			recordPreflightCheck("client_path", "command -v mysql 2>/dev/null || command -v mariadb 2>/dev/null"),
		}, "")
		preflightChecks += `
command -v mysqldump >/dev/null 2>&1 || command -v mariadb-dump >/dev/null 2>&1 || { fail=1; msg="$msg missing:dump-tool;"; }
command -v mysql >/dev/null 2>&1 || command -v mariadb >/dev/null 2>&1 || { fail=1; msg="$msg missing:mysql-client;"; }`
	}
	dbCheck := "command -v mysql >/dev/null && mysql --version 2>/dev/null; command -v mariadb >/dev/null && mariadb --version 2>/dev/null; command -v mysqldump >/dev/null && mysqldump --version 2>/dev/null; command -v mariadb-dump >/dev/null && mariadb-dump --version 2>/dev/null"
	if p.IsDocker {
		dbCheck = fmt.Sprintf("docker exec %s sh -c 'command -v mysqldump >/dev/null && mysqldump --version; command -v mariadb-dump >/dev/null && mariadb-dump --version' 2>/dev/null || true", shellEscape(p.ContainerID))
	}
	return fmt.Sprintf(`set +e
fail=0
msg=""
%s
echo "===OS==="
uname -a 2>/dev/null
(lsb_release -a 2>/dev/null || cat /etc/os-release 2>/dev/null)
echo "===DB==="
%s
echo "===TOOLS==="
command -v zstd >/dev/null && zstd --version 2>/dev/null | head -1
command -v gzip >/dev/null && gzip --version 2>/dev/null | head -1
command -v sha256sum >/dev/null && echo sha256sum ok
command -v dd >/dev/null && echo dd ok
%s
echo "===CHECKS==="
%s
echo "===DISK==="
for p in %s; do
  eval target="$p"
  df -Pk "$target" 2>/dev/null | tail -1 | awk -v requested="$p" '{print $1"|"$4"|"requested}'
done
echo "===WRITE==="
for p in %s; do
  eval target="$p"
  mkdir -p "$target" 2>/dev/null && touch "$target/.dback-write-test" 2>/dev/null && rm -f "$target/.dback-write-test" 2>/dev/null && echo "ok|$p"
done
echo "===REQUIRED_KB==="
echo %d
echo "===RESULT==="
echo "fail=$fail"
echo "msg=$msg"
`, preflightChecks, dbCheck, dockerBlock, checksBlock, paths, paths, requiredKB)
}

// BuildRemoteTmpDir returns operation-specific tmp dir on remote host.
func BuildRemoteTmpDir(operationID string) string {
	return fmt.Sprintf("/tmp/dback/%s", operationID)
}

// BuildCleanupCommand removes remote tmp directory.
func BuildCleanupCommand(tmpDir string) string {
	return fmt.Sprintf("rm -rf %s", shellEscape(tmpDir))
}

// BuildFileSizeCommand returns remote file size in bytes.
func BuildFileSizeCommand(path string) string {
	return fmt.Sprintf("stat -c %%s %s 2>/dev/null || wc -c < %s", shellEscape(path), shellEscape(path))
}

// BuildChecksumCommand returns sha256 checksum of remote file.
func BuildChecksumCommand(path string) string {
	return fmt.Sprintf("sha256sum %s 2>/dev/null | awk '{print $1}'", shellEscape(path))
}

// BuildUploadCommand writes stdin to remote path, optionally appending.
func BuildUploadCommand(path string, appendMode bool) string {
	if appendMode {
		return fmt.Sprintf("dd of=%s oflag=append conv=notrunc 2>/dev/null || cat >> %s", shellEscape(path), shellEscape(path))
	}
	return fmt.Sprintf("cat > %s", shellEscape(path))
}

// BuildDownloadChunkCommand reads file from offset (best-effort resume).
func BuildDownloadChunkCommand(path string, offset int64) string {
	if offset <= 0 {
		return fmt.Sprintf("cat %s", shellEscape(path))
	}
	return fmt.Sprintf("dd if=%s bs=1M skip=%d 2>/dev/null || tail -c +%d %s",
		shellEscape(path), offset/1024/1024, offset+1, shellEscape(path))
}

// BuildDatabaseApproxRowCountCommand returns approximate total row count from information_schema.
func BuildDatabaseApproxRowCountCommand(p models.Profile) (string, error) {
	if !mysqlOrMariaDB(p) {
		return "", errors.New("row count only supported for MySQL/MariaDB")
	}
	if err := ValidateProfileForRemoteOps(p); err != nil {
		return "", err
	}
	dbName := strings.ReplaceAll(strings.TrimSpace(p.TargetDBName), "'", "''")
	if dbName == "" {
		return "", errors.New("target database name is required")
	}
	query := fmt.Sprintf(
		"SELECT COALESCE(SUM(TABLE_ROWS), 0) FROM information_schema.tables WHERE table_schema = '%s'",
		dbName,
	)
	return BuildQueryCommand(p, query, false)
}

// BuildDatabaseSizeCommand returns a remote command that estimates uncompressed DB size in bytes.
func BuildDatabaseSizeCommand(p models.Profile) (string, error) {
	if !mysqlOrMariaDB(p) {
		return "", errors.New("database size estimate only supported for MySQL/MariaDB")
	}
	if err := ValidateProfileForRemoteOps(p); err != nil {
		return "", err
	}
	dbName := strings.ReplaceAll(strings.TrimSpace(p.TargetDBName), "'", "''")
	if dbName == "" {
		return "", errors.New("target database name is required")
	}
	query := fmt.Sprintf(
		"SELECT COALESCE(SUM(data_length + index_length), 0) FROM information_schema.tables WHERE table_schema = '%s'",
		dbName,
	)
	return BuildQueryCommand(p, query, false)
}

// ParseDatabaseSizeBytes extracts the first integer byte count from mysql batch output.
func ParseDatabaseSizeBytes(out string) int64 {
	result := ParseMySQLBatchOutput(strings.TrimSpace(out))
	if len(result.Rows) > 0 && len(result.Rows[0]) > 0 {
		var size int64
		if _, err := fmt.Sscanf(strings.TrimSpace(result.Rows[0][0]), "%d", &size); err == nil && size > 0 {
			return size
		}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "COALESCE") {
			continue
		}
		var size int64
		if _, err := fmt.Sscanf(line, "%d", &size); err == nil && size > 0 {
			return size
		}
	}
	return 0
}

// EstimateCompressedBackupSize converts raw InnoDB bytes to an expected .sql.gz size.
func EstimateCompressedBackupSize(rawBytes int64) int64 {
	if rawBytes <= 0 {
		return 0
	}
	// SQL text dumps (WordPress, etc.) gzip to roughly 3–8% of on-disk InnoDB size.
	estimated := rawBytes / 20
	if estimated < 512*1024 {
		if rawBytes >= 512*1024 {
			estimated = 512 * 1024
		} else {
			estimated = rawBytes / 5
			if estimated < 1024 {
				return 0
			}
		}
	}
	return estimated
}
