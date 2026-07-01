package workbench

import (
	"context"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// PauseTimer pauses the visit total timer.
func (s *Service) PauseTimer(ctx context.Context, sessionID string) (*model.VisitSession, error) {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session.TimerPaused = true
	session.PausedAt = &now
	session.UpdatedAt = now
	session.LastActivityAt = &now
	_ = s.visitRepo.Update(ctx, session)

	return session, nil
}

// ResumeTimer resumes the visit total timer.
func (s *Service) ResumeTimer(ctx context.Context, sessionID string) (*model.VisitSession, error) {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	session.TimerPaused = false
	session.PausedAt = nil
	now := time.Now()
	session.UpdatedAt = now
	session.LastActivityAt = &now
	_ = s.visitRepo.Update(ctx, session)

	return session, nil
}
