package workbench

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

const (
	followUpSummaryPrompt = `你是医疗复诊上下文摘要助手。请根据上次问诊资料，生成给下一次复诊医生使用的中文摘要。
要求：
1. 覆盖主诉、关键追问/回答、检查/处置、诊断、用药/治疗、复诊注意点。
2. 不编造资料中没有的信息；缺失项写“未记录”。
3. 控制在 600 字以内，使用紧凑条目。`
	maxPriorTimelineItems = 1000
	maxPriorSummaryRunes  = 1800
)

func (s *Service) buildPriorRecordsForFollowUp(ctx context.Context, session *model.VisitSession) ([]interface{}, error) {
	if session.EntryType != string(model.VisitEntryTypeFollowUp) || session.ParentSessionID == nil || *session.ParentSessionID == "" {
		return nil, nil
	}

	parent, err := s.visitRepo.FindByID(ctx, *session.ParentSessionID)
	if err != nil {
		return nil, fmt.Errorf("find parent session: %w", err)
	}
	if parent.PatientID != session.PatientID {
		return nil, model.ErrSessionNotFound
	}

	items, err := s.listTimelineForPriorSummary(ctx, parent.ID)
	if err != nil {
		return nil, fmt.Errorf("list parent timeline: %w", err)
	}

	summary := s.summarizeParentVisit(ctx, parent, items)
	outcome := buildPriorOutcome(parent, items)
	record := medagent.SessionRecord{
		SessionID: parent.ID,
		Initial:   parent.EntryType == string(model.VisitEntryTypeNew),
		StartedAt: parent.StartedAt,
		EndedAt:   parent.EndedAt,
		Turns: []medagent.RecordedTurn{{
			At:   parent.UpdatedAt,
			Kind: "doctor",
			Text: "【上次问诊摘要】\n" + summary,
		}},
		Outcome: outcome,
	}
	return []interface{}{record}, nil
}

func (s *Service) listTimelineForPriorSummary(ctx context.Context, sessionID string) ([]model.TimelineItem, error) {
	var all []model.TimelineItem
	var cursor *string
	for len(all) < maxPriorTimelineItems {
		items, nextCursor, hasMore, err := s.timelineRepo.ListBySession(ctx, sessionID, cursor, 100)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !hasMore || nextCursor == nil {
			break
		}
		cursor = nextCursor
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	if len(all) > maxPriorTimelineItems {
		all = all[:maxPriorTimelineItems]
	}
	return all, nil
}

func (s *Service) summarizeParentVisit(ctx context.Context, parent *model.VisitSession, items []model.TimelineItem) string {
	fallback := buildFallbackPriorSummary(parent, items)
	if s.llmClient == nil {
		return fallback
	}

	raw, err := s.llmClient.ChatComplete(ctx, followUpSummaryPrompt, buildPriorSummaryInput(parent, items))
	if err != nil {
		return fallback
	}
	summary := trimRunes(strings.TrimSpace(raw), maxPriorSummaryRunes)
	if summary == "" {
		return fallback
	}
	return summary
}

func buildPriorSummaryInput(parent *model.VisitSession, items []model.TimelineItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "会话ID：%s\n", parent.ID)
	fmt.Fprintf(&b, "状态：%s\n", parent.Status)
	fmt.Fprintf(&b, "开始时间：%s\n", parent.StartedAt.Format("2006-01-02 15:04:05"))
	if parent.EndedAt != nil {
		fmt.Fprintf(&b, "结束时间：%s\n", parent.EndedAt.Format("2006-01-02 15:04:05"))
	}
	writeVisitSummary(&b, parent.Summary)

	b.WriteString("\n时间线：\n")
	for _, item := range items {
		switch item.Kind {
		case "message":
			if item.Content != "" {
				fmt.Fprintf(&b, "- %s：%s\n", timelineRoleName(item.Role), item.Content)
			}
		case "flow_card":
			if item.Card != nil {
				writeFlowCardSummary(&b, item.Card)
			}
		}
	}
	return trimRunes(b.String(), 12000)
}

func buildFallbackPriorSummary(parent *model.VisitSession, items []model.TimelineItem) string {
	var b strings.Builder
	b.WriteString("上次问诊摘要：\n")
	writeVisitSummary(&b, parent.Summary)

	latestPatient := ""
	latestDoctor := ""
	for _, item := range items {
		if item.Kind != "message" || item.Content == "" {
			continue
		}
		switch item.Role {
		case "patient":
			latestPatient = item.Content
		case "assistant", "doctor":
			latestDoctor = item.Content
		}
	}
	if latestPatient != "" {
		fmt.Fprintf(&b, "最近患者描述：%s\n", latestPatient)
	}
	if latestDoctor != "" {
		fmt.Fprintf(&b, "最近医生回复：%s\n", latestDoctor)
	}
	for _, item := range items {
		if item.Kind == "flow_card" && item.Card != nil {
			writeFlowCardSummary(&b, item.Card)
		}
	}
	text := trimRunes(strings.TrimSpace(b.String()), maxPriorSummaryRunes)
	if text == "" {
		return "上次问诊无可用摘要。"
	}
	return text
}

func writeVisitSummary(b *strings.Builder, summary model.VisitSummary) {
	if summary.ChiefComplaint != nil && *summary.ChiefComplaint != "" {
		fmt.Fprintf(b, "主诉：%s\n", *summary.ChiefComplaint)
	}
	if summary.Diagnosis != nil && *summary.Diagnosis != "" {
		fmt.Fprintf(b, "诊断：%s\n", *summary.Diagnosis)
	}
	if summary.TreatmentSummary != nil && *summary.TreatmentSummary != "" {
		fmt.Fprintf(b, "治疗/处置：%s\n", *summary.TreatmentSummary)
	}
	if summary.LastMessage != nil && *summary.LastMessage != "" {
		fmt.Fprintf(b, "最后消息：%s\n", *summary.LastMessage)
	}
}

func writeFlowCardSummary(b *strings.Builder, card *model.FlowCard) {
	switch card.Kind {
	case string(model.FlowCardKindDiagnosis):
		if card.Diagnosis != "" {
			fmt.Fprintf(b, "- 诊断卡：%s", card.Diagnosis)
			if len(card.Evidence) > 0 {
				fmt.Fprintf(b, "，依据：%s", strings.Join(card.Evidence, "；"))
			}
			b.WriteString("\n")
		}
	case string(model.FlowCardKindTreatmentPlan):
		if card.Summary != "" || card.Plan != "" {
			fmt.Fprintf(b, "- 处置方案：%s %s\n", card.Plan, card.Summary)
		}
	case string(model.FlowCardKindMedicationFulfillment):
		if len(card.Medications) > 0 {
			names := make([]string, 0, len(card.Medications))
			for _, med := range card.Medications {
				names = append(names, strings.TrimSpace(med.Name+" "+med.Dosage))
			}
			fmt.Fprintf(b, "- 用药：%s\n", strings.Join(names, "；"))
		}
	case string(model.FlowCardKindCompletedVisit):
		if card.Diagnosis != "" || card.TreatmentSummary != "" || card.FollowUpSuggestion != "" {
			fmt.Fprintf(b, "- 就诊完成：诊断=%s；处置=%s；复诊建议=%s\n",
				card.Diagnosis, card.TreatmentSummary, card.FollowUpSuggestion)
		}
	case string(model.FlowCardKindAdviceOnly):
		if len(card.Advices) > 0 || card.FollowUpRecommendation != "" {
			fmt.Fprintf(b, "- 医嘱：%s %s\n", strings.Join(card.Advices, "；"), card.FollowUpRecommendation)
		}
	case string(model.FlowCardKindLabExecution):
		if card.ResultSummary != nil && *card.ResultSummary != "" {
			fmt.Fprintf(b, "- 检验结果：%s\n", *card.ResultSummary)
		}
	}
}

func buildPriorOutcome(parent *model.VisitSession, items []model.TimelineItem) *medagent.Result {
	result := &medagent.Result{}
	if parent.Summary.Diagnosis != nil && *parent.Summary.Diagnosis != "" {
		result.Diagnosis = &medagent.Diagnosis{Name: *parent.Summary.Diagnosis}
	}
	if parent.Summary.TreatmentSummary != nil {
		result.Plan = *parent.Summary.TreatmentSummary
	}

	for _, item := range items {
		if item.Kind != "flow_card" || item.Card == nil {
			continue
		}
		card := item.Card
		switch card.Kind {
		case string(model.FlowCardKindDiagnosis):
			if card.Diagnosis != "" {
				if result.Diagnosis == nil {
					result.Diagnosis = &medagent.Diagnosis{}
				}
				result.Diagnosis.Name = card.Diagnosis
				result.Diagnosis.Basis = strings.Join(card.Evidence, "；")
			}
		case string(model.FlowCardKindMedicationFulfillment):
			for _, med := range card.Medications {
				result.Medications = append(result.Medications, medagent.Medication{
					Name:     med.Name,
					Dosage:   med.Dosage,
					Quantity: med.Quantity,
				})
			}
		case string(model.FlowCardKindCompletedVisit):
			if card.Diagnosis != "" {
				if result.Diagnosis == nil {
					result.Diagnosis = &medagent.Diagnosis{}
				}
				result.Diagnosis.Name = card.Diagnosis
			}
			if card.TreatmentSummary != "" {
				result.Plan = card.TreatmentSummary
			}
			if card.FollowUpSuggestion != "" {
				result.Advice = card.FollowUpSuggestion
			}
		case string(model.FlowCardKindAdviceOnly):
			if len(card.Advices) > 0 {
				result.Advice = strings.Join(card.Advices, "；")
			}
			if card.FollowUpRecommendation != "" {
				result.Advice = strings.TrimSpace(result.Advice + " " + card.FollowUpRecommendation)
			}
		}
	}

	if result.Diagnosis == nil && result.Plan == "" && result.Advice == "" && len(result.Medications) == 0 {
		return nil
	}
	return result
}

func timelineRoleName(role string) string {
	switch role {
	case "patient":
		return "患者"
	case "assistant", "doctor":
		return "医生"
	default:
		return role
	}
}

func trimRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
