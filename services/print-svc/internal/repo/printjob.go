package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Job status constants.
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

// PrintJob mirrors the print_jobs table.
type PrintJob struct {
	ID          string
	WorkspaceID string
	LabelID     string
	UserID      string
	Format      string
	Size        string
	Status      string
	PDFS3Key    string
	ZPLPayload  string
	ErrorMsg    string
	Copies      int
	PrinterID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PrintJobRepo struct {
	db *pgxpool.Pool
}

func NewPrintJobRepo(db *pgxpool.Pool) *PrintJobRepo {
	return &PrintJobRepo{db: db}
}

// Create inserts a new print job in pending status.
func (r *PrintJobRepo) Create(ctx context.Context, workspaceID, labelID, userID, format, size string, copies int, printerID string) (*PrintJob, error) {
	var j PrintJob
	err := r.db.QueryRow(ctx, `
		INSERT INTO print_jobs (workspace_id, label_id, user_id, format, size, copies, printer_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''))
		RETURNING id, workspace_id, label_id, user_id, format, size, status,
		          COALESCE(pdf_s3_key,''), COALESCE(zpl_payload,''), COALESCE(error_msg,''),
		          copies, COALESCE(printer_id,''), created_at, updated_at
	`, workspaceID, labelID, userID, format, size, copies, printerID).Scan(
		&j.ID, &j.WorkspaceID, &j.LabelID, &j.UserID, &j.Format, &j.Size, &j.Status,
		&j.PDFS3Key, &j.ZPLPayload, &j.ErrorMsg, &j.Copies, &j.PrinterID,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// GetByID fetches a job by ID, verifying workspace ownership.
func (r *PrintJobRepo) GetByID(ctx context.Context, id, workspaceID string) (*PrintJob, error) {
	var j PrintJob
	err := r.db.QueryRow(ctx, `
		SELECT id, workspace_id, label_id, user_id, format, size, status,
		       COALESCE(pdf_s3_key,''), COALESCE(zpl_payload,''), COALESCE(error_msg,''),
		       copies, COALESCE(printer_id,''), created_at, updated_at
		FROM print_jobs
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID).Scan(
		&j.ID, &j.WorkspaceID, &j.LabelID, &j.UserID, &j.Format, &j.Size, &j.Status,
		&j.PDFS3Key, &j.ZPLPayload, &j.ErrorMsg, &j.Copies, &j.PrinterID,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &j, err
}

// SetProcessing marks a job as processing.
func (r *PrintJobRepo) SetProcessing(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE print_jobs SET status = 'processing', updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

// SetDone marks a job as done with the generated S3 key.
func (r *PrintJobRepo) SetDone(ctx context.Context, id, s3Key string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE print_jobs SET status = 'done', pdf_s3_key = $2, updated_at = NOW() WHERE id = $1
	`, id, s3Key)
	return err
}

// SetZPLDone marks a job as done with the generated ZPL payload (stored in DB, no S3).
func (r *PrintJobRepo) SetZPLDone(ctx context.Context, id, zplPayload string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE print_jobs SET status = 'done', zpl_payload = $2, updated_at = NOW() WHERE id = $1
	`, id, zplPayload)
	return err
}

// SetFailed marks a job as failed with an error message.
func (r *PrintJobRepo) SetFailed(ctx context.Context, id, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE print_jobs SET status = 'failed', error_msg = $2, updated_at = NOW() WHERE id = $1
	`, id, errMsg)
	return err
}
