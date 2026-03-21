package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/crypto"
)

type AgentConfigRepo struct {
	db  *pgxpool.Pool
	kms *crypto.KMS // nil = encryption disabled
}

func NewAgentConfigRepo(db *pgxpool.Pool, kms *crypto.KMS) *AgentConfigRepo {
	return &AgentConfigRepo{db: db, kms: kms}
}

var ErrNotFound = errors.New("not found")

// AgentConfig is the DB-layer representation of workspace agent configuration.
type AgentConfig struct {
	WorkspaceID      string
	VisionProvider   string
	VisionModel      string
	TranslProvider   string
	TranslModel      string
	ValidProvider    string
	ValidModel       string
	FallbackProvider string
	FallbackModel    string
	OllamaURL        string
	// APIKey is the plaintext key — set on read (decrypted) and write (will be encrypted).
	APIKey string
}

func (r *AgentConfigRepo) GetByWorkspaceID(ctx context.Context, workspaceID string) (*AgentConfig, error) {
	var cfg AgentConfig
	var encryptedKey []byte
	err := r.db.QueryRow(ctx, `
		SELECT workspace_id,
		       vision_provider, vision_model,
		       transl_provider, transl_model,
		       valid_provider, COALESCE(valid_model, ''),
		       COALESCE(fallback_provider, ''), COALESCE(fallback_model, ''),
		       COALESCE(ollama_url, ''), api_key_encrypted
		FROM agent_configs
		WHERE workspace_id = $1
	`, workspaceID).Scan(
		&cfg.WorkspaceID,
		&cfg.VisionProvider, &cfg.VisionModel,
		&cfg.TranslProvider, &cfg.TranslModel,
		&cfg.ValidProvider, &cfg.ValidModel,
		&cfg.FallbackProvider, &cfg.FallbackModel,
		&cfg.OllamaURL, &encryptedKey,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if r.kms != nil && len(encryptedKey) > 0 {
		cfg.APIKey, _ = r.kms.Decrypt(encryptedKey)
	}
	return &cfg, nil
}

func (r *AgentConfigRepo) Upsert(ctx context.Context, cfg AgentConfig, updatedByUserID string) error {
	var encryptedKey any
	if cfg.APIKey != "" && r.kms != nil {
		enc, err := r.kms.Encrypt(cfg.APIKey)
		if err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
		encryptedKey = enc
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO agent_configs (
			workspace_id,
			vision_provider, vision_model,
			transl_provider, transl_model,
			valid_provider,  valid_model,
			fallback_provider, fallback_model,
			ollama_url, api_key_encrypted,
			updated_at, updated_by_user_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),$12)
		ON CONFLICT (workspace_id) DO UPDATE SET
			vision_provider    = EXCLUDED.vision_provider,
			vision_model       = EXCLUDED.vision_model,
			transl_provider    = EXCLUDED.transl_provider,
			transl_model       = EXCLUDED.transl_model,
			valid_provider     = EXCLUDED.valid_provider,
			valid_model        = EXCLUDED.valid_model,
			fallback_provider  = EXCLUDED.fallback_provider,
			fallback_model     = EXCLUDED.fallback_model,
			ollama_url         = EXCLUDED.ollama_url,
			api_key_encrypted  = COALESCE(EXCLUDED.api_key_encrypted, agent_configs.api_key_encrypted),
			updated_at         = NOW(),
			updated_by_user_id = EXCLUDED.updated_by_user_id
	`,
		cfg.WorkspaceID,
		cfg.VisionProvider, cfg.VisionModel,
		cfg.TranslProvider, cfg.TranslModel,
		cfg.ValidProvider, nilIfEmpty(cfg.ValidModel),
		nilIfEmpty(cfg.FallbackProvider), nilIfEmpty(cfg.FallbackModel),
		nilIfEmpty(cfg.OllamaURL), encryptedKey,
		updatedByUserID,
	)
	return err
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
