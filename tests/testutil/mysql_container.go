package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

// SetupMySQL starts a MySQL testcontainer and returns the DSN and a cleanup function.
func SetupMySQL(t *testing.T) (dsn string, teardown func()) {
	t.Helper()

	ctx := context.Background()

	mysqlContainer, err := mysql.RunContainer(ctx,
		testcontainers.WithImage("mysql:8.0"),
		mysql.WithDatabase("neuhis_test"),
		mysql.WithUsername("root"),
		mysql.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("failed to start mysql container: %v", err)
	}

	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get mysql host: %v", err)
	}

	port, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatalf("failed to get mysql port: %v", err)
	}

	dsn = fmt.Sprintf("root:test@tcp(%s:%s)/neuhis_test?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true", host, port.Port())

	// Wait for MySQL to be ready
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("mysql not ready: %v", err)
	}

	teardown = func() {
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate mysql container: %v", err)
		}
	}

	return dsn, teardown
}

// RunMigrations runs all SQL migration files in the given directory against the database.
func RunMigrations(t *testing.T, dsn, migrationsDir string) {
	t.Helper()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open db for migrations: %v", err)
	}
	defer func() { _ = db.Close() }()

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		t.Fatalf("failed to glob migration files: %v", err)
	}
	sort.Strings(files)

	for _, f := range files {
		content, err := os.ReadFile(f) // #nosec G304
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("failed to execute migration %s: %v", f, err)
		}
	}
}
