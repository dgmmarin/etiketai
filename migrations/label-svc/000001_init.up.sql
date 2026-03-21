CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL,
    sku             VARCHAR(100),
    name            VARCHAR(255) NOT NULL,
    category        VARCHAR(50),
    default_fields  JSONB,          -- default translated fields reused across labels
    print_count     INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_workspace ON products (workspace_id);
CREATE INDEX idx_products_sku       ON products (workspace_id, sku) WHERE sku IS NOT NULL;

CREATE TABLE IF NOT EXISTS labels (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL,
    created_by_user_id  UUID NOT NULL,
    product_id          UUID REFERENCES products (id) ON DELETE SET NULL,
    status              VARCHAR(50) NOT NULL DEFAULT 'draft',
    image_s3_key        VARCHAR(500) NOT NULL,
    category            VARCHAR(50),
    detected_language   VARCHAR(10),
    ai_raw_json         JSONB,
    fields_translated   JSONB,
    compliance_score    SMALLINT,
    missing_fields      JSONB,      -- [{"field":"importer_address","severity":"blocker"}]
    confirmed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_labels_workspace_created  ON labels (workspace_id, created_at DESC);
CREATE INDEX idx_labels_workspace_status   ON labels (workspace_id, status);
CREATE INDEX idx_labels_product            ON labels (product_id) WHERE product_id IS NOT NULL;

-- Full-text search on translated product name
CREATE INDEX idx_labels_fts ON labels USING gin (
    to_tsvector('simple', COALESCE(fields_translated->>'product_name', ''))
);

CREATE TABLE IF NOT EXISTS label_audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label_id    UUID NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    action      VARCHAR(100) NOT NULL,  -- upload|ai_processed|field_edited|confirmed|printed
    changes     JSONB,                  -- diff of changed fields vs AI output
    metadata    JSONB,                  -- IP, device, provider_used, etc.
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_label ON label_audit_log (label_id);

CREATE TABLE IF NOT EXISTS print_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label_id        UUID NOT NULL REFERENCES labels (id),
    workspace_id    UUID NOT NULL,
    format          VARCHAR(20) NOT NULL DEFAULT 'pdf',     -- pdf | zpl
    label_size      VARCHAR(50) NOT NULL DEFAULT '100x50',  -- 50x30 | 62x29 | 100x50 | custom
    custom_width_mm INTEGER,
    custom_height_mm INTEGER,
    copies          INTEGER NOT NULL DEFAULT 1,
    labels_per_page INTEGER,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending|generating|ready|sent|printed|failed
    pdf_s3_key      VARCHAR(500),
    zpl_content     TEXT,
    printer_id      VARCHAR(255),
    print_gateway_id VARCHAR(255),
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_print_jobs_label     ON print_jobs (label_id);
CREATE INDEX idx_print_jobs_workspace ON print_jobs (workspace_id, created_at DESC);
