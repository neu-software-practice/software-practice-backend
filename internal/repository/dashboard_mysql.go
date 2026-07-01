package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type dashboardMySQLRepo struct {
	db *sql.DB
}

// NewDashboardRepository creates a new MySQL-based DashboardRepository.
func NewDashboardRepository(db *sql.DB) DashboardRepository {
	return &dashboardMySQLRepo{db: db}
}

func (r *dashboardMySQLRepo) CountPatients(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM patients`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count patients: %w", err)
	}
	return count, nil
}

func (r *dashboardMySQLRepo) CountPatientsSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM patients WHERE created_at >= ?`, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count patients since: %w", err)
	}
	return count, nil
}

func (r *dashboardMySQLRepo) CountSessions(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM visits`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sessions: %w", err)
	}
	return count, nil
}

func (r *dashboardMySQLRepo) CountActiveSessions(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM visits WHERE status IN (?, ?, ?, ?, ?)`,
		"new", "in_progress", "waiting_lab", "waiting_payment", "waiting_fulfillment",
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active sessions: %w", err)
	}
	return count, nil
}

func (r *dashboardMySQLRepo) CountSessionsSince(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM visits WHERE started_at >= ?`, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sessions since: %w", err)
	}
	return count, nil
}

func (r *dashboardMySQLRepo) ListPatients(ctx context.Context, query model.AdminPatientQuery) ([]model.AdminPatientItem, int, error) {
	whereClause := ""
	args := []interface{}{}
	if query.Search != "" {
		whereClause = `WHERE p.name LIKE ? OR p.phone_masked LIKE ?`
		searchPattern := "%" + query.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Count total
	var total int
	//nolint:gosec // whereClause built from controlled conditions, user inputs use parameterized args
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM patients p %s`, whereClause)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count patients for list: %w", err)
	}

	// Fetch page
	offset := (query.Page - 1) * query.PageSize
	//nolint:gosec // whereClause built from controlled conditions, user inputs use parameterized args
	//nolint:gosec // whereClause built from controlled conditions, user inputs use parameterized args
	listQuery := fmt.Sprintf(
		`SELECT p.id, p.name, p.phone_masked, p.gender,
			'' as birth_date, -- patients table has no birth_date column; kept as placeholder
			p.created_at,
			(SELECT COUNT(*) FROM visits v WHERE v.patient_id = p.id) as session_count
		FROM patients p %s
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?`, whereClause,
	)
	listArgs := append(args, query.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list patients: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []model.AdminPatientItem
	for rows.Next() {
		var item model.AdminPatientItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.RealName, &item.Phone, &item.Gender,
			&item.BirthDate, &createdAt, &item.SessionCount); err != nil {
			return nil, 0, fmt.Errorf("scan patient item: %w", err)
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if items == nil {
		items = []model.AdminPatientItem{}
	}

	return items, total, nil
}

func (r *dashboardMySQLRepo) ListSessions(ctx context.Context, query model.AdminSessionQuery) ([]model.AdminSessionItem, int, error) {
	whereParts := []string{}
	args := []interface{}{}

	if query.Status != "" {
		whereParts = append(whereParts, "v.status = ?")
		args = append(args, query.Status)
	}
	if query.PatientID != "" {
		whereParts = append(whereParts, "v.patient_id = ?")
		args = append(args, query.PatientID)
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// Count total
	var total int
	//nolint:gosec // whereClause built from controlled conditions, user inputs use parameterized args
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM visits v %s`, whereClause,
	)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sessions for list: %w", err)
	}

	// Fetch page
	offset := (query.Page - 1) * query.PageSize
	//nolint:gosec // whereClause built from controlled conditions, user inputs use parameterized args
	listQuery := fmt.Sprintf(
		`SELECT v.id, v.patient_id,
			COALESCE(p.name, '') as patient_name,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(v.summary, '$.title')), '') as title,
			v.status, v.started_at, v.updated_at
		FROM visits v
		LEFT JOIN patients p ON p.id = v.patient_id
		%s
		ORDER BY v.started_at DESC
		LIMIT ? OFFSET ?`, whereClause,
	)
	listArgs := append(args, query.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []model.AdminSessionItem
	for rows.Next() {
		var item model.AdminSessionItem
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&item.ID, &item.PatientID, &item.PatientName, &item.Title,
			&item.Status, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan session item: %w", err)
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		item.UpdatedAt = updatedAt.Format(time.RFC3339)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	if items == nil {
		items = []model.AdminSessionItem{}
	}

	return items, total, nil
}
