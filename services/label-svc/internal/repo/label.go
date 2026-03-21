package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Label status constants.
const (
	StatusDraft      = "draft"
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusConfirmed  = "confirmed"
	StatusFailed     = "failed"
)

type Label struct {
	ID                string
	WorkspaceID       string
	CreatedByUserID   string
	ProductID         string
	Status            string
	ImageS3Key        string
	Category          string
	DetectedLanguage  string
	AIRawJSON         map[string]any
	FieldsTranslated  map[string]*string
	ComplianceScore   int
	MissingFields     []map[string]string
	ConfirmedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ListFilter struct {
	WorkspaceID string
	Status      string
	Category    string
	Query       string
	From        string
	To          string
	Page        int
	PerPage     int
}

type LabelRepo struct {
	db *pgxpool.Pool
}

func NewLabelRepo(db *pgxpool.Pool) *LabelRepo {
	return &LabelRepo{db: db}
}

// Create inserts a new label in draft status.
func (r *LabelRepo) Create(ctx context.Context, workspaceID, userID, imageS3Key string) (*Label, error) {
	var l Label
	err := r.db.QueryRow(ctx, `
		INSERT INTO labels (workspace_id, created_by_user_id, image_s3_key)
		VALUES ($1, $2, $3)
		RETURNING id, workspace_id, created_by_user_id, COALESCE(product_id::text,''),
		          status, image_s3_key, COALESCE(category,''), COALESCE(detected_language,''),
		          created_at, updated_at
	`, workspaceID, userID, imageS3Key).Scan(
		&l.ID, &l.WorkspaceID, &l.CreatedByUserID, &l.ProductID,
		&l.Status, &l.ImageS3Key, &l.Category, &l.DetectedLanguage,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetByID returns a label, checking workspace ownership.
func (r *LabelRepo) GetByID(ctx context.Context, id, workspaceID string) (*Label, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, workspace_id, created_by_user_id, COALESCE(product_id::text,''),
		       status, image_s3_key, COALESCE(category,''), COALESCE(detected_language,''),
		       ai_raw_json, fields_translated, COALESCE(compliance_score,0),
		       missing_fields, confirmed_at, created_at, updated_at
		FROM labels
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID)

	return scanLabel(row)
}

// SetStatus updates the status field.
func (r *LabelRepo) SetStatus(ctx context.Context, id, workspaceID, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE labels SET status = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID, status)
	return err
}

// SetAIResult stores the AI vision output and transitions to ready status.
func (r *LabelRepo) SetAIResult(ctx context.Context, id, workspaceID, category, detectedLang string, aiRaw map[string]any, translated map[string]*string, score int, missing []map[string]string) error {
	aiJSON, _ := json.Marshal(aiRaw)
	translJSON, _ := json.Marshal(translated)
	missingJSON, _ := json.Marshal(missing)

	_, err := r.db.Exec(ctx, `
		UPDATE labels
		SET status             = 'ready',
		    category           = $3,
		    detected_language  = $4,
		    ai_raw_json        = $5,
		    fields_translated  = $6,
		    compliance_score   = $7,
		    missing_fields     = $8,
		    updated_at         = NOW()
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID, category, detectedLang, aiJSON, translJSON, score, missingJSON)
	return err
}

// UpdateFields updates the translated fields (operator edits).
func (r *LabelRepo) UpdateFields(ctx context.Context, id, workspaceID string, fields map[string]*string) error {
	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE labels
		SET fields_translated = fields_translated || $3::jsonb,
		    updated_at        = NOW()
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID, fieldsJSON)
	return err
}

// Confirm marks a label as confirmed and records the timestamp.
func (r *LabelRepo) Confirm(ctx context.Context, id, workspaceID, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE labels SET status = 'confirmed', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND status = 'ready'
	`, id, workspaceID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO label_audit_log (label_id, user_id, action)
		VALUES ($1, $2, 'confirmed')
	`, id, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete soft-deletes a label (status = 'deleted').
func (r *LabelRepo) Delete(ctx context.Context, id, workspaceID string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM labels WHERE id = $1 AND workspace_id = $2 AND status != 'confirmed'
	`, id, workspaceID)
	return err
}

// List returns paginated labels for a workspace with optional filters.
func (r *LabelRepo) List(ctx context.Context, f ListFilter) ([]Label, int, error) {
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PerPage

	// Build WHERE clause
	args := []any{f.WorkspaceID}
	where := "workspace_id = $1"
	if f.Status != "" {
		args = append(args, f.Status)
		where += " AND status = $" + itoa(len(args))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		where += " AND category = $" + itoa(len(args))
	}
	if f.Query != "" {
		args = append(args, f.Query)
		where += " AND to_tsvector('simple', COALESCE(fields_translated->>'product_name','')) @@ plainto_tsquery('simple', $" + itoa(len(args)) + ")"
	}
	if f.From != "" {
		args = append(args, f.From)
		where += " AND created_at >= $" + itoa(len(args))
	}
	if f.To != "" {
		args = append(args, f.To)
		where += " AND created_at <= $" + itoa(len(args))
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM labels WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.PerPage, offset)
	rows, err := r.db.Query(ctx, `
		SELECT id, workspace_id, created_by_user_id, COALESCE(product_id::text,''),
		       status, image_s3_key, COALESCE(category,''), COALESCE(detected_language,''),
		       NULL::jsonb, NULL::jsonb, COALESCE(compliance_score,0),
		       NULL::jsonb, confirmed_at, created_at, updated_at
		FROM labels
		WHERE `+where+`
		ORDER BY created_at DESC
		LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var labels []Label
	for rows.Next() {
		l, err := scanLabel(rows)
		if err != nil {
			return nil, 0, err
		}
		labels = append(labels, *l)
	}
	return labels, total, rows.Err()
}

// AppendAuditLog records an action on a label.
func (r *LabelRepo) AppendAuditLog(ctx context.Context, labelID, userID, action string, changes, metadata map[string]any) error {
	changesJSON, _ := json.Marshal(changes)
	metaJSON, _ := json.Marshal(metadata)
	_, err := r.db.Exec(ctx, `
		INSERT INTO label_audit_log (label_id, user_id, action, changes, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, labelID, userID, action, changesJSON, metaJSON)
	return err
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanLabel(row scanner) (*Label, error) {
	var l Label
	var aiRawJSON, translJSON, missingJSON []byte
	var confirmedAt *time.Time

	err := row.Scan(
		&l.ID, &l.WorkspaceID, &l.CreatedByUserID, &l.ProductID,
		&l.Status, &l.ImageS3Key, &l.Category, &l.DetectedLanguage,
		&aiRawJSON, &translJSON, &l.ComplianceScore,
		&missingJSON, &confirmedAt, &l.CreatedAt, &l.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if aiRawJSON != nil {
		_ = json.Unmarshal(aiRawJSON, &l.AIRawJSON)
	}
	if translJSON != nil {
		_ = json.Unmarshal(translJSON, &l.FieldsTranslated)
	}
	if missingJSON != nil {
		_ = json.Unmarshal(missingJSON, &l.MissingFields)
	}
	l.ConfirmedAt = confirmedAt
	return &l, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
