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
	s.visitRepo.Update(ctx, session)

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
	s.visitRepo.Update(ctx, session)

	return session, nil
}
