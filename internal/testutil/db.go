// Package testutil provides shared helpers for the test suite. NewDB returns a
// fresh, isolated database so integration tests exercise the real
// handler→service→repository→DB stack against MySQL.
//
// Each call creates an isolated temporary database on the MySQL instance
// specified by TEST_DATABASE_DSN, applies the production golang-migrate
// migrations, and drops the database on test cleanup.
package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/neu-software-practice/software-practice-backend/internal/migrate"
)

// NewDB returns a ready-to-use, empty test database on a real MySQL instance.
// TEST_DATABASE_DSN must be set (e.g. root:rootpw@tcp(127.0.0.1:3307)/mysql?...).
// The database named in the DSN is used only for bootstrap (CREATE / DROP
// DATABASE); the actual test data lives in a temporary database that is
// dropped on t.Cleanup.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_DSN is required for MySQL integration tests; start the test MySQL container with 'make test-mysql-up' or 'docker compose -f docker-compose.test.yml up -d'")
	}
	return newMySQLDB(t, dsn)
}

func newMySQLDB(t *testing.T, baseDSN string) *gorm.DB {
	t.Helper()

	cfg, err := mysql.ParseDSN(baseDSN)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_DSN: %v", err)
	}

	suffix, err := randomSuffix()
	if err != nil {
		t.Fatalf("generate random suffix: %v", err)
	}
	dbName := "his_test_" + suffix

	// Bootstrap: connect to the mysql system database to CREATE / DROP databases.
	bootCfg := *cfg
	bootCfg.DBName = "mysql"
	bootDB, err := sql.Open("mysql", bootCfg.FormatDSN())
	if err != nil {
		t.Fatalf("open bootstrap connection: %v", err)
	}
	defer func() { _ = bootDB.Close() }()
	if err := bootDB.Ping(); err != nil {
		t.Fatalf("ping bootstrap connection (is test MySQL running?): %v", err)
	}

	if _, err := bootDB.Exec(fmt.Sprintf(
		"CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName,
	)); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	// Register cleanup immediately after CREATE DATABASE succeeds, so the
	// database is always dropped even if migrate.Up or gorm.Open fails.
	// t.Cleanup callbacks run in LIFO order, so connection close (registered
	// later) runs before database drop.
	t.Cleanup(func() {
		dropDB, err := sql.Open("mysql", bootCfg.FormatDSN())
		if err != nil {
			t.Logf("cleanup: open bootstrap for DROP DATABASE: %v", err)
			return
		}
		defer func() { _ = dropDB.Close() }()
		if _, err := dropDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)); err != nil {
			t.Logf("cleanup: drop %s: %v", dbName, err)
		}
	})

	// Build DSN pointing to the new database and run migrations.
	testCfg := *cfg
	testCfg.DBName = dbName
	testDSN := testCfg.FormatDSN()
	if err := migrate.Up(testDSN); err != nil {
		t.Fatalf("migrate up on %s: %v", dbName, err)
	}

	db, err := gorm.Open(gmysql.Open(testDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open gorm on %s: %v", dbName, err)
	}

	// Close the GORM connection before the database is dropped (LIFO).
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func randomSuffix() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
