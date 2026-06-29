package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.Status == "" {
		item.Status = "done"
	}
	return r.AppendBatch(ctx, []model.TimelineItem{*item})
}

func (r *timelineMySQLRepo) AppendBatch(ctx context.Context, items []model.TimelineItem) error {
	for i := range items {
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
		if items[i].Status == "" {
			items[i].Status = "done"
		}

		contentJSON, err := json.Marshal(items[i])
		if err != nil {
			return fmt.Errorf("marshal timeline item: %w", err)
		}

		_, err = r.db.ExecContext(ctx,
			`INSERT INTO timeline_items (id, session_id, kind, status, content, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			items[i].ID, items[i].SessionID, items[i].Kind, items[i].Status,
			string(contentJSON), items[i].CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("append timeline item: %w", err)
		}
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
			`SELECT content FROM timeline_items
			WHERE session_id = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?`,
			sessionID, *cursor, pageSize+1,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT content FROM timeline_items
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
		if err := rows.Scan(&contentJSON); err != nil {
			return nil, nil, false, fmt.Errorf("scan timeline item: %w", err)
		}
		var item model.TimelineItem
		if err := json.Unmarshal([]byte(contentJSON), &item); err != nil {
			return nil, nil, false, fmt.Errorf("unmarshal timeline item: %w", err)
		}
		items = append(items, item)
	}

	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}

	var nextCursor *string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		c := last.CreatedAt.Format("2006-01-02 15:04:05.999999999")
		nextCursor = &c
	}

	return items, nextCursor, hasMore, nil
}

func (r *timelineMySQLRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE timeline_items SET status=? WHERE id=?`,
		status, id,
	)
	return err
}
