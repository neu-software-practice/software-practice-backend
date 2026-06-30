package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type settingsMySQLRepo struct {
	db *sql.DB
}

// NewSettingsRepository creates a new MySQL-based SettingsRepository.
func NewSettingsRepository(db *sql.DB) SettingsRepository {
	return &settingsMySQLRepo{db: db}
}

func (r *settingsMySQLRepo) Get(ctx context.Context) (*model.SystemSettings, error) {
	var s model.SystemSettings
	err := r.db.QueryRowContext(ctx,
		`SELECT site_name, max_concurrent_sessions, session_timeout_minutes, enable_registration
		FROM system_settings WHERE id = 1`,
	).Scan(&s.SiteName, &s.MaxConcurrentSessions, &s.SessionTimeoutMinutes, &s.EnableRegistration)
	if err == sql.ErrNoRows {
		return &model.SystemSettings{
			SiteName:              "NEUHIS Agent",
			MaxConcurrentSessions: 3,
			SessionTimeoutMinutes: 30,
			EnableRegistration:    true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get system settings: %w", err)
	}
	return &s, nil
}

func (r *settingsMySQLRepo) Update(ctx context.Context, input model.UpdateSystemSettingsInput) (*model.SystemSettings, error) {
	current, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}

	if input.SiteName != nil {
		current.SiteName = *input.SiteName
	}
	if input.MaxConcurrentSessions != nil {
		current.MaxConcurrentSessions = *input.MaxConcurrentSessions
	}
	if input.SessionTimeoutMinutes != nil {
		current.SessionTimeoutMinutes = *input.SessionTimeoutMinutes
	}
	if input.EnableRegistration != nil {
		current.EnableRegistration = *input.EnableRegistration
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO system_settings (id, site_name, max_concurrent_sessions, session_timeout_minutes, enable_registration)
		VALUES (1, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			site_name = VALUES(site_name),
			max_concurrent_sessions = VALUES(max_concurrent_sessions),
			session_timeout_minutes = VALUES(session_timeout_minutes),
			enable_registration = VALUES(enable_registration)`,
		current.SiteName, current.MaxConcurrentSessions, current.SessionTimeoutMinutes, current.EnableRegistration,
	)
	if err != nil {
		return nil, fmt.Errorf("update system settings: %w", err)
	}

	return current, nil
}
