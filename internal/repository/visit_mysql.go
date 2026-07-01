package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type visitMySQLRepo struct {
	db *sql.DB
}

// NewVisitRepository creates a new MySQL-based VisitRepository.
func NewVisitRepository(db *sql.DB) VisitRepository {
	return &visitMySQLRepo{db: db}
}

func (r *visitMySQLRepo) Create(ctx context.Context, visit *model.VisitSession) error {
	summaryJSON, err := json.Marshal(visit.Summary)
	if err != nil {
		return fmt.Errorf("marshal visit summary: %w", err)
	}
	now := time.Now()
	visit.StartedAt = now
	visit.UpdatedAt = now

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO visits (id, patient_id, entry_type, status, machine_state,
		started_at, updated_at, ended_at, timeout_at, paused_at,
		ask_round, ask_round_limit, lab_round, lab_round_limit,
		parent_session_id, terminal_reason, active_card_id, medagent_session_id, timer_paused, summary)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		visit.ID, visit.PatientID, visit.EntryType, visit.Status,
		visit.MachineState, // machine_state
		visit.StartedAt, visit.UpdatedAt, visit.EndedAt, visit.TimeoutAt, visit.PausedAt,
		visit.AskRound, visit.AskRoundLimit, visit.LabRound, visit.LabRoundLimit,
		visit.ParentSessionID, visit.TerminalReason, visit.ActiveCardID,
		visit.MedAgentSessionID, visit.TimerPaused, string(summaryJSON),
	)
	if err != nil {
		return fmt.Errorf("create visit: %w", err)
	}
	return nil
}

func (r *visitMySQLRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	var v model.VisitSession
	var summaryJSON string
	var machineState string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, patient_id, entry_type, status, machine_state,
		started_at, updated_at, ended_at, timeout_at, paused_at,
		ask_round, ask_round_limit, lab_round, lab_round_limit,
		parent_session_id, terminal_reason, active_card_id, medagent_session_id, timer_paused, summary
		FROM visits WHERE id = ?`, id,
	).Scan(&v.ID, &v.PatientID, &v.EntryType, &v.Status,
		&machineState, // machine_state
		&v.StartedAt, &v.UpdatedAt, &v.EndedAt, &v.TimeoutAt, &v.PausedAt,
		&v.AskRound, &v.AskRoundLimit, &v.LabRound, &v.LabRoundLimit,
		&v.ParentSessionID, &v.TerminalReason, &v.ActiveCardID,
		&v.MedAgentSessionID, &v.TimerPaused,
		&summaryJSON,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find visit by id: %w", err)
	}

	v.MachineState = machineState
	_ = json.Unmarshal([]byte(summaryJSON), &v.Summary)
	return &v, nil
}

// scanVisitSummary scans a single VisitSessionSummary row.
func scanVisitSummary(scanner rowScanner) (*model.VisitSessionSummary, error) {
	var s model.VisitSessionSummary
	var summaryJSON string

	err := scanner.Scan(
		&s.ID, &s.PatientID, &s.EntryType, &s.Status,
		&s.StartedAt, &s.UpdatedAt, &s.EndedAt,
		&s.ParentSessionID, &s.TerminalReason,
		&summaryJSON,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(summaryJSON), &s.Summary)
	return &s, nil
}

func (r *visitMySQLRepo) ListByPatient(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}

	var rows *sql.Rows
	var err error

	if cursor != nil && *cursor != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, patient_id, entry_type, status,
			started_at, updated_at, ended_at, parent_session_id, terminal_reason, summary
			FROM visits WHERE patient_id = ? AND started_at < ? ORDER BY started_at DESC LIMIT ?`,
			patientID, *cursor, pageSize+1,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, patient_id, entry_type, status,
			started_at, updated_at, ended_at, parent_session_id, terminal_reason, summary
			FROM visits WHERE patient_id = ? ORDER BY started_at DESC LIMIT ?`,
			patientID, pageSize+1,
		)
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("list visits: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var summaries []model.VisitSessionSummary
	for rows.Next() {
		s, err := scanVisitSummary(rows)
		if err != nil {
			return nil, nil, false, fmt.Errorf("scan visit summary: %w", err)
		}
		summaries = append(summaries, *s)
	}

	summaries, nextCursor, hasMore := PaginateCursor(summaries, pageSize, func(s model.VisitSessionSummary) time.Time {
		return s.StartedAt
	})

	return summaries, nextCursor, hasMore, nil
}

func (r *visitMySQLRepo) UpdateStatus(ctx context.Context, id string, status string, machineState string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE visits SET status=?, machine_state=?, updated_at=? WHERE id=?`,
		status, machineState, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("update visit status: %w", err)
	}
	return nil
}

func (r *visitMySQLRepo) Update(ctx context.Context, visit *model.VisitSession) error {
	summaryJSON, err := json.Marshal(visit.Summary)
	if err != nil {
		return fmt.Errorf("marshal visit summary: %w", err)
	}
	visit.UpdatedAt = time.Now()

	_, err = r.db.ExecContext(ctx,
		`UPDATE visits SET status=?, machine_state=?, updated_at=?, ended_at=?,
		timeout_at=?, paused_at=?, ask_round=?, lab_round=?,
		terminal_reason=?, active_card_id=?, medagent_session_id=?, timer_paused=?, summary=?
		WHERE id=?`,
		visit.Status, visit.MachineState, visit.UpdatedAt, visit.EndedAt,
		visit.TimeoutAt, visit.PausedAt, visit.AskRound, visit.LabRound,
		visit.TerminalReason, visit.ActiveCardID, visit.MedAgentSessionID, visit.TimerPaused,
		string(summaryJSON), visit.ID,
	)
	if err != nil {
		return fmt.Errorf("update visit: %w", err)
	}
	return nil
}
