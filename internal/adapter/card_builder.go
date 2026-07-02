// Package adapter provides mapping functions from medAgent domain types to
// frontend-facing model types (SSE events, FlowCards, TimelineItems).
//
// DESIGN NOTE: This package imports internal/service/medagent for the medAgent
// domain types (Step, Result, StepKind, etc.). This creates a dependency
// adapter → service/medagent. The workbench service depends on both adapter
// and service/medagent, so there is no true circular dependency. However,
// for ideal clean architecture, the shared medAgent types could be moved to
// internal/model/medagent/ in the future.
package adapter

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuhis/software-practice-backend/internal/model"
	medagent "github.com/neuhis/software-practice-backend/internal/service/medagent"
)

// newBaseCard creates a FlowCard with the common shared fields populated.
func newBaseCard(sessionID string, kind model.FlowCardKind, status model.FlowCardStatus, title string) *model.FlowCard {
	now := time.Now()
	return &model.FlowCard{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Kind:      string(kind),
		Status:    string(status),
		Title:     title,
		CreatedAt: now,
	}
}

// BuildLabDecisionCard creates a lab_decision FlowCard from a NEED_TESTS step.
func BuildLabDecisionCard(sessionID string, step *medagent.Step) *model.FlowCard {
	card := newBaseCard(sessionID, model.FlowCardKindLabDecision, model.FlowCardStatusPending, "是否需要进行检验？")
	card.Blocking = true

	for _, item := range step.TestItems {
		card.TestItems = append(card.TestItems, model.TestItem{
			Code: item,
			Name: item,
		})
	}

	// reason is required by the frontend schema (z.string().trim().min(1)); fall
	// back to a clinical default when medAgent does not provide a doctor message.
	card.Reason = step.DoctorSay
	if card.Reason == "" {
		card.Reason = "根据病情需要进一步检验以明确诊断"
	}

	// differentialTargets is required by the frontend schema (z.array(...));
	// always emit at least an empty array so the JSON key is present with [].
	card.DifferentialTargets = []string{}

	card.EstimatedFee = model.Float64Ptr(50.0) // 血常规固定费用

	return card
}

// BuildDiagnosisCard creates a diagnosis FlowCard from a DONE step result.
func BuildDiagnosisCard(sessionID string, result *medagent.Result) *model.FlowCard {
	card := newBaseCard(sessionID, model.FlowCardKindDiagnosis, model.FlowCardStatusCompleted, "诊断结果")
	now := time.Now()
	card.HandledAt = &now

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
	card := newBaseCard(sessionID, model.FlowCardKindTreatmentPlan, model.FlowCardStatusCompleted, "处置方案")

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
func BuildMedicationFulfillmentCard(sessionID string, step *medagent.Step, resolvedMedications ...[]model.MedicationItem) *model.FlowCard {
	card := newBaseCard(sessionID, model.FlowCardKindMedicationFulfillment, model.FlowCardStatusPending, "购药确认")
	card.Blocking = true

	if len(resolvedMedications) > 0 {
		card.Medications = append(card.Medications, resolvedMedications[0]...)
	} else {
		for _, order := range step.Orders {
			card.Medications = append(card.Medications, model.MedicationItem{
				Name:     order.Name,
				Quantity: order.Quantity,
				Spec:     fmt.Sprintf("%d盒", order.Quantity),
				Price:    0, // 价格由药房系统填充
			})
		}
	}
	card.AvailableModes = []string{"pickup", "delivery"}
	card.FulfillmentStatus = model.MedicationFulfillmentStatusPending

	return card
}

// BuildCompletedVisitCard creates a completed_visit FlowCard.
func BuildCompletedVisitCard(sessionID string, result *medagent.Result) *model.FlowCard {
	card := newBaseCard(sessionID, model.FlowCardKindCompletedVisit, model.FlowCardStatusCompleted, "就诊完成")
	now := time.Now()
	card.HandledAt = &now
	card.CompletedAt = now

	if result.Diagnosis != nil {
		card.Diagnosis = result.Diagnosis.Name
	}
	card.TreatmentSummary = result.Plan
	card.FollowUpSuggestion = result.Advice

	return card
}

// BuildAdviceOnlyCard creates an advice_only FlowCard.
func BuildAdviceOnlyCard(sessionID string, result *medagent.Result) *model.FlowCard {
	card := newBaseCard(sessionID, model.FlowCardKindAdviceOnly, model.FlowCardStatusPending, "医嘱确认")
	card.Blocking = true

	card.Advices = append(card.Advices, result.Advice)
	card.FollowUpRecommendation = "请遵医嘱，如有不适及时复诊"

	return card
}

// BuildPaymentCard creates a payment FlowCard.
func BuildPaymentCard(sessionID, purpose string, items []model.PaymentLineItem, totalAmount float64) *model.FlowCard {
	card := newBaseCard(sessionID, model.FlowCardKindPayment, model.FlowCardStatusPending, "缴费")
	card.Blocking = true
	card.Purpose = purpose
	card.Items = items
	card.TotalAmount = model.Float64Ptr(totalAmount)
	card.InsuranceAmount = model.Float64Ptr(0)
	card.SelfPayAmount = model.Float64Ptr(totalAmount)
	card.PaymentStatus = string(model.PaymentStatusUnpaid)
	card.PaymentID = card.ID

	return card
}
