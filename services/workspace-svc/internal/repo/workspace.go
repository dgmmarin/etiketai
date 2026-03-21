package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Workspace is the DB-level workspace record.
type Workspace struct {
	ID                    string
	Name                  string
	CUI                   string
	Address               string
	Phone                 string
	LogoS3Key             string
	Plan                  string
	LabelQuotaMonthly     int
	LabelQuotaUsed        int
	StripeCustomerID      string
	StripeSubscriptionID  string
	SubscriptionPeriodEnd *time.Time
	CreatedAt             time.Time
}

// WorkspaceMember is a member row.
type WorkspaceMember struct {
	ID          string
	WorkspaceID string
	UserID      string
	Email       string
	Role        string
	InviteToken string
	AcceptedAt  *time.Time
	RevokedAt   *time.Time
}

type WorkspaceRepo struct {
	db *pgxpool.Pool
}

func NewWorkspaceRepo(db *pgxpool.Pool) *WorkspaceRepo {
	return &WorkspaceRepo{db: db}
}

// Create inserts a new workspace and its first admin member atomically.
func (r *WorkspaceRepo) Create(ctx context.Context, name, cui, ownerUserID, ownerEmail string) (*Workspace, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var ws Workspace
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, cui)
		VALUES ($1, $2)
		RETURNING id, name, COALESCE(cui,''), plan, label_quota_monthly, label_quota_used, created_at
	`, name, nilIfEmpty(cui)).Scan(
		&ws.ID, &ws.Name, &ws.CUI,
		&ws.Plan, &ws.LabelQuotaMonthly, &ws.LabelQuotaUsed,
		&ws.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, email, role, accepted_at)
		VALUES ($1, $2, $3, 'admin', NOW())
	`, ws.ID, ownerUserID, ownerEmail)
	if err != nil {
		return nil, err
	}

	return &ws, tx.Commit(ctx)
}

// GetByID returns workspace by primary key.
func (r *WorkspaceRepo) GetByID(ctx context.Context, id string) (*Workspace, error) {
	var ws Workspace
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(cui,''), COALESCE(address,''), COALESCE(phone,''), COALESCE(logo_s3_key,''),
		       plan, label_quota_monthly, label_quota_used,
		       COALESCE(stripe_customer_id,''), COALESCE(stripe_subscription_id,''),
		       subscription_expires_at, created_at
		FROM workspaces WHERE id = $1
	`, id).Scan(
		&ws.ID, &ws.Name, &ws.CUI, &ws.Address, &ws.Phone, &ws.LogoS3Key,
		&ws.Plan, &ws.LabelQuotaMonthly, &ws.LabelQuotaUsed,
		&ws.StripeCustomerID, &ws.StripeSubscriptionID,
		&ws.SubscriptionPeriodEnd, &ws.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &ws, err
}

// UpdateProfile updates editable workspace profile fields.
func (r *WorkspaceRepo) UpdateProfile(ctx context.Context, workspaceID, name, cui, address, phone, logoS3Key string) (*Workspace, error) {
	var ws Workspace
	err := r.db.QueryRow(ctx, `
		UPDATE workspaces
		SET name       = COALESCE(NULLIF($2,''), name),
		    cui        = COALESCE(NULLIF($3,''), cui),
		    address    = COALESCE(NULLIF($4,''), address),
		    phone      = COALESCE(NULLIF($5,''), phone),
		    logo_s3_key = COALESCE(NULLIF($6,''), logo_s3_key)
		WHERE id = $1
		RETURNING id, name, COALESCE(cui,''), COALESCE(address,''), COALESCE(phone,''), COALESCE(logo_s3_key,''),
		          plan, label_quota_monthly, label_quota_used,
		          COALESCE(stripe_customer_id,''), COALESCE(stripe_subscription_id,''),
		          subscription_expires_at, created_at
	`, workspaceID, name, cui, address, phone, logoS3Key).Scan(
		&ws.ID, &ws.Name, &ws.CUI, &ws.Address, &ws.Phone, &ws.LogoS3Key,
		&ws.Plan, &ws.LabelQuotaMonthly, &ws.LabelQuotaUsed,
		&ws.StripeCustomerID, &ws.StripeSubscriptionID,
		&ws.SubscriptionPeriodEnd, &ws.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &ws, err
}

// GetMemberByUserID returns workspace + role for a given user.
func (r *WorkspaceRepo) GetMemberByUserID(ctx context.Context, userID string) (*WorkspaceMember, error) {
	var m WorkspaceMember
	err := r.db.QueryRow(ctx, `
		SELECT wm.id, wm.workspace_id, wm.user_id, COALESCE(wm.email,''), wm.role
		FROM workspace_members wm
		WHERE wm.user_id = $1
		  AND wm.revoked_at IS NULL
		  AND wm.accepted_at IS NOT NULL
		LIMIT 1
	`, userID).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Email, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

// GetMemberByEmail looks up a member by denormalized email.
func (r *WorkspaceRepo) GetMemberByEmail(ctx context.Context, email string) (*WorkspaceMember, error) {
	var m WorkspaceMember
	err := r.db.QueryRow(ctx, `
		SELECT wm.id, wm.workspace_id, wm.user_id, COALESCE(wm.email,''), wm.role
		FROM workspace_members wm
		WHERE wm.email = $1
		  AND wm.revoked_at IS NULL
		  AND wm.accepted_at IS NOT NULL
		LIMIT 1
	`, email).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Email, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

// CheckAndIncrementQuota atomically checks quota and increments if allowed.
// Returns (allowed bool, used int).
func (r *WorkspaceRepo) CheckAndIncrementQuota(ctx context.Context, workspaceID string) (bool, int, error) {
	var allowed bool
	var used int
	err := r.db.QueryRow(ctx, `
		UPDATE workspaces
		SET label_quota_used = label_quota_used + 1
		WHERE id = $1
		  AND label_quota_used < label_quota_monthly
		RETURNING true, label_quota_used
	`, workspaceID).Scan(&allowed, &used)
	if errors.Is(err, pgx.ErrNoRows) {
		// Quota exceeded
		var current int
		_ = r.db.QueryRow(ctx, `SELECT label_quota_used FROM workspaces WHERE id = $1`, workspaceID).Scan(&current)
		return false, current, nil
	}
	return allowed, used, err
}

// CheckQuota returns (allowed, used) without incrementing.
func (r *WorkspaceRepo) CheckQuota(ctx context.Context, workspaceID string) (bool, int, error) {
	var monthly, used int
	err := r.db.QueryRow(ctx, `
		SELECT label_quota_monthly, label_quota_used FROM workspaces WHERE id = $1
	`, workspaceID).Scan(&monthly, &used)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, ErrNotFound
	}
	return used < monthly, used, err
}

// ListMembers returns active members of a workspace.
func (r *WorkspaceRepo) ListMembers(ctx context.Context, workspaceID string) ([]WorkspaceMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, workspace_id, user_id, COALESCE(email,''), role
		FROM workspace_members
		WHERE workspace_id = $1
		  AND revoked_at IS NULL
		ORDER BY invited_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []WorkspaceMember
	for rows.Next() {
		var m WorkspaceMember
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Email, &m.Role); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// InviteMember inserts a pending invite.
func (r *WorkspaceRepo) InviteMember(ctx context.Context, workspaceID, email, role, tokenHash, invitedByUserID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, email, role, invite_token_hash, invited_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (workspace_id, user_id) DO NOTHING
	`, workspaceID, email, role, tokenHash, nilIfEmpty(invitedByUserID))
	return err
}

// AcceptInvite marks an invite as accepted and sets the user_id.
func (r *WorkspaceRepo) AcceptInvite(ctx context.Context, tokenHash, userID string) (*WorkspaceMember, error) {
	var m WorkspaceMember
	err := r.db.QueryRow(ctx, `
		UPDATE workspace_members
		SET user_id = $2, accepted_at = NOW(), invite_token_hash = NULL
		WHERE invite_token_hash = $1
		  AND accepted_at IS NULL
		  AND revoked_at IS NULL
		RETURNING id, workspace_id, user_id, COALESCE(email,''), role
	`, tokenHash, userID).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Email, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

// RevokeMember soft-deletes a member.
func (r *WorkspaceRepo) RevokeMember(ctx context.Context, workspaceID, memberID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE workspace_members SET revoked_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND role != 'admin'
	`, memberID, workspaceID)
	return err
}

// SetStripeCustomerID updates the Stripe customer ID for a workspace.
func (r *WorkspaceRepo) SetStripeCustomerID(ctx context.Context, workspaceID, customerID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE workspaces SET stripe_customer_id = $2 WHERE id = $1
	`, workspaceID, customerID)
	return err
}

// SetSubscription updates subscription fields after a Stripe webhook.
func (r *WorkspaceRepo) SetSubscription(ctx context.Context, workspaceID, subscriptionID, plan string, periodEnd int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE workspaces
		SET stripe_subscription_id = $2,
		    plan = $3,
		    subscription_expires_at = to_timestamp($4)
		WHERE id = $1
	`, workspaceID, subscriptionID, plan, periodEnd)
	return err
}

// ListAll returns all workspaces ordered by creation date (superadmin use only).
func (r *WorkspaceRepo) ListAll(ctx context.Context, limit, offset int) ([]Workspace, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(cui,''), COALESCE(address,''), COALESCE(phone,''), COALESCE(logo_s3_key,''),
		       plan, label_quota_monthly, label_quota_used,
		       COALESCE(stripe_customer_id,''), COALESCE(stripe_subscription_id,''),
		       subscription_expires_at, created_at
		FROM workspaces
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []Workspace
	for rows.Next() {
		var ws Workspace
		if err := rows.Scan(
			&ws.ID, &ws.Name, &ws.CUI, &ws.Address, &ws.Phone, &ws.LogoS3Key,
			&ws.Plan, &ws.LabelQuotaMonthly, &ws.LabelQuotaUsed,
			&ws.StripeCustomerID, &ws.StripeSubscriptionID,
			&ws.SubscriptionPeriodEnd, &ws.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, ws)
	}
	return result, total, rows.Err()
}

// ListExpiringSubscriptions returns workspaces whose subscription ends before cutoff.
func (r *WorkspaceRepo) ListExpiringSubscriptions(ctx context.Context, cutoff time.Time) ([]Workspace, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(cui,''), COALESCE(address,''), COALESCE(phone,''), COALESCE(logo_s3_key,''),
		       plan, label_quota_monthly, label_quota_used,
		       COALESCE(stripe_customer_id,''), COALESCE(stripe_subscription_id,''),
		       subscription_expires_at, created_at
		FROM workspaces
		WHERE subscription_expires_at <= $1
		  AND stripe_subscription_id != ''
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Workspace
	for rows.Next() {
		var ws Workspace
		if err := rows.Scan(
			&ws.ID, &ws.Name, &ws.CUI, &ws.Address, &ws.Phone, &ws.LogoS3Key,
			&ws.Plan, &ws.LabelQuotaMonthly, &ws.LabelQuotaUsed,
			&ws.StripeCustomerID, &ws.StripeSubscriptionID,
			&ws.SubscriptionPeriodEnd, &ws.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, ws)
	}
	return result, rows.Err()
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
