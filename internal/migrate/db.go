package migrate

import (
	"database/sql"
	"fmt"

	driver "github.com/go-sql-driver/mysql"
)

// openMySQL opens a *sql.DB for golang-migrate. It forces MultiStatements so a
// single migration file containing several statements runs as one Exec, and
// ParseTime for consistency with the GORM connection. The same GORM DSN is
// reused — we only flip these driver flags.
func openMySQL(dsn string) (*sql.DB, error) {
	cfg, err := driver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn: %w", err)
	}
	cfg.MultiStatements = true
	cfg.ParseTime = true

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}
