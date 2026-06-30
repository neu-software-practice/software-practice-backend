package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type flowCardMySQLRepo struct {
	db *sql.DB
}

// NewFlowCardRepository creates a new MySQL-based FlowCardRepository.
func NewFlowCardRepository(db *sql.DB) FlowCardRepository {
	return &flowCardMySQLRepo{db: db}
}

func (r *flowCardMySQLRepo) Create(ctx context.Context, card *model.FlowCard) error {
	contentJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal flow card: %w", err)
	}

	now := time.Now()
	if card.CreatedAt.IsZero() {
		card.CreatedAt = now
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO flow_cards (id, session_id, kind, status, blocking, title, content, lock_reason, created_at, handled_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		card.ID, card.SessionID, card.Kind, card.Status,
		card.Blocking, card.Title, string(contentJSON),
		"", card.CreatedAt, card.HandledAt,
	)
	if err != nil {
		return fmt.Errorf("create flow card: %w", err)
	}
	return nil
}

func (r *flowCardMySQLRepo) FindByID(ctx context.Context, id string) (*model.FlowCard, error) {
	var contentJSON, lockReason string
	card := &model.FlowCard{}

	err := r.db.QueryRowContext(ctx,
		`SELECT id, session_id, kind, status, blocking, title, content, lock_reason, created_at, handled_at
		FROM flow_cards WHERE id = ?`, id,
	).Scan(&card.ID, &card.SessionID, &card.Kind, &card.Status,
		&card.Blocking, &card.Title, &contentJSON, &lockReason,
		&card.CreatedAt, &card.HandledAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrCardNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find flow card by id: %w", err)
	}

	// Save DB column values that may be overwritten by the JSON content.
	dbStatus := card.Status

	if err := json.Unmarshal([]byte(contentJSON), card); err != nil {
		return nil, fmt.Errorf("unmarshal flow card: %w", err)
	}
	card.LockReason = &lockReason
	// Restore the DB column status (the content JSON may carry a stale value).
	card.Status = dbStatus

	return card, nil
}

func (r *flowCardMySQLRepo) ListBySession(ctx context.Context, sessionID string) ([]model.FlowCard, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT content, status FROM flow_cards WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list flow cards: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var cards []model.FlowCard
	for rows.Next() {
		var contentJSON string
		var dbStatus string
		if err := rows.Scan(&contentJSON, &dbStatus); err != nil {
			return nil, fmt.Errorf("scan flow card: %w", err)
		}
		var card model.FlowCard
		if err := json.Unmarshal([]byte(contentJSON), &card); err != nil {
			return nil, fmt.Errorf("unmarshal flow card: %w", err)
		}
		// Restore the DB column status (the content JSON may carry a stale value).
		card.Status = dbStatus
		cards = append(cards, card)
	}
	return cards, nil
}

func (r *flowCardMySQLRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE flow_cards SET status=?, handled_at=? WHERE id=?`,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("update flow card status: %w", err)
	}
	return nil
}

func (r *flowCardMySQLRepo) Update(ctx context.Context, card *model.FlowCard) error {
	contentJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal flow card: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE flow_cards SET status=?, content=?, handled_at=? WHERE id=?`,
		card.Status, string(contentJSON), card.HandledAt, card.ID,
	)
	if err != nil {
		return fmt.Errorf("update flow card: %w", err)
	}
	return nil
}
