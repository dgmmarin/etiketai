CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE print_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    TEXT NOT NULL,
    label_id        TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    format          TEXT NOT NULL DEFAULT 'pdf',   -- 'pdf' | 'zpl'
    size            TEXT NOT NULL DEFAULT '62x29', -- label size mm
    status          TEXT NOT NULL DEFAULT 'pending',
    pdf_s3_key      TEXT,
    zpl_payload     TEXT,
    error_msg       TEXT,
    copies          INT  NOT NULL DEFAULT 1,
    printer_id      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX print_jobs_workspace_idx ON print_jobs (workspace_id, created_at DESC);
CREATE INDEX print_jobs_label_idx     ON print_jobs (label_id);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$;

CREATE TRIGGER print_jobs_updated_at
    BEFORE UPDATE ON print_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
