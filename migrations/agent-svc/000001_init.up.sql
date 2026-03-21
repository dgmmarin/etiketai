CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS agent_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL UNIQUE,
    vision_provider     VARCHAR(50) NOT NULL DEFAULT 'claude',
    vision_model        VARCHAR(100) NOT NULL DEFAULT 'claude-sonnet-4-6',
    transl_provider     VARCHAR(50) NOT NULL DEFAULT 'claude',
    transl_model        VARCHAR(100) NOT NULL DEFAULT 'claude-sonnet-4-6',
    valid_provider      VARCHAR(50) NOT NULL DEFAULT 'rules_engine',
    valid_model         VARCHAR(100),
    fallback_provider   VARCHAR(50),
    fallback_model      VARCHAR(100),
    ollama_url          VARCHAR(500),
    api_key_encrypted   BYTEA,       -- AES-256-GCM encrypted; null = use global env key
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_user_id  UUID NOT NULL
);

CREATE INDEX idx_agent_configs_workspace ON agent_configs (workspace_id);

CREATE TABLE IF NOT EXISTS agent_call_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL,
    label_id        UUID NOT NULL,
    agent_type      VARCHAR(50) NOT NULL,       -- vision | translation | validation
    provider        VARCHAR(50) NOT NULL,
    model           VARCHAR(100) NOT NULL,
    tokens_input    INTEGER,
    tokens_output   INTEGER,
    cost_usd        NUMERIC(10, 6) NOT NULL DEFAULT 0,
    latency_ms      INTEGER NOT NULL,
    success         BOOLEAN NOT NULL,
    error_message   TEXT,
    called_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_call_logs_workspace ON agent_call_logs (workspace_id, called_at DESC);
CREATE INDEX idx_agent_call_logs_label     ON agent_call_logs (label_id);
CREATE INDEX idx_agent_call_logs_called_at ON agent_call_logs (called_at);  -- for cleanup job
