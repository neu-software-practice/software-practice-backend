package adapter

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// BuildLabDecisionCard creates a lab_decision FlowCard from a NEED_TESTS step.
func BuildLabDecisionCard(sessionID string, step *medagent.Step) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.FlowCardKindLabDecision),
		Status:    string(model.FlowCardStatusPending),
		Blocking:  true,
		Title:     "是否需要进行检验？",
		CreatedAt: now,
	}

	for _, item := range step.TestItems {
		card.TestItems = append(card.TestItems, model.TestItem{
			Code: item,
			Name: item,
		})
	}

	card.Reason = step.DoctorSay
	card.EstimatedFee = 50.0 // 血常规固定费用

	return card
}

// BuildDiagnosisCard creates a diagnosis FlowCard from a DONE step result.
func BuildDiagnosisCard(sessionID string, result *medagent.Result) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.FlowCardKindDiagnosis),
		Status:    string(model.FlowCardStatusCompleted),
		Blocking:  false,
		Title:     "诊断结果",
		CreatedAt: now,
		HandledAt: &now,
	}

	if result.Diagnosis != nil {
		card.Diagnosis = result.Diagnosis.Name
		card.Evidence = append(card.Evidence, result.Diagnosis.Basis)
		confidence := "medium"
		if result.Diagnosis.Confidence >= 0.8 {
			confidence = "high"
		} else if result.Diagnosis.Confidence < 0.5 {
			confidence = "low"
		}
		card.Confidence = confidence
	}

	return card
}

// BuildTreatmentPlanCard creates a treatment_plan FlowCard.
func BuildTreatmentPlanCard(sessionID string, result *medagent.Result) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.FlowCardKindTreatmentPlan),
		Status:    string(model.FlowCardStatusCompleted),
		Blocking:  false,
		Title:     "处置方案",
		CreatedAt: now,
	}

	card.Plan = result.Plan
	card.Summary = result.Advice

	capability := "available"
	if result.Plan == "REFERRAL" {
		capability = "unavailable"
	}
	card.Capability = capability

	return card
}

// BuildMedicationFulfillmentCard creates a medication_fulfillment FlowCard from a PURCHASE step.
func BuildMedicationFulfillmentCard(sessionID string, step *medagent.Step) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.FlowCardKindMedicationFulfillment),
		Status:    string(model.FlowCardStatusPending),
		Blocking:  true,
		Title:     "购药确认",
		CreatedAt: now,
	}

	for _, order := range step.Orders {
		card.Medications = append(card.Medications, model.MedicationItem{
			Name:     order.Name,
			Quantity: order.Quantity,
			Spec:     fmt.Sprintf("%d盒", order.Quantity),
			Price:    0, // 价格由药房系统填充
		})
	}
	card.AvailableModes = []string{"pickup", "delivery"}
	card.FulfillmentStatus = "pending"

	return card
}

// BuildCompletedVisitCard creates a completed_visit FlowCard.
func BuildCompletedVisitCard(sessionID string, result *medagent.Result) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Kind:        string(model.FlowCardKindCompletedVisit),
		Status:      string(model.FlowCardStatusCompleted),
		Blocking:    false,
		Title:       "就诊完成",
		CreatedAt:   now,
		HandledAt:   &now,
		CompletedAt: now,
	}

	if result.Diagnosis != nil {
		card.Diagnosis = result.Diagnosis.Name
	}
	card.TreatmentSummary = result.Plan
	card.FollowUpSuggestion = result.Advice

	return card
}

// BuildAdviceOnlyCard creates an advice_only FlowCard.
func BuildAdviceOnlyCard(sessionID string, result *medagent.Result) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(model.FlowCardKindAdviceOnly),
		Status:    string(model.FlowCardStatusPending),
		Blocking:  true,
		Title:     "医嘱确认",
		CreatedAt: now,
	}

	card.Advices = append(card.Advices, result.Advice)
	card.FollowUpRecommendation = "请遵医嘱，如有不适及时复诊"

	return card
}

// BuildPaymentCard creates a payment FlowCard.
func BuildPaymentCard(sessionID, purpose string, items []model.PaymentLineItem, totalAmount float64) *model.FlowCard {
	now := time.Now()
	card := &model.FlowCard{
		ID:              uuid.New().String(),
		SessionID:       sessionID,
		Kind:            string(model.FlowCardKindPayment),
		Status:          string(model.FlowCardStatusPending),
		Blocking:        true,
		Title:           "缴费",
		CreatedAt:       now,
		Purpose:         purpose,
		Items:           items,
		TotalAmount:     totalAmount,
		InsuranceAmount: 0,
		SelfPayAmount:   totalAmount,
		PaymentStatus:   string(model.PaymentStatusUnpaid),
	}

	return card
}
