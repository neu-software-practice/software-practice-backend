package repository

import (
	"context"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// DashboardRepository defines the data access interface for admin dashboard and listing queries.
type DashboardRepository interface {
	// Dashboard statistics
	CountPatients(ctx context.Context) (int, error)
	CountPatientsSince(ctx context.Context, since time.Time) (int, error)
	CountSessions(ctx context.Context) (int, error)
	CountActiveSessions(ctx context.Context) (int, error)
	CountSessionsSince(ctx context.Context, since time.Time) (int, error)

	// Patient listing
	ListPatients(ctx context.Context, query model.AdminPatientQuery) ([]model.AdminPatientItem, int, error)

	// Session listing
	ListSessions(ctx context.Context, query model.AdminSessionQuery) ([]model.AdminSessionItem, int, error)
}
