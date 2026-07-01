package workbench

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/neuhis/software-practice-backend/internal/llm"
	"github.com/neuhis/software-practice-backend/internal/model"
)

const titlePrompt = `你是医疗问诊记录标题生成器。根据以下对话内容，生成一个简短的中文标题（不超过50字）。
标题应概括患者的主要症状和持续时间，或已确定的诊断。
格式示例：发热伴咳嗽3天、反复头痛一周、急性胃肠炎
只输出标题本身，不要输出任何其他内容。`

const maxTitleLen = 50

// GenerateTitle generates an AI title for a visit session.
// If a title already exists, it returns the existing one (idempotent).
// On LLM failure, it falls back to a truncated chiefComplaint.
func (s *Service) GenerateTitle(ctx context.Context, sessionID string) (string, error) {
	session, err := s.visitRepo.FindByID(ctx, sessionID)
	if err != nil {
		if err == model.ErrSessionNotFound {
			return "", fmt.Errorf("%w: session not found", model.ErrSessionNotFound)
		}
		return "", fmt.Errorf("find session: %w", err)
	}

	// Idempotent: return existing title
	if session.Summary.Title != nil && *session.Summary.Title != "" {
		return *session.Summary.Title, nil
	}

	// Build conversation context for LLM
	conversationText := s.buildTitleContext(ctx, session)

	// Call LLM
	title, err := s.llmClient.ChatComplete(ctx, titlePrompt, conversationText)
	if err != nil {
		if errors.Is(err, llm.ErrLLMUnavailable) {
			// Fallback: use chiefComplaint truncated
			title = s.fallbackTitle(session)
		} else {
			// Context cancelled or other non-retryable error
			title = s.fallbackTitle(session)
		}
	}

	// Sanitize and validate title
	title = sanitizeTitle(title)

	// Persist title to session summary
	session.UpdatedAt = time.Now()
	session.Summary.Title = &title
	if err := s.visitRepo.Update(ctx, session); err != nil {
		return "", fmt.Errorf("update session title: %w", err)
	}

	return title, nil
}

// buildTitleContext constructs the conversation text for the LLM prompt.
func (s *Service) buildTitleContext(ctx context.Context, session *model.VisitSession) string {
	// If diagnosis exists, prioritize it
	if session.Summary.Diagnosis != nil && *session.Summary.Diagnosis != "" {
		return fmt.Sprintf("已确定诊断：%s", *session.Summary.Diagnosis)
	}

	// Fetch timeline messages
	items, _, _, err := s.timelineRepo.ListBySession(ctx, session.ID, nil, 50)
	if err != nil {
		// Fallback to chiefComplaint if timeline unavailable
		if session.Summary.ChiefComplaint != nil {
			return *session.Summary.ChiefComplaint
		}
		return ""
	}

	var patientMsgs []string
	var assistantMsgs []string
	const maxAssistantMsgs = 2

	for _, item := range items {
		if item.Kind != "message" {
			continue
		}
		switch item.Role {
		case "patient":
			patientMsgs = append(patientMsgs, item.Content)
		case "assistant":
			if len(assistantMsgs) < maxAssistantMsgs {
				assistantMsgs = append(assistantMsgs, item.Content)
			}
		}
	}

	var sb strings.Builder
	if len(patientMsgs) > 0 {
		sb.WriteString("患者消息：\n")
		for _, msg := range patientMsgs {
			sb.WriteString("- ")
			sb.WriteString(msg)
			sb.WriteString("\n")
		}
	}
	if len(assistantMsgs) > 0 {
		sb.WriteString("助手消息：\n")
		for _, msg := range assistantMsgs {
			sb.WriteString("- ")
			sb.WriteString(msg)
			sb.WriteString("\n")
		}
	}

	if sb.Len() == 0 && session.Summary.ChiefComplaint != nil {
		return *session.Summary.ChiefComplaint
	}

	return sb.String()
}

// fallbackTitle generates a fallback title from chiefComplaint.
func (s *Service) fallbackTitle(session *model.VisitSession) string {
	if session.Summary.ChiefComplaint == nil || *session.Summary.ChiefComplaint == "" {
		return "问诊记录"
	}
	cc := *session.Summary.ChiefComplaint
	if utf8.RuneCountInString(cc) <= 15 {
		return cc
	}
	// Truncate to 13 runes + "…"
	runes := []rune(cc)
	return string(runes[:13]) + "…"
}

// sanitizeTitle trims whitespace, removes trailing punctuation, and enforces length limits.
func sanitizeTitle(title string) string {
	title = strings.TrimSpace(title)

	// Remove surrounding quotes if present
	if len(title) >= 2 {
		if (title[0] == '"' && title[len(title)-1] == '"') ||
			(title[0] == '\'' && title[len(title)-1] == '\'') {
			title = title[1 : len(title)-1]
			title = strings.TrimSpace(title)
		}
	}
	// Also handle Chinese quotes
	if strings.HasPrefix(title, "“") && strings.HasSuffix(title, "”") {
		title = strings.TrimPrefix(title, "“")
		title = strings.TrimSuffix(title, "”")
		title = strings.TrimSpace(title)
	}

	// Remove trailing punctuation
	trailingPunct := []string{"。", "，", "；", "、", ".", ",", ";", "！", "!", "？", "?"}
	for _, p := range trailingPunct {
		title = strings.TrimSuffix(title, p)
	}

	// Enforce max length (50 runes)
	if utf8.RuneCountInString(title) > maxTitleLen {
		runes := []rune(title)
		title = string(runes[:maxTitleLen])
	}

	// If empty after sanitization, use default
	if title == "" {
		title = "问诊记录"
	}

	return title
}
