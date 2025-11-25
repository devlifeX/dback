package db

import (
	"dback/models"
	"fmt"
)

// BuildExportCommand constructs the shell command to dump the database.
// The output of this command will be the gzipped SQL dump.
func BuildExportCommand(p models.Profile) string {
	var cmd string

	if p.DBType == models.DBTypePostgreSQL {
		// PostgreSQL Logic
		// Format: PGPASSWORD='pass' pg_dump -h host -p port -U user dbname

		authEnv := fmt.Sprintf("PGPASSWORD='%s'", p.DBPassword)
		args := fmt.Sprintf("-U %s %s", p.DBUser, p.TargetDBName)

		if p.IsDocker {
			// Docker: docker exec -e PGPASSWORD=... container pg_dump -U user dbname
			// Note: pg_dump connects to localhost inside container by default if -h not specified,
			// or socket. Usually safe to omit -h inside container or use localhost.
			cmd = fmt.Sprintf("docker exec -e %s %s pg_dump %s",
				authEnv, p.ContainerID, args)
		} else {
			// Native
			hostArgs := fmt.Sprintf("-h %s -p %s", p.DBHost, p.DBPort)
			cmd = fmt.Sprintf("%s pg_dump %s %s", authEnv, hostArgs, args)
		}

	} else {
		// MySQL/MariaDB Logic
		authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)
		hostArgs := ""
		if !p.IsDocker {
			hostArgs = fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
		}

		if p.IsDocker {
			// Docker: docker exec -i container mysqldump ...
			cmd = fmt.Sprintf("docker exec -i %s mysqldump %s %s",
				p.ContainerID, authArgs, p.TargetDBName)
		} else {
			// Native: mysqldump -h ... ...
			cmd = fmt.Sprintf("mysqldump %s %s %s",
				hostArgs, authArgs, p.TargetDBName)
		}
	}

	// Pipe through gzip
	return fmt.Sprintf("%s | gzip", cmd)
}

// BuildImportCommand constructs the shell command to restore the database.
// It expects the input (stdin) to be a gzipped SQL stream.
func BuildImportCommand(p models.Profile) string {
	var cmd string

	if p.DBType == models.DBTypePostgreSQL {
		// PostgreSQL Logic
		// psql usually takes SQL on stdin.
		// Format: PGPASSWORD='pass' psql -h host -p port -U user dbname

		authEnv := fmt.Sprintf("PGPASSWORD='%s'", p.DBPassword)
		args := fmt.Sprintf("-U %s %s", p.DBUser, p.TargetDBName)

		if p.IsDocker {
			// Docker: docker exec -i -e PGPASSWORD=... container psql ...
			cmd = fmt.Sprintf("docker exec -i -e %s %s psql %s",
				authEnv, p.ContainerID, args)
		} else {
			// Native
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
			// Docker: docker exec -i container mysql ...
			cmd = fmt.Sprintf("docker exec -i %s mysql %s %s",
				p.ContainerID, authArgs, p.TargetDBName)
		} else {
			// Native: mysql -h ... ...
			cmd = fmt.Sprintf("mysql %s %s %s",
				hostArgs, authArgs, p.TargetDBName)
		}
	}

	// We expect compressed input, so we unzip before piping to db command
	return fmt.Sprintf("gunzip -c | %s", cmd)
}
