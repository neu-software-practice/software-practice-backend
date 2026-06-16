package testutil

import (
	"testing"
	"time"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
)

// TestNewDB_RoundTrip is a smoke test for the harness: it must auto-migrate all
// tables and round-trip a row carrying date/datetime/decimal columns.
func TestNewDB_RoundTrip(t *testing.T) {
	db := NewDB(t)

	birth := time.Date(1990, 5, 20, 0, 0, 0, 0, time.UTC)
	visit := time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)
	reg := &model.Register{
		CaseNumber:  "MR0001",
		RealName:    "张三",
		Gender:      "男",
		Birthdate:   birth,
		Age:         36,
		VisitDate:   visit,
		RegistMoney: 12.50,
		VisitState:  1,
	}
	if err := db.Create(reg).Error; err != nil {
		t.Fatalf("create register: %v", err)
	}
	if reg.ID == 0 {
		t.Fatal("expected autoincrement id to be set")
	}

	var got model.Register
	if err := db.First(&got, reg.ID).Error; err != nil {
		t.Fatalf("read register: %v", err)
	}
	if got.RealName != "张三" || got.RegistMoney != 12.50 || got.VisitState != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.Birthdate.Year() != 1990 || got.Birthdate.Month() != time.May {
		t.Fatalf("birthdate not preserved: %v", got.Birthdate)
	}
}
