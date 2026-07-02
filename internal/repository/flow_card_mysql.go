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

	touchTimestamps(&card.CreatedAt, nil)

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
	// Unmarshal into a separate variable first to avoid polluting column-scanned values
	card := &model.FlowCard{}

	// Save all column values before unmarshaling the JSON blob
	var (
		dbID, dbSessionID, dbKind, dbStatus, dbTitle string
		dbBlocking                                   bool
		dbCreatedAt                                  time.Time
		dbHandledAt                                  sql.NullTime
	)

	err := r.db.QueryRowContext(ctx,
		`SELECT id, session_id, kind, status, blocking, title, content, lock_reason, created_at, handled_at
		FROM flow_cards WHERE id = ?`, id,
	).Scan(&dbID, &dbSessionID, &dbKind, &dbStatus,
		&dbBlocking, &dbTitle, &contentJSON, &lockReason,
		&dbCreatedAt, &dbHandledAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrCardNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find flow card by id: %w", err)
	}

	// Unmarshal non-column fields from JSON
	if err := json.Unmarshal([]byte(contentJSON), card); err != nil {
		return nil, fmt.Errorf("unmarshal flow card: %w", err)
	}

	// Restore all column values — the content JSON may carry stale values
	card.ID = dbID
	card.SessionID = dbSessionID
	card.Kind = dbKind
	card.Status = dbStatus
	card.Blocking = dbBlocking
	card.Title = dbTitle
	card.CreatedAt = dbCreatedAt
	if dbHandledAt.Valid {
		card.HandledAt = &dbHandledAt.Time
	}
	card.LockReason = &lockReason

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
		`UPDATE flow_cards SET status=?, blocking=?, content=?, handled_at=? WHERE id=?`,
		card.Status, card.Blocking, string(contentJSON), card.HandledAt, card.ID,
	)
	if err != nil {
		return fmt.Errorf("update flow card: %w", err)
	}
	return nil
}
