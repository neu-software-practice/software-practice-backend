package testutil

import (
	"database/sql"
	"fmt"
	"testing"
)

// DBTest holds the test database connection and provides utilities for per-test isolation.
type DBTest struct {
	DB  *sql.DB
	DSN string
}

// NewDBTest creates a new per-test database and runs migrations.
// It creates a random database name and cleans up after the test.
func NewDBTest(t *testing.T, baseDSN, migrationsDir string) *DBTest {
	t.Helper()

	dbName := fmt.Sprintf("neuhis_test_%s", t.Name())
	dbName = sanitizeDBName(dbName)
	if len(dbName) > 64 {
		dbName = dbName[:64]
	}

	// Connect to base to create the test database
	baseDB, err := sql.Open("mysql", baseDSN)
	if err != nil {
		t.Fatalf("failed to connect to base db: %v", err)
	}
	defer func() { _ = baseDB.Close() }()

	if _, err := baseDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)); err != nil {
		t.Fatalf("failed to create test database %s: %v", dbName, err)
	}

	// Build DSN for the test database
	testDSN := baseDSN + dbName
	if _, err := baseDB.Exec(fmt.Sprintf("USE `%s`", dbName)); err != nil {
		t.Fatalf("failed to use test database: %v", err)
	}

	// Run migrations
	RunMigrations(t, testDSN, migrationsDir)

	// Connect to test database
	testDB, err := sql.Open("mysql", testDSN)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	t.Cleanup(func() {
		testDB.Close()
		if _, err := baseDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)); err != nil {
			t.Logf("failed to drop test database %s: %v", dbName, err)
		}
	})

	return &DBTest{
		DB:  testDB,
		DSN: testDSN,
	}
}

func sanitizeDBName(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
