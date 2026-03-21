package agentfactory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent/providers/claude"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent/providers/ollama"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/storage"
)

// Config holds agent configuration for a workspace.
type Config struct {
	WorkspaceID      string
	VisionProvider   string // "claude" | "ollama"
	VisionModel      string
	TranslProvider   string
	TranslModel      string
	ValidProvider    string // "claude" | "ollama" | "rules_engine"
	ValidModel       string
	FallbackProvider string
	FallbackModel    string
	OllamaURL        string
	APIKey           string // decrypted per-workspace key; may be empty (uses global)
}

// Factory builds agents from workspace configuration.
type Factory struct {
	configRepo     *repo.AgentConfigRepo
	redis          *redis.Client
	s3             *storage.S3Client
	globalAPIKey   string // ANTHROPIC_API_KEY env var
	cacheTTL       time.Duration
	logger         *zap.Logger
}

func NewFactory(configRepo *repo.AgentConfigRepo, redis *redis.Client, s3 *storage.S3Client, globalAPIKey string, cacheTTL time.Duration, logger *zap.Logger) *Factory {
	return &Factory{
		configRepo:   configRepo,
		redis:        redis,
		s3:           s3,
		globalAPIKey: globalAPIKey,
		cacheTTL:     cacheTTL,
		logger:       logger,
	}
}

// GetVisionAgent returns the primary vision agent for a workspace.
func (f *Factory) GetVisionAgent(ctx context.Context, workspaceID string) (agent.VisionAgent, error) {
	cfg, err := f.loadConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return f.buildVisionAgent(cfg.VisionProvider, cfg.VisionModel, cfg.OllamaURL, cfg.APIKey)
}

// GetFallbackVisionAgent returns the fallback vision agent, or nil if not configured.
func (f *Factory) GetFallbackVisionAgent(ctx context.Context, workspaceID string) (agent.VisionAgent, error) {
	cfg, err := f.loadConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if cfg.FallbackProvider == "" {
		return nil, nil
	}
	return f.buildVisionAgent(cfg.FallbackProvider, cfg.FallbackModel, cfg.OllamaURL, cfg.APIKey)
}

// GetTranslationAgent returns the translation agent for a workspace.
func (f *Factory) GetTranslationAgent(ctx context.Context, workspaceID string) (agent.TranslationAgent, error) {
	cfg, err := f.loadConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return f.buildTranslationAgent(cfg.TranslProvider, cfg.TranslModel, cfg.OllamaURL, cfg.APIKey)
}

// InvalidateCache removes the cached config for a workspace.
func (f *Factory) InvalidateCache(ctx context.Context, workspaceID string) error {
	return f.redis.Del(ctx, cacheKey(workspaceID)).Err()
}

// ─── Internal ─────────────────────────────────────────────────────────────────

func (f *Factory) loadConfig(ctx context.Context, workspaceID string) (*Config, error) {
	// 1. Try Redis cache
	key := cacheKey(workspaceID)
	data, err := f.redis.Get(ctx, key).Bytes()
	if err == nil {
		var cfg Config
		if json.Unmarshal(data, &cfg) == nil {
			return &cfg, nil
		}
	}

	// 2. Load from DB
	dbCfg, err := f.configRepo.GetByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load agent config: %w", err)
	}

	cfg := &Config{
		WorkspaceID:      dbCfg.WorkspaceID,
		VisionProvider:   dbCfg.VisionProvider,
		VisionModel:      dbCfg.VisionModel,
		TranslProvider:   dbCfg.TranslProvider,
		TranslModel:      dbCfg.TranslModel,
		ValidProvider:    dbCfg.ValidProvider,
		ValidModel:       dbCfg.ValidModel,
		FallbackProvider: dbCfg.FallbackProvider,
		FallbackModel:    dbCfg.FallbackModel,
		OllamaURL:        dbCfg.OllamaURL,
		// APIKey is populated by a separate decryption step (future work);
		// empty means the factory will fall back to the global env key.
	}

	// 3. Cache it
	go func() {
		b, _ := json.Marshal(cfg)
		f.redis.Set(context.Background(), key, b, f.cacheTTL)
	}()

	return cfg, nil
}

func (f *Factory) buildVisionAgent(provider, model, ollamaURL, apiKey string) (agent.VisionAgent, error) {
	switch provider {
	case "claude":
		key := f.resolveAPIKey(apiKey)
		if key == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
		}
		return claude.NewVisionAgent(key, model, f.s3), nil
	case "ollama":
		if ollamaURL == "" {
			return nil, fmt.Errorf("ollama_url not configured for workspace")
		}
		return ollama.NewVisionAgent(ollamaURL, model, f.s3), nil
	default:
		return nil, fmt.Errorf("unknown vision provider: %q", provider)
	}
}

func (f *Factory) buildTranslationAgent(provider, model, ollamaURL, apiKey string) (agent.TranslationAgent, error) {
	switch provider {
	case "claude":
		key := f.resolveAPIKey(apiKey)
		if key == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
		}
		return claude.NewTranslationAgent(key, model), nil
	default:
		return nil, fmt.Errorf("unknown translation provider: %q", provider)
	}
}

func (f *Factory) resolveAPIKey(perWorkspace string) string {
	if perWorkspace != "" {
		return perWorkspace
	}
	return f.globalAPIKey
}

func cacheKey(workspaceID string) string {
	return "agent:config:" + workspaceID
}
