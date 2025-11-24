package db

import (
	"fmt"
	"dback/models"
)

// BuildExportCommand constructs the shell command to dump the database.
// The output of this command will be the gzipped SQL dump.
func BuildExportCommand(p models.Profile) string {
	var cmd string
	
	// Basic mysqldump arguments
	// Note: passing password directly is insecure in process list, but standard for simple tools.
	// Ideally using a config file, but following prompt examples.
	authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)
	hostArgs := ""
	if !p.IsDocker {
		hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
	}

	if p.IsDocker {
		// Docker: docker exec -i container mysqldump ...
		// We don't typically use -h 127.0.0.1 inside the container unless specified, 
		// usually it connects to localhost inside container.
		cmd = fmt.Sprintf("docker exec -i %s mysqldump %s %s", 
			p.ContainerID, authArgs, p.TargetDBName)
	} else {
		// Native: mysqldump -h ... ...
		cmd = fmt.Sprintf("mysqldump %s %s %s", 
			hostArgs, authArgs, p.TargetDBName)
	}

	// Pipe through gzip
	return fmt.Sprintf("%s | gzip", cmd)
}

// BuildImportCommand constructs the shell command to restore the database.
// It expects the input (stdin) to be a gzipped SQL stream.
// To handle non-gzipped input, we'd need to know the source format, 
// but for this helper we'll assume the transfer sends a gzip stream or we gzip it on the fly.
// Actually, the UI requirement says "Select a local .sql or .sql.gz".
// If we standardize on sending a compressed stream to the server to save bandwidth,
// then the server command should always expect gzip input.
func BuildImportCommand(p models.Profile) string {
	var cmd string

	// Basic mysql arguments
	authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)
	hostArgs := ""
	if !p.IsDocker {
		hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
	}

	// The base command that accepts SQL on stdin
	var mysqlCmd string
	if p.IsDocker {
		// Docker: docker exec -i container mysql ...
		mysqlCmd = fmt.Sprintf("docker exec -i %s mysql %s %s", 
			p.ContainerID, authArgs, p.TargetDBName)
	} else {
		// Native: mysql -h ... ...
		mysqlCmd = fmt.Sprintf("mysql %s %s %s", 
			hostArgs, authArgs, p.TargetDBName)
	}

	// We expect compressed input, so we unzip before piping to mysql
	return fmt.Sprintf("gunzip -c | %s", mysqlCmd)
}
