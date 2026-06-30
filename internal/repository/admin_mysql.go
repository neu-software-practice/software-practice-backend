package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type adminMySQLRepo struct {
	db *sql.DB
}

// NewAdminRepository creates a new MySQL-based AdminRepository.
func NewAdminRepository(db *sql.DB) AdminRepository {
	return &adminMySQLRepo{db: db}
}

func (r *adminMySQLRepo) FindByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var u model.AdminUser
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, display_name, created_at
		FROM admin_users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.DisplayName, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrAdminNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find admin by username: %w", err)
	}
	return &u, nil
}

func (r *adminMySQLRepo) FindByID(ctx context.Context, id string) (*model.AdminUser, error) {
	var u model.AdminUser
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, display_name, created_at
		FROM admin_users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.DisplayName, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrAdminNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find admin by id: %w", err)
	}
	return &u, nil
}

func (r *adminMySQLRepo) Create(ctx context.Context, admin *model.AdminUser) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO admin_users (id, username, password_hash, role, display_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		admin.ID, admin.Username, admin.PasswordHash, admin.Role, admin.DisplayName, admin.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	return nil
}
