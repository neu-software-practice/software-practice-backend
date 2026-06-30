package workbench_test

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
	"github.com/neuhis/software-practice-backend/internal/model"
	wbsvc "github.com/neuhis/software-practice-backend/internal/service/workbench"
)

// mockLLMClient implements wbsvc.LLMClient for testing.
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) ChatComplete(_ context.Context, _, _ string) (string, error) {
	return m.response, m.err
}

func newTestServiceWithLLM(
	visitRepo *mockVisitRepo,
	timelineRepo *mockTimelineRepo,
	llmClient wbsvc.LLMClient,
) *wbsvc.Service {
	return wbsvc.NewService(
		&mockPatientRepo{},
		visitRepo,
		timelineRepo,
		&mockFlowCardRepo{},
		&mockAddressRepo{},
		nil,
		"http",
		llmClient,
	)
}

func TestGenerateTitle_SessionNotFound(t *testing.T) {
	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return nil, model.ErrSessionNotFound
		},
	}
	tr := &mockTimelineRepo{}
	svc := newTestServiceWithLLM(vr, tr, &mockLLMClient{})

	_, err := svc.GenerateTitle(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *apperrors.ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ApiError, got %T: %v", err, err)
	}
	if apiErr.Code != apperrors.CodeSessionNotFound {
		t.Errorf("expected code %s, got %s", apperrors.CodeSessionNotFound, apiErr.Code)
	}
}

func TestGenerateTitle_AlreadyHasTitle(t *testing.T) {
	existingTitle := "已有标题"
	session := makeSession("p001")
	session.Summary.Title = &existingTitle

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
	}
	tr := &mockTimelineRepo{}
	svc := newTestServiceWithLLM(vr, tr, &mockLLMClient{})

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != existingTitle {
		t.Errorf("got %q, want %q", title, existingTitle)
	}
}

func TestGenerateTitle_LLMSuccess(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "我头痛三天了"},
				{Kind: "message", Role: "assistant", Content: "您好，请问头痛是持续性的还是间歇性的？"},
			}, nil, false, nil
		},
	}
	llm := &mockLLMClient{response: "头痛三天"}
	svc := newTestServiceWithLLM(vr, tr, llm)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "头痛三天" {
		t.Errorf("got %q, want %q", title, "头痛三天")
	}
}

func TestGenerateTitle_LLMFails_FallbackToChiefComplaint(t *testing.T) {
	cc := "我发烧了三天还一直咳嗽"
	session := makeSession("p001")
	session.Summary.ChiefComplaint = &cc

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "我发烧了三天还一直咳嗽"},
			}, nil, false, nil
		},
	}
	llm := &mockLLMClient{err: errors.New("connection refused")}
	svc := newTestServiceWithLLM(vr, tr, llm)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fallback to chiefComplaint truncated to 15 chars
	if len([]rune(title)) > 15 {
		t.Errorf("fallback title too long: %q (%d runes)", title, len([]rune(title)))
	}
}

func TestGenerateTitle_DiagnosisPriority(t *testing.T) {
	diag := "上呼吸道感染"
	session := makeSession("p001")
	session.Summary.Diagnosis = &diag

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "我感冒了"},
				{Kind: "message", Role: "assistant", Content: "诊断为上呼吸道感染"},
			}, nil, false, nil
		},
	}
	llm := &mockLLMClient{response: "上呼吸道感染"}
	svc := newTestServiceWithLLM(vr, tr, llm)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "上呼吸道感染" {
		t.Errorf("got %q, want %q", title, "上呼吸道感染")
	}
}

func TestGenerateTitle_TitleTrimAndTruncate(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "测试"},
			}, nil, false, nil
		},
	}
	// LLM returns a title with trailing punctuation and leading/trailing spaces
	llm := &mockLLMClient{response: "  发热伴咳嗽三天。  "}
	svc := newTestServiceWithLLM(vr, tr, llm)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be trimmed and punctuation removed
	if title != "发热伴咳嗽三天" {
		t.Errorf("got %q, want %q", title, "发热伴咳嗽三天")
	}
}

func TestGenerateTitle_NoTimeline_FallbackEmpty(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{}, nil, false, nil
		},
	}
	// LLM will fail on empty input
	llm := &mockLLMClient{err: errors.New("empty input")}
	svc := newTestServiceWithLLM(vr, tr, llm)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return fallback "问诊记录"
	if title != "问诊记录" {
		t.Errorf("got %q, want %q", title, "问诊记录")
	}
}

func TestGenerateTitle_FallbackLongChiefComplaint(t *testing.T) {
	longCC := "我已经头痛了很久很久了并且感觉非常不舒服"
	session := makeSession("p001")
	session.Summary.ChiefComplaint = &longCC

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: longCC},
			}, nil, false, nil
		},
	}
	// LLM fails, fallback to chiefComplaint truncation
	llmClient := &mockLLMClient{err: errors.New("unavailable")}
	svc := newTestServiceWithLLM(vr, tr, llmClient)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be truncated to 13 runes + "…"
	runes := []rune(title)
	if len(runes) > 15 {
		t.Errorf("fallback title too long: %q (%d runes)", title, len(runes))
	}
}

func TestGenerateTitle_FallbackShortChiefComplaint(t *testing.T) {
	shortCC := "头痛三天"
	session := makeSession("p001")
	session.Summary.ChiefComplaint = &shortCC

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: shortCC},
			}, nil, false, nil
		},
	}
	// LLM fails, fallback to chiefComplaint directly (<=15 chars)
	llmClient := &mockLLMClient{err: errors.New("unavailable")}
	svc := newTestServiceWithLLM(vr, tr, llmClient)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != shortCC {
		t.Errorf("got %q, want %q", title, shortCC)
	}
}

func TestGenerateTitle_SanitizeQuotedTitle(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "我发烧了"},
			}, nil, false, nil
		},
	}
	// LLM returns quoted title
	llmClient := &mockLLMClient{response: `"发热问诊"`}
	svc := newTestServiceWithLLM(vr, tr, llmClient)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "发热问诊" {
		t.Errorf("got %q, want %q", title, "发热问诊")
	}
}

func TestGenerateTitle_SanitizeLongTitle(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "我发烧了"},
			}, nil, false, nil
		},
	}
	// LLM returns very long title (>50 chars)
	longTitle := "这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的标题超过五十个字符"
	llmClient := &mockLLMClient{response: longTitle}
	svc := newTestServiceWithLLM(vr, tr, llmClient)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	runes := []rune(title)
	if len(runes) > 50 {
		t.Errorf("title too long: %d runes, want <=50", len(runes))
	}
}

func TestGenerateTitle_SanitizeChineseQuotes(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "头痛"},
			}, nil, false, nil
		},
	}
	// LLM returns Chinese-quoted title
	llmClient := &mockLLMClient{response: "“头痛三天”"}
	svc := newTestServiceWithLLM(vr, tr, llmClient)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "头痛三天" {
		t.Errorf("got %q, want %q", title, "头痛三天")
	}
}

func TestGenerateTitle_TimelineError_FallsBackGracefully(t *testing.T) {
	session := makeSession("p001")
	cc := "头疼三天了"
	session.Summary.ChiefComplaint = &cc

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return nil
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return nil, nil, false, errors.New("db error")
		},
	}
	// Timeline fails -> buildTitleContext returns chiefComplaint -> LLM called with it -> returns title
	llmClient := &mockLLMClient{response: "头疼三天"}
	svc := newTestServiceWithLLM(vr, tr, llmClient)

	title, err := svc.GenerateTitle(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "头疼三天" {
		t.Errorf("got %q, want %q", title, "头疼三天")
	}
}

func TestGenerateTitle_UpdateRepoError(t *testing.T) {
	session := makeSession("p001")

	vr := &mockVisitRepo{
		findByIDFunc: func(_ context.Context, _ string) (*model.VisitSession, error) {
			return session, nil
		},
		updateFunc: func(_ context.Context, _ *model.VisitSession) error {
			return errors.New("db error")
		},
	}
	tr := &mockTimelineRepo{
		listFunc: func(_ context.Context, _ string, _ *string, _ int) ([]model.TimelineItem, *string, bool, error) {
			return []model.TimelineItem{
				{Kind: "message", Role: "patient", Content: "头痛"},
			}, nil, false, nil
		},
	}
	llm := &mockLLMClient{response: "头痛"}
	svc := newTestServiceWithLLM(vr, tr, llm)

	_, err := svc.GenerateTitle(context.Background(), session.ID)
	if err == nil {
		t.Fatal("expected error from update")
	}
}
