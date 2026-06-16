// Package migrate runs the embedded golang-migrate migrations against a MySQL
// DSN. Used by cmd/migrate and by the integration-test harness so production and
// tests share one schema source of truth.
package migrate

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	migrations "github.com/neu-software-practice/software-practice-backend/migrations"
)

// newMigrator builds a migrate.Migrate bound to the embedded files and the
// given GORM-style DSN. golang-migrate expects a "mysql://" URL, so callers pass
// a DSN and we wrap it with the mysql database driver directly.
func newMigrator(dsn string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}

	db, err := openMySQL(dsn)
	if err != nil {
		return nil, err
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("init mysql migrate driver: %w", err)
	}
	return migrate.NewWithInstance("iofs", src, "mysql", driver)
}

// Up applies all pending migrations. A no-op (ErrNoChange) is not an error.
func Up(dsn string) error {
	m, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Down rolls every migration back. Used by tests to reset the schema.
func Down(dsn string) error {
	m, err := newMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
