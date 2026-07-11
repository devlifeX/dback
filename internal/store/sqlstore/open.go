package sqlstore

import (
	"fmt"

	"dback/internal/config"
)

// OpenStore creates a SQL-backed store.
func OpenStore(baseDir string, dbCfg config.DBConfig) (*Store, error) {
	return openStore(baseDir, dbCfg)
}

func openStore(baseDir string, dbCfg config.DBConfig) (*Store, error) {
	driver := dbCfg.DriverOrDefault()
	sqlDriver := "sqlite3"
	switch driver {
	case config.DBDriverSQLite:
	case config.DBDriverMySQL:
		sqlDriver = "mysql"
	default:
		return nil, fmt.Errorf("unsupported database driver %q", driver)
	}
	dsn, err := dbCfg.ResolveDSN()
	if err != nil {
		return nil, err
	}
	if baseDir == "" {
		baseDir = "."
	}
	return &Store{baseDir: baseDir, driver: sqlDriver, dsn: dsn}, nil
}
