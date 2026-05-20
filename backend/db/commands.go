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

func mysqlDumpExec(p models.Profile) string {
	authArgs := fmt.Sprintf("-u %s -p%s", p.DBUser, shellEscape(p.DBPassword))
	hostArgs := ""
	if !p.IsDocker {
		hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
	}
	return fmt.Sprintf(
		"if command -v mariadb-dump >/dev/null 2>&1; then mariadb-dump %s %s %s; elif command -v mysqldump >/dev/null 2>&1; then mysqldump %s %s %s; else echo 'no dump tool' >&2; exit 127; fi",
		hostArgs, authArgs, shellEscape(p.TargetDBName),
		hostArgs, authArgs, shellEscape(p.TargetDBName),
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
	return "if command -v zstd >/dev/null 2>&1; then zstd; else gzip; fi"
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
		return fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(shellWithPipefail(inner)))
	}
	return shellWithPipefail(inner)
}

// BuildExportToFileCommand writes compressed dump to remotePath on host.
func BuildExportToFileCommand(p models.Profile, remotePath string) string {
	dump := mysqlDumpExec(p)
	inner := fmt.Sprintf("%s | { %s; } > %s", dump, compressCmd(), shellEscape(remotePath))
	if p.IsDocker {
		// Dump inside container, write to host path via docker exec stdout redirected on host
		containerDump := fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(shellWithPipefail(fmt.Sprintf("%s | { %s; }", dump, compressCmd()))))
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
		return fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(inner))
	}
	return fmt.Sprintf("sh -c %s", shellEscape(inner))
}

// BuildImportStreamCommand streams dump from stdin into mysql/mariadb.
func BuildImportStreamCommand(p models.Profile, compression string) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	client := mysqlClientExec(p, p.TargetDBName, "")
	sessionSetup := `printf "SET SESSION sql_mode=''; SET FOREIGN_KEY_CHECKS=0;\n"`
	pipe := fmt.Sprintf("{ %s; %s; } | %s", sessionSetup, importDecompressStream(compression), client)
	cmd := shellWithPipefail(pipe)
	if p.IsDocker {
		return fmt.Sprintf("docker exec -i %s sh -c %s", p.ContainerID, shellEscape(cmd))
	}
	return cmd
}

// BuildImportFromFileCommand imports from remote compressed file on host.
func BuildImportFromFileCommand(p models.Profile, remotePath, compression string) string {
	if !mysqlOrMariaDB(p) {
		return ""
	}
	client := mysqlClientExec(p, p.TargetDBName, "")
	sessionSetup := `printf "SET SESSION sql_mode=''; SET FOREIGN_KEY_CHECKS=0;\n"`
	pipe := fmt.Sprintf(
		"{ %s; %s %s; } | %s",
		sessionSetup, importDecompressStream(compression), shellEscape(remotePath), client,
	)
	if p.IsDocker {
		hostPipe := fmt.Sprintf("%s | docker exec -i %s sh -c %s",
			importDecompressStream(compression)+" "+shellEscape(remotePath),
			p.ContainerID,
			shellEscape(shellWithPipefail(fmt.Sprintf("{ %s; cat; } | %s", sessionSetup, mysqlClientExec(p, p.TargetDBName, "")))),
		)
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

// BuildPreflightScript returns a shell script that gathers server info and disk space.
func BuildPreflightScript(p models.Profile, requiredBytes int64, candidatePaths []string) string {
	paths := strings.Join(candidatePaths, " ")
	requiredKB := requiredBytes / 1024
	if requiredKB < 512*1024 {
		requiredKB = 512 * 1024 // 512MB minimum safe threshold
	}
	dockerBlock := ""
	preflightChecks := `if ! uname -s 2>/dev/null | grep -qi linux; then fail=1; msg="$msg not-linux;"; fi
command -v sh >/dev/null 2>&1 || { fail=1; msg="$msg missing:sh;"; }
command -v gzip >/dev/null 2>&1 || command -v zstd >/dev/null 2>&1 || { fail=1; msg="$msg missing:compression;"; }`
	if p.IsDocker {
		dockerBlock = fmt.Sprintf(`
echo "===DOCKER==="
command -v docker >/dev/null 2>&1 && docker --version 2>/dev/null || echo "docker missing"
docker inspect -f '{{.State.Status}}' %s 2>/dev/null || echo "container not found"
docker exec %s sh -c 'command -v mysql >/dev/null && mysql --version || command -v mariadb >/dev/null && mariadb --version || echo no mysql client' 2>/dev/null || true
`, shellEscape(p.ContainerID), shellEscape(p.ContainerID))
		preflightChecks += fmt.Sprintf(`
command -v docker >/dev/null 2>&1 || { fail=1; msg="$msg missing:docker;"; }
docker inspect %s >/dev/null 2>&1 || { fail=1; msg="$msg container-not-found;"; }
[ "$(docker inspect -f '{{.State.Status}}' %s 2>/dev/null)" = "running" ] || { fail=1; msg="$msg container-not-running;"; }
docker exec %s sh -c 'command -v mysqldump >/dev/null || command -v mariadb-dump >/dev/null' >/dev/null 2>&1 || { fail=1; msg="$msg missing:container-dump-tool;"; }
docker exec %s sh -c 'command -v mysql >/dev/null || command -v mariadb >/dev/null' >/dev/null 2>&1 || { fail=1; msg="$msg missing:container-mysql-client;"; }`,
			shellEscape(p.ContainerID),
			shellEscape(p.ContainerID),
			shellEscape(p.ContainerID),
			shellEscape(p.ContainerID),
		)
	} else {
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
`, preflightChecks, dbCheck, dockerBlock, paths, paths, requiredKB)
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
