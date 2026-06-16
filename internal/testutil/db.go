// Package testutil provides shared helpers for the test suite. NewDB returns a
// fresh, isolated database so integration tests exercise the real
// handler→service→repository→DB stack (PLAN §5).
//
// By default it uses an in-memory-style SQLite file (pure Go, no CGO, perfectly
// isolated per test). When TEST_DATABASE_DSN is set — e.g. the CI MySQL service
// job — it runs the real golang-migrate migrations against MySQL instead, so the
// production schema is exercised end-to-end.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/neu-software-practice/software-practice-backend/internal/migrate"
	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// NewDB returns a ready-to-use, empty test database.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
		return newMySQLDB(t, dsn)
	}
	return newSQLiteDB(t)
}

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "his_test.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(model.All()...); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func newMySQLDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	if err := migrate.Up(dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	truncateAll(t, db)
	return db
}

// truncateAll empties every table so MySQL-backed tests start clean. There are
// no FK constraints (only indexes), so order is irrelevant; the FK-check toggle
// is kept defensive.
func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	for _, m := range model.All() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			t.Fatalf("parse model %T: %v", m, err)
		}
		db.Exec("TRUNCATE TABLE " + stmt.Schema.Table)
	}
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}
