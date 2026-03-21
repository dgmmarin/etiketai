package grpchandler

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentv1 "github.com/dgmmarin/etiketai/gen/agent/v1"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent/providers/rules"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/service"
)

// AgentHandler implements agentv1.AgentServiceServer.
type AgentHandler struct {
	agentv1.UnimplementedAgentServiceServer
	svc        *service.AgentService
	config     *repo.AgentConfigRepo
	validation agent.ValidationAgent
}

func NewAgentHandler(svc *service.AgentService, config *repo.AgentConfigRepo) *AgentHandler {
	return &AgentHandler{
		svc:        svc,
		config:     config,
		validation: rules.NewValidationAgent(),
	}
}

func Register(srv *grpc.Server, svc *service.AgentService, config *repo.AgentConfigRepo) {
	agentv1.RegisterAgentServiceServer(srv, NewAgentHandler(svc, config))
}

func (h *AgentHandler) ProcessVision(ctx context.Context, req *agentv1.VisionRequest) (*agentv1.VisionResponse, error) {
	result, err := h.svc.ProcessVision(ctx, agent.VisionRequest{
		LabelID:        req.LabelId,
		ImageS3Key:     req.ImageS3Key,
		WorkspaceID:    req.WorkspaceId,
		TargetLanguage: req.TargetLanguage,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}

	rawJSON, _ := json.Marshal(result)
	confidence := agent.ComputeQualityScore(result.Confidence)

	return &agentv1.VisionResponse{
		LabelId:      req.LabelId,
		RawJson:      string(rawJSON),
		Confidence:   confidence,
		DetectedLang: result.DetectedLang,
		ProviderUsed: result.ProviderUsed,
		TokensUsed:   int32(result.TokensUsed),
		LatencyMs:    int32(result.LatencyMS),
	}, nil
}

func (h *AgentHandler) ProcessTranslation(ctx context.Context, req *agentv1.TranslRequest) (*agentv1.TranslResponse, error) {
	var fields map[string]*string
	if req.FieldsJson != "" {
		if err := json.Unmarshal([]byte(req.FieldsJson), &fields); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid fields_json: %s", err.Error())
		}
	}

	result, err := h.svc.ProcessTranslation(ctx, agent.TranslRequest{
		LabelID:     req.LabelId,
		WorkspaceID: req.WorkspaceId,
		Fields:      fields,
		Category:    agent.ProductCategory(req.Category),
		SourceLang:  req.SourceLang,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}

	translatedJSON, _ := json.Marshal(result.Translated)
	return &agentv1.TranslResponse{
		LabelId:        req.LabelId,
		TranslatedJson: string(translatedJSON),
		ProviderUsed:   result.ProviderUsed,
		TokensUsed:     int32(result.TokensUsed),
		LatencyMs:      int32(result.LatencyMS),
	}, nil
}

func (h *AgentHandler) ProcessValidation(ctx context.Context, req *agentv1.ValidRequest) (*agentv1.ValidResponse, error) {
	var fields map[string]*string
	if req.TranslatedJson != "" {
		if err := json.Unmarshal([]byte(req.TranslatedJson), &fields); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid translated_json: %s", err.Error())
		}
	}

	result, err := h.validation.Validate(ctx, agent.ValidRequest{
		LabelID:     req.LabelId,
		WorkspaceID: req.WorkspaceId,
		Fields:      fields,
		Category:    agent.ProductCategory(req.Category),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}

	var missing []*agentv1.MissingField
	for _, m := range result.Missing {
		missing = append(missing, &agentv1.MissingField{
			Field:    m.Field,
			Severity: m.Severity,
			Message:  m.Message,
		})
	}

	return &agentv1.ValidResponse{
		LabelId:         req.LabelId,
		ComplianceScore: int32(result.Score),
		Missing:         missing,
		RulesVersion:    result.RulesVersion,
	}, nil
}

func (h *AgentHandler) GetAgentConfig(ctx context.Context, req *agentv1.ConfigRequest) (*agentv1.AgentConfig, error) {
	cfg, err := h.config.GetByWorkspaceID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "config not found: %s", err.Error())
	}
	return &agentv1.AgentConfig{
		WorkspaceId: cfg.WorkspaceID,
		Vision:      &agentv1.ProviderConfig{Provider: cfg.VisionProvider, Model: cfg.VisionModel},
		Translation: &agentv1.ProviderConfig{Provider: cfg.TranslProvider, Model: cfg.TranslModel},
		Validation:  &agentv1.ProviderConfig{Provider: cfg.ValidProvider, Model: cfg.ValidModel},
		Fallback:    &agentv1.ProviderConfig{Provider: cfg.FallbackProvider, Model: cfg.FallbackModel},
		OllamaUrl:   cfg.OllamaURL,
	}, nil
}

func (h *AgentHandler) UpdateAgentConfig(ctx context.Context, req *agentv1.UpdateConfigRequest) (*agentv1.UpdateConfigResponse, error) {
	cfg := repo.AgentConfig{WorkspaceID: req.WorkspaceId, OllamaURL: req.OllamaUrl}
	if req.Vision != nil {
		cfg.VisionProvider = req.Vision.Provider
		cfg.VisionModel = req.Vision.Model
	}
	if req.Translation != nil {
		cfg.TranslProvider = req.Translation.Provider
		cfg.TranslModel = req.Translation.Model
	}
	if req.Validation != nil {
		cfg.ValidProvider = req.Validation.Provider
		cfg.ValidModel = req.Validation.Model
	}
	if req.Fallback != nil {
		cfg.FallbackProvider = req.Fallback.Provider
		cfg.FallbackModel = req.Fallback.Model
	}
	if err := h.config.Upsert(ctx, cfg, req.UpdatedBy); err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &agentv1.UpdateConfigResponse{Success: true, CacheInvalidated: true}, nil
}

func (h *AgentHandler) TestAgentConfig(ctx context.Context, req *agentv1.TestConfigRequest) (*agentv1.TestConfigResponse, error) {
	cfg, err := h.config.GetByWorkspaceID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%s", err.Error())
	}
	var provider, model string
	switch req.AgentType {
	case "translation":
		provider, model = cfg.TranslProvider, cfg.TranslModel
	case "validation":
		provider, model = cfg.ValidProvider, cfg.ValidModel
	default:
		provider, model = cfg.VisionProvider, cfg.VisionModel
	}
	return &agentv1.TestConfigResponse{
		Success:  true,
		Provider: provider,
		Model:    model,
	}, nil
}
