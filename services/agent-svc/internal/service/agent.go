package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agentfactory"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/repo"
)

// AgentService orchestrates vision, translation and validation pipeline.
type AgentService struct {
	factory *agentfactory.Factory
	logs    *repo.CallLogRepo
	logger  *zap.Logger
}

func NewAgentService(factory *agentfactory.Factory, logs *repo.CallLogRepo, logger *zap.Logger) *AgentService {
	return &AgentService{factory: factory, logs: logs, logger: logger}
}

// ProcessVision runs the vision agent with fallback support.
func (s *AgentService) ProcessVision(ctx context.Context, req agent.VisionRequest) (*agent.VisionResult, error) {
	primary, err := s.factory.GetVisionAgent(ctx, req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("get vision agent: %w", err)
	}

	result, err := primary.ExtractFields(ctx, req)
	if err == nil {
		s.logCall(ctx, req.WorkspaceID, req.LabelID, "vision", primary.Name(), result.TokensUsed, result.LatencyMS, true, "")
		return result, nil
	}

	s.logger.Warn("primary vision agent failed, trying fallback",
		zap.String("provider", primary.Name()),
		zap.Error(err),
	)
	s.logCall(ctx, req.WorkspaceID, req.LabelID, "vision", primary.Name(), 0, 0, false, err.Error())

	fallback, fErr := s.factory.GetFallbackVisionAgent(ctx, req.WorkspaceID)
	if fErr != nil || fallback == nil {
		return nil, fmt.Errorf("primary failed (%w), no fallback configured", err)
	}

	result, err = fallback.ExtractFields(ctx, req)
	if err != nil {
		s.logCall(ctx, req.WorkspaceID, req.LabelID, "vision", fallback.Name(), 0, 0, false, err.Error())
		return nil, fmt.Errorf("both primary and fallback failed: %w", err)
	}

	s.logCall(ctx, req.WorkspaceID, req.LabelID, "vision", fallback.Name(), result.TokensUsed, result.LatencyMS, true, "")
	return result, nil
}

// ProcessTranslation runs the translation agent.
// For cosmetic labels the "ingredients" field is excluded from translation to
// preserve INCI (International Nomenclature of Cosmetic Ingredients) naming
// (T-0806). The original value is re-attached after translation completes.
func (s *AgentService) ProcessTranslation(ctx context.Context, req agent.TranslRequest) (*agent.TranslResult, error) {
	ag, err := s.factory.GetTranslationAgent(ctx, req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("get translation agent: %w", err)
	}

	// Preserve INCI ingredients for cosmetics — pass a copy without the field.
	var inciIngredients *string
	if req.Category == agent.CategoryCosmetic {
		if v, ok := req.Fields["ingredients"]; ok {
			inciIngredients = v
			fieldsWithoutIngredients := make(map[string]*string, len(req.Fields))
			for k, fv := range req.Fields {
				if k != "ingredients" {
					fieldsWithoutIngredients[k] = fv
				}
			}
			req.Fields = fieldsWithoutIngredients
		}
	}

	result, err := ag.Translate(ctx, req)
	if err != nil {
		s.logCall(ctx, req.WorkspaceID, req.LabelID, "translation", ag.Name(), 0, 0, false, err.Error())
		return nil, err
	}

	// Re-attach the original INCI ingredients (untranslated).
	if inciIngredients != nil {
		result.Translated["ingredients"] = inciIngredients
	}

	s.logCall(ctx, req.WorkspaceID, req.LabelID, "translation", ag.Name(), result.TokensUsed, result.LatencyMS, true, "")
	return result, nil
}

// ─── Logging ──────────────────────────────────────────────────────────────────

func (s *AgentService) logCall(ctx context.Context, workspaceID, labelID, agentType, provider string, tokens int, latencyMS int64, success bool, errMsg string) {
	s.logCallDetailed(ctx, workspaceID, labelID, agentType, provider, "", 0, 0, tokens, latencyMS, success, errMsg)
}

func (s *AgentService) logCallDetailed(ctx context.Context, workspaceID, labelID, agentType, provider, model string, tokensIn, tokensOut, tokensTotal int, latencyMS int64, success bool, errMsg string) {
	go func() {
		entry := repo.CallLogEntry{
			WorkspaceID:  workspaceID,
			LabelID:      labelID,
			AgentType:    agentType,
			Provider:     provider,
			Model:        model,
			TokensInput:  tokensIn,
			TokensOutput: tokensOut,
			LatencyMS:    int(latencyMS),
			Success:      success,
			ErrorMessage: errMsg,
			CalledAt:     time.Now(),
		}
		if tokensIn == 0 && tokensOut == 0 && tokensTotal > 0 {
			entry.TokensInput = int(float64(tokensTotal) * 0.6)
			entry.TokensOutput = tokensTotal - entry.TokensInput
		}
		entry.CostUSD = agent.ComputeCost(provider, model, entry.TokensInput, entry.TokensOutput, tokensTotal)
		if err := s.logs.Insert(context.Background(), entry); err != nil {
			s.logger.Warn("failed to log agent call", zap.Error(err))
		}
	}()
}
