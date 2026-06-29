package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type userMySQLRepo struct {
	db *sql.DB
}

// NewUserRepository creates a new MySQL-based UserRepository.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userMySQLRepo{db: db}
}

func (r *userMySQLRepo) Create(ctx context.Context, user *model.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, phone, password_hash, real_name, patient_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Phone, user.PasswordHash, user.RealName, user.PatientID,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userMySQLRepo) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, phone, password_hash, real_name, patient_id, created_at, updated_at
		FROM users WHERE phone = ?`, phone,
	).Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.RealName, &u.PatientID, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by phone: %w", err)
	}
	return &u, nil
}

func (r *userMySQLRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, phone, password_hash, real_name, patient_id, created_at, updated_at
		FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.RealName, &u.PatientID, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &u, nil
}
