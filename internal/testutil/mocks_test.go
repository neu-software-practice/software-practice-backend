package testutil_test

import (
	"context"
	"testing"

	"github.com/neuhis/software-practice-backend/internal/model"
	"github.com/neuhis/software-practice-backend/internal/testutil"
)

// TestAllMocks_CoverAllMethods exercises every mock method twice:
// once with nil func (default ErrNotImplemented) and once with a configured func.

func TestMockPatientRepo_AllMethods(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := &testutil.MockPatientRepo{}
		if _, err := r.FindByCredential(context.Background(), "id_card", "x"); err != testutil.ErrNotImplemented {
			t.Errorf("FindByCredential: %v", err)
		}
		if _, err := r.FindByID(context.Background(), "x"); err != testutil.ErrNotImplemented {
			t.Errorf("FindByID: %v", err)
		}
		if err := r.Create(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("Create: %v", err)
		}
		if _, err := r.UpdateProfile(context.Background(), "x", model.ProfileUpdateInput{}); err != testutil.ErrNotImplemented {
			t.Errorf("UpdateProfile: %v", err)
		}
	})
	t.Run("configured", func(t *testing.T) {
		r := &testutil.MockPatientRepo{
			FindByCredentialFunc: func(ctx context.Context, ct, c string) (*model.PatientProfile, error) {
				return &model.PatientProfile{ID: "p1"}, nil
			},
			FindByIDFunc: func(ctx context.Context, id string) (*model.PatientProfile, error) {
				return &model.PatientProfile{ID: id}, nil
			},
			CreateFunc: func(ctx context.Context, p *model.PatientProfile) error { return nil },
			UpdateProfileFunc: func(ctx context.Context, id string, input model.ProfileUpdateInput) (*model.PatientProfile, error) {
				return &model.PatientProfile{ID: id}, nil
			},
		}
		if p, _ := r.FindByCredential(context.Background(), "id_card", "x"); p.ID != "p1" {
			t.Error("FindByCredential")
		}
		if p, _ := r.FindByID(context.Background(), "p1"); p.ID != "p1" {
			t.Error("FindByID")
		}
		if err := r.Create(context.Background(), nil); err != nil {
			t.Errorf("Create: %v", err)
		}
		if p, _ := r.UpdateProfile(context.Background(), "x", model.ProfileUpdateInput{}); p.ID != "x" {
			t.Error("UpdateProfile")
		}
	})
}

func TestMockVisitRepo_AllMethods(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := &testutil.MockVisitRepo{}
		if err := r.Create(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("Create: %v", err)
		}
		if _, err := r.FindByID(context.Background(), "x"); err != testutil.ErrNotImplemented {
			t.Errorf("FindByID: %v", err)
		}
		if _, _, _, err := r.ListByPatient(context.Background(), "x", nil, 20); err != testutil.ErrNotImplemented {
			t.Errorf("ListByPatient: %v", err)
		}
		if err := r.UpdateStatus(context.Background(), "x", "s", "ms"); err != testutil.ErrNotImplemented {
			t.Errorf("UpdateStatus: %v", err)
		}
		if err := r.Update(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("Update: %v", err)
		}
	})
	t.Run("configured", func(t *testing.T) {
		r := &testutil.MockVisitRepo{
			CreateFunc: func(ctx context.Context, v *model.VisitSession) error { return nil },
			FindByIDFunc: func(ctx context.Context, id string) (*model.VisitSession, error) {
				return &model.VisitSession{ID: id}, nil
			},
			ListByPatientFunc: func(ctx context.Context, pid string, c *string, ps int) ([]model.VisitSessionSummary, *string, bool, error) {
				return nil, nil, false, nil
			},
			UpdateStatusFunc: func(ctx context.Context, id, s, ms string) error { return nil },
			UpdateFunc:       func(ctx context.Context, v *model.VisitSession) error { return nil },
		}
		if err := r.Create(context.Background(), nil); err != nil {
			t.Error("Create")
		}
		if v, _ := r.FindByID(context.Background(), "v1"); v.ID != "v1" {
			t.Error("FindByID")
		}
		if _, _, _, err := r.ListByPatient(context.Background(), "x", nil, 20); err != nil {
			t.Error("ListByPatient")
		}
		if err := r.UpdateStatus(context.Background(), "x", "s", "ms"); err != nil {
			t.Error("UpdateStatus")
		}
		if err := r.Update(context.Background(), nil); err != nil {
			t.Error("Update")
		}
	})
}

func TestMockTimelineRepo_AllMethods(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := &testutil.MockTimelineRepo{}
		if err := r.Append(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("Append: %v", err)
		}
		if err := r.AppendBatch(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("AppendBatch: %v", err)
		}
		if _, _, _, err := r.ListBySession(context.Background(), "x", nil, 20); err != testutil.ErrNotImplemented {
			t.Errorf("ListBySession: %v", err)
		}
		if _, err := r.FindLastPatientMessage(context.Background(), "x"); err != testutil.ErrNotImplemented {
			t.Errorf("FindLastPatientMessage: %v", err)
		}
		if err := r.UpdateStatus(context.Background(), "x", "s"); err != testutil.ErrNotImplemented {
			t.Errorf("UpdateStatus: %v", err)
		}
	})
	t.Run("configured", func(t *testing.T) {
		r := &testutil.MockTimelineRepo{
			AppendFunc:      func(ctx context.Context, item *model.TimelineItem) error { return nil },
			AppendBatchFunc: func(ctx context.Context, items []model.TimelineItem) error { return nil },
			ListBySessionFunc: func(ctx context.Context, sid string, c *string, ps int) ([]model.TimelineItem, *string, bool, error) {
				return nil, nil, false, nil
			},
			FindLastPatientMessageFunc: func(ctx context.Context, sid string) (string, error) { return "hello", nil },
			UpdateStatusFunc:           func(ctx context.Context, id, s string) error { return nil },
		}
		if err := r.Append(context.Background(), nil); err != nil {
			t.Error("Append")
		}
		if err := r.AppendBatch(context.Background(), nil); err != nil {
			t.Error("AppendBatch")
		}
		if _, _, _, err := r.ListBySession(context.Background(), "x", nil, 20); err != nil {
			t.Error("ListBySession")
		}
		if msg, _ := r.FindLastPatientMessage(context.Background(), "x"); msg != "hello" {
			t.Error("FindLastPatientMessage")
		}
		if err := r.UpdateStatus(context.Background(), "x", "s"); err != nil {
			t.Error("UpdateStatus")
		}
	})
}

func TestMockFlowCardRepo_AllMethods(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := &testutil.MockFlowCardRepo{}
		if err := r.Create(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("Create: %v", err)
		}
		if _, err := r.FindByID(context.Background(), "x"); err != testutil.ErrNotImplemented {
			t.Errorf("FindByID: %v", err)
		}
		if _, err := r.ListBySession(context.Background(), "x"); err != testutil.ErrNotImplemented {
			t.Errorf("ListBySession: %v", err)
		}
		if err := r.UpdateStatus(context.Background(), "x", "s"); err != testutil.ErrNotImplemented {
			t.Errorf("UpdateStatus: %v", err)
		}
		if err := r.Update(context.Background(), nil); err != testutil.ErrNotImplemented {
			t.Errorf("Update: %v", err)
		}
	})
	t.Run("configured", func(t *testing.T) {
		r := &testutil.MockFlowCardRepo{
			CreateFunc: func(ctx context.Context, c *model.FlowCard) error { return nil },
			FindByIDFunc: func(ctx context.Context, id string) (*model.FlowCard, error) {
				return &model.FlowCard{ID: id}, nil
			},
			ListBySessionFunc: func(ctx context.Context, sid string) ([]model.FlowCard, error) { return nil, nil },
			UpdateStatusFunc:  func(ctx context.Context, id, s string) error { return nil },
			UpdateFunc:        func(ctx context.Context, c *model.FlowCard) error { return nil },
		}
		if err := r.Create(context.Background(), nil); err != nil {
			t.Error("Create")
		}
		if c, _ := r.FindByID(context.Background(), "c1"); c.ID != "c1" {
			t.Error("FindByID")
		}
		if _, err := r.ListBySession(context.Background(), "x"); err != nil {
			t.Error("ListBySession")
		}
		if err := r.UpdateStatus(context.Background(), "x", "s"); err != nil {
			t.Error("UpdateStatus")
		}
		if err := r.Update(context.Background(), nil); err != nil {
			t.Error("Update")
		}
	})
}

func TestErrNotImplemented(t *testing.T) {
	if testutil.ErrNotImplemented.Error() != "mock: method not implemented" {
		t.Errorf("unexpected error message: %v", testutil.ErrNotImplemented)
	}
}
