package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type refreshTokenMySQLRepo struct {
	db *sql.DB
}

// NewRefreshTokenRepository creates a new MySQL-based RefreshTokenRepository.
func NewRefreshTokenRepository(db *sql.DB) RefreshTokenRepository {
	return &refreshTokenMySQLRepo{db: db}
}

func (r *refreshTokenMySQLRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	token.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, token_hash, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		token.ID, token.TokenHash, token.UserID, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenMySQLRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	var usedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, token_hash, user_id, expires_at, used_at, created_at
		FROM refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&t.ID, &t.TokenHash, &t.UserID, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrRefreshTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh token by hash: %w", err)
	}

	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return &t, nil
}

func (r *refreshTokenMySQLRepo) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET used_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("mark refresh token used: %w", err)
	}
	return nil
}

func (r *refreshTokenMySQLRepo) RevokeAllByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM refresh_tokens WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return nil
}
