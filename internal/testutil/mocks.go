// Package testutil provides shared test helpers and mock implementations.
package testutil

import (
	"context"
	"fmt"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/repository"
)

// ErrNotImplemented is returned by mock methods that have not been configured.
var ErrNotImplemented = fmt.Errorf("mock: method not implemented")

// Compile-time interface checks.
var (
	_ repository.PatientRepository  = (*MockPatientRepo)(nil)
	_ repository.VisitRepository    = (*MockVisitRepo)(nil)
	_ repository.TimelineRepository = (*MockTimelineRepo)(nil)
	_ repository.FlowCardRepository = (*MockFlowCardRepo)(nil)
)

// MockPatientRepo is a shared mock implementation of PatientRepository.
type MockPatientRepo struct {
	FindByCredentialFunc func(ctx context.Context, credType, credential string) (*model.PatientProfile, error)
	FindByIDFunc         func(ctx context.Context, id string) (*model.PatientProfile, error)
	CreateFunc           func(ctx context.Context, patient *model.PatientProfile) error
	UpdateProfileFunc    func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error)
}

func (m *MockPatientRepo) FindByCredential(ctx context.Context, credType, credential string) (*model.PatientProfile, error) {
	if m.FindByCredentialFunc != nil {
		return m.FindByCredentialFunc(ctx, credType, credential)
	}
	return nil, ErrNotImplemented
}
func (m *MockPatientRepo) FindByID(ctx context.Context, id string) (*model.PatientProfile, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, ErrNotImplemented
}
func (m *MockPatientRepo) Create(ctx context.Context, patient *model.PatientProfile) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, patient)
	}
	return ErrNotImplemented
}
func (m *MockPatientRepo) UpdateProfile(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
	if m.UpdateProfileFunc != nil {
		return m.UpdateProfileFunc(ctx, id, input)
	}
	return nil, ErrNotImplemented
}

// MockVisitRepo is a shared mock implementation of VisitRepository.
type MockVisitRepo struct {
	CreateFunc        func(ctx context.Context, v *model.VisitSession) error
	FindByIDFunc      func(ctx context.Context, id string) (*model.VisitSession, error)
	ListByPatientFunc func(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error)
	UpdateStatusFunc  func(ctx context.Context, id, status, machineState string) error
	UpdateFunc        func(ctx context.Context, v *model.VisitSession) error
}

func (m *MockVisitRepo) Create(ctx context.Context, v *model.VisitSession) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, v)
	}
	return ErrNotImplemented
}
func (m *MockVisitRepo) FindByID(ctx context.Context, id string) (*model.VisitSession, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, ErrNotImplemented
}
func (m *MockVisitRepo) ListByPatient(ctx context.Context, patientID string, cursor *string, pageSize int) ([]model.VisitSessionSummary, *string, bool, error) {
	if m.ListByPatientFunc != nil {
		return m.ListByPatientFunc(ctx, patientID, cursor, pageSize)
	}
	return nil, nil, false, ErrNotImplemented
}
func (m *MockVisitRepo) UpdateStatus(ctx context.Context, id, status, machineState string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status, machineState)
	}
	return ErrNotImplemented
}
func (m *MockVisitRepo) Update(ctx context.Context, v *model.VisitSession) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, v)
	}
	return ErrNotImplemented
}

// MockTimelineRepo is a shared mock implementation of TimelineRepository.
type MockTimelineRepo struct {
	AppendFunc                   func(ctx context.Context, item *model.TimelineItem) error
	AppendBatchFunc              func(ctx context.Context, items []model.TimelineItem) error
	ListBySessionFunc            func(ctx context.Context, sid string, cursor *string, ps int) ([]model.TimelineItem, *string, bool, error)
	FindLastPatientMessageFunc   func(ctx context.Context, sessionID string) (string, error)
	FindLastStreamingMessageFunc func(ctx context.Context, sessionID string) (*model.TimelineItem, error)
	UpdateStatusFunc             func(ctx context.Context, id, status string) error
	UpdateContentFunc            func(ctx context.Context, id string, item *model.TimelineItem) error
}

func (m *MockTimelineRepo) Append(ctx context.Context, item *model.TimelineItem) error {
	if m.AppendFunc != nil {
		return m.AppendFunc(ctx, item)
	}
	return ErrNotImplemented
}
func (m *MockTimelineRepo) AppendBatch(ctx context.Context, items []model.TimelineItem) error {
	if m.AppendBatchFunc != nil {
		return m.AppendBatchFunc(ctx, items)
	}
	return ErrNotImplemented
}
func (m *MockTimelineRepo) ListBySession(ctx context.Context, sid string, cursor *string, ps int) ([]model.TimelineItem, *string, bool, error) {
	if m.ListBySessionFunc != nil {
		return m.ListBySessionFunc(ctx, sid, cursor, ps)
	}
	return nil, nil, false, ErrNotImplemented
}
func (m *MockTimelineRepo) FindLastPatientMessage(ctx context.Context, sessionID string) (string, error) {
	if m.FindLastPatientMessageFunc != nil {
		return m.FindLastPatientMessageFunc(ctx, sessionID)
	}
	return "", ErrNotImplemented
}
func (m *MockTimelineRepo) FindLastStreamingMessage(ctx context.Context, sessionID string) (*model.TimelineItem, error) {
	if m.FindLastStreamingMessageFunc != nil {
		return m.FindLastStreamingMessageFunc(ctx, sessionID)
	}
	return nil, ErrNotImplemented
}
func (m *MockTimelineRepo) UpdateStatus(ctx context.Context, id, status string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return ErrNotImplemented
}
func (m *MockTimelineRepo) UpdateContent(ctx context.Context, id string, item *model.TimelineItem) error {
	if m.UpdateContentFunc != nil {
		return m.UpdateContentFunc(ctx, id, item)
	}
	return ErrNotImplemented
}

// MockFlowCardRepo is a shared mock implementation of FlowCardRepository.
type MockFlowCardRepo struct {
	CreateFunc        func(ctx context.Context, card *model.FlowCard) error
	FindByIDFunc      func(ctx context.Context, id string) (*model.FlowCard, error)
	ListBySessionFunc func(ctx context.Context, sessionID string) ([]model.FlowCard, error)
	UpdateStatusFunc  func(ctx context.Context, id, status string) error
	UpdateFunc        func(ctx context.Context, card *model.FlowCard) error
}

func (m *MockFlowCardRepo) Create(ctx context.Context, card *model.FlowCard) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, card)
	}
	return ErrNotImplemented
}
func (m *MockFlowCardRepo) FindByID(ctx context.Context, id string) (*model.FlowCard, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, ErrNotImplemented
}
func (m *MockFlowCardRepo) ListBySession(ctx context.Context, sessionID string) ([]model.FlowCard, error) {
	if m.ListBySessionFunc != nil {
		return m.ListBySessionFunc(ctx, sessionID)
	}
	return nil, ErrNotImplemented
}
func (m *MockFlowCardRepo) UpdateStatus(ctx context.Context, id, status string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return ErrNotImplemented
}
func (m *MockFlowCardRepo) Update(ctx context.Context, card *model.FlowCard) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, card)
	}
	return ErrNotImplemented
}
