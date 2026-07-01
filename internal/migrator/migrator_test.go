package migrator_test

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/neuhis/software-practice-backend/internal/migrator"
	"github.com/neuhis/software-practice-backend/tests/testutil"
)

func TestMigrator_RunsAllMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn, teardown := testutil.SetupMySQL(t)
	defer teardown()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	// First run: all migrations should be applied
	err = migrator.Run(db, "../../db/migrations")
	if err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Verify tracking table has the expected number of rows
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count applied: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one migration applied, got 0")
	}
	t.Logf("first run applied %d migrations", count)

	// Second run: idempotent — no new migrations should be applied
	err = migrator.Run(db, "../../db/migrations")
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	var countAfter int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countAfter); err != nil {
		t.Fatalf("count after second run: %v", err)
	}
	if countAfter != count {
		t.Errorf("expected count to stay at %d after second run, got %d", count, countAfter)
	}
	t.Logf("second run: count unchanged (%d), idempotent", countAfter)
}

func TestMigrator_SchemaIsComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn, teardown := testutil.SetupMySQL(t)
	defer teardown()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	err = migrator.Run(db, "../../db/migrations")
	if err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Verify all expected columns exist in the visits table
	expectedColumns := []string{
		"id", "patient_id", "patient_name", "entry_type", "status", "machine_state",
		"started_at", "updated_at", "ended_at", "timeout_at", "paused_at",
		"last_activity_at", "ask_round", "ask_round_limit", "lab_round", "lab_round_limit",
		"parent_session_id", "terminal_reason", "active_card_id", "medagent_session_id",
		"timer_paused", "summary",
	}

	for _, col := range expectedColumns {
		var colName string
		err := db.QueryRow(
			"SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'visits' AND COLUMN_NAME = ?",
			col,
		).Scan(&colName)
		if err == sql.ErrNoRows {
			t.Errorf("missing column %q in visits table", col)
		} else if err != nil {
			t.Errorf("check column %q: %v", col, err)
		}
	}
}
