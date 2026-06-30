package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type adminRefreshTokenMySQLRepo struct {
	db *sql.DB
}

// NewAdminRefreshTokenRepository creates a new MySQL-based AdminRefreshTokenRepository.
func NewAdminRefreshTokenRepository(db *sql.DB) AdminRefreshTokenRepository {
	return &adminRefreshTokenMySQLRepo{db: db}
}

func (r *adminRefreshTokenMySQLRepo) Create(ctx context.Context, token *model.AdminRefreshToken) error {
	token.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO admin_refresh_tokens (id, token_hash, admin_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		token.ID, token.TokenHash, token.AdminID, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create admin refresh token: %w", err)
	}
	return nil
}

func (r *adminRefreshTokenMySQLRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.AdminRefreshToken, error) {
	var t model.AdminRefreshToken
	var usedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, token_hash, admin_id, expires_at, used_at, created_at
		FROM admin_refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&t.ID, &t.TokenHash, &t.AdminID, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrAdminInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("find admin refresh token by hash: %w", err)
	}

	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return &t, nil
}

func (r *adminRefreshTokenMySQLRepo) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admin_refresh_tokens SET used_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("mark admin refresh token used: %w", err)
	}
	return nil
}

func (r *adminRefreshTokenMySQLRepo) RevokeAllByAdminID(ctx context.Context, adminID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM admin_refresh_tokens WHERE admin_id = ?`,
		adminID,
	)
	if err != nil {
		return fmt.Errorf("revoke all admin refresh tokens: %w", err)
	}
	return nil
}
