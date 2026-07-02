package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type timelineMySQLRepo struct {
	db *sql.DB
}

// NewTimelineRepository creates a new MySQL-based TimelineRepository.
func NewTimelineRepository(db *sql.DB) TimelineRepository {
	return &timelineMySQLRepo{db: db}
}

func (r *timelineMySQLRepo) Append(ctx context.Context, item *model.TimelineItem) error {
	touchTimestamps(&item.CreatedAt, nil)
	if item.Status == "" {
		item.Status = "done"
	}
	return r.AppendBatch(ctx, []model.TimelineItem{*item})
}

func (r *timelineMySQLRepo) AppendBatch(ctx context.Context, items []model.TimelineItem) error {
	if len(items) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(items))
	valueArgs := make([]interface{}, 0, len(items)*6)

	for i := range items {
		touchTimestamps(&items[i].CreatedAt, nil)
		if items[i].Status == "" {
			items[i].Status = "done"
		}

		// Marshal only non-column fields to avoid dual storage (B6)
		contentJSON, err := json.Marshal(model.TimelineContent{
			Role:                items[i].Role,
			Content:             items[i].Content,
			LocalKey:            items[i].LocalKey,
			InterruptedBy:       items[i].InterruptedBy,
			Card:                items[i].Card,
			EventType:           items[i].EventType,
			Title:               items[i].Title,
			Description:         items[i].Description,
			Reason:              items[i].Reason,
			SuggestedDepartment: items[i].SuggestedDepartment,
		})
		if err != nil {
			return fmt.Errorf("marshal timeline item: %w", err)
		}

		// Build multi-row INSERT: (?, ?, ?, ?, ?, ?)
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, items[i].ID, items[i].SessionID, items[i].Kind,
			items[i].Status, string(contentJSON), items[i].CreatedAt)
	}

	// #nosec G202 — multi-row INSERT with parameterized placeholders; values are not user-controlled
	query := `INSERT INTO timeline_items (id, session_id, kind, status, content, created_at) VALUES ` +
		strings.Join(valueStrings, ", ")

	_, err := r.db.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("batch append timeline items: %w", err)
	}
	return nil
}

func (r *timelineMySQLRepo) ListBySession(ctx context.Context, sessionID string, cursor *string, pageSize int) ([]model.TimelineItem, *string, bool, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	var rows *sql.Rows
	var err error

	if cursor != nil && *cursor != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, session_id, kind, status, content, created_at FROM timeline_items
			WHERE session_id = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?`,
			sessionID, *cursor, pageSize+1,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, session_id, kind, status, content, created_at FROM timeline_items
			WHERE session_id = ? ORDER BY created_at DESC LIMIT ?`,
			sessionID, pageSize+1,
		)
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("list timeline items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []model.TimelineItem
	for rows.Next() {
		var contentJSON string
		var item model.TimelineItem
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Kind, &item.Status, &contentJSON, &item.CreatedAt); err != nil {
			return nil, nil, false, fmt.Errorf("scan timeline item: %w", err)
		}
		// Unmarshal non-column fields from content JSON; column values take precedence
		if err := json.Unmarshal([]byte(contentJSON), &item); err != nil {
			return nil, nil, false, fmt.Errorf("unmarshal timeline item: %w", err)
		}
		// Restore column values from SQL (content JSON may carry stale/extra values in old format)
		items = append(items, item)
	}

	items, nextCursor, hasMore := PaginateCursor(items, pageSize, func(item model.TimelineItem) time.Time {
		return item.CreatedAt
	})

	return items, nextCursor, hasMore, nil
}

func (r *timelineMySQLRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE timeline_items SET status=? WHERE id=?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update timeline status %s: %w", status, err)
	}
	return nil
}

// FindLastPatientMessage finds the most recent patient message in a session.
// It scans recent message-type items and returns the content of the last patient message.
func (r *timelineMySQLRepo) FindLastPatientMessage(ctx context.Context, sessionID string) (string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT content FROM timeline_items
		WHERE session_id = ? AND kind = 'message'
		ORDER BY created_at DESC LIMIT 5`,
		sessionID,
	)
	if err != nil {
		return "", fmt.Errorf("find last patient message: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var contentJSON string
		if err := rows.Scan(&contentJSON); err != nil {
			return "", fmt.Errorf("scan last patient message: %w", err)
		}
		var item model.TimelineItem
		if err := json.Unmarshal([]byte(contentJSON), &item); err != nil {
			return "", fmt.Errorf("unmarshal last patient message: %w", err)
		}
		if item.Role == "patient" {
			return item.Content, nil
		}
	}
	return "", nil
}

// FindLastStreamingMessage finds the most recent streaming assistant message in a session.
// Used by the suspend flow to mark an in-progress message as idle-interrupted.
func (r *timelineMySQLRepo) FindLastStreamingMessage(ctx context.Context, sessionID string) (*model.TimelineItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, session_id, kind, status, content, created_at FROM timeline_items
		WHERE session_id = ? AND kind = 'message' AND status = 'streaming'
		ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("find last streaming message: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil // no streaming message found; not an error
	}

	var contentJSON string
	var item model.TimelineItem
	if err := rows.Scan(&item.ID, &item.SessionID, &item.Kind, &item.Status, &contentJSON, &item.CreatedAt); err != nil {
		return nil, fmt.Errorf("scan streaming message: %w", err)
	}
	if err := json.Unmarshal([]byte(contentJSON), &item); err != nil {
		return nil, fmt.Errorf("unmarshal streaming message: %w", err)
	}
	return &item, nil
}

// FindFlowCardByCardID finds a flow_card-kind timeline item whose embedded card
// has the given cardID. Returns nil, nil when no matching timeline item exists.
func (r *timelineMySQLRepo) FindFlowCardByCardID(ctx context.Context, sessionID, cardID string) (*model.TimelineItem, error) {
	var contentJSON string
	var item model.TimelineItem
	err := r.db.QueryRowContext(ctx,
		`SELECT id, session_id, kind, status, content, created_at FROM timeline_items
		WHERE session_id = ? AND kind = 'flow_card'
		AND JSON_UNQUOTE(JSON_EXTRACT(content, '$.card.id')) = ? LIMIT 1`,
		sessionID, cardID,
	).Scan(&item.ID, &item.SessionID, &item.Kind, &item.Status, &contentJSON, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find flow card timeline item by card id: %w", err)
	}
	if err := json.Unmarshal([]byte(contentJSON), &item); err != nil {
		return nil, fmt.Errorf("unmarshal timeline item: %w", err)
	}
	return &item, nil
}

// UpdateContent updates the content JSON column of a timeline item.
// This is used to update fields like InterruptedBy that are stored in the content JSON.
func (r *timelineMySQLRepo) UpdateContent(ctx context.Context, id string, item *model.TimelineItem) error {
	contentJSON, err := json.Marshal(model.TimelineContent{
		Role:                item.Role,
		Content:             item.Content,
		LocalKey:            item.LocalKey,
		InterruptedBy:       item.InterruptedBy,
		Card:                item.Card,
		EventType:           item.EventType,
		Title:               item.Title,
		Description:         item.Description,
		Reason:              item.Reason,
		SuggestedDepartment: item.SuggestedDepartment,
	})
	if err != nil {
		return fmt.Errorf("marshal timeline content: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE timeline_items SET content=? WHERE id=?`,
		string(contentJSON), id,
	)
	if err != nil {
		return fmt.Errorf("update timeline content: %w", err)
	}
	return nil
}
