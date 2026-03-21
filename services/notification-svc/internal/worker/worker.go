package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/notification-svc/internal/email"
)

// Task type constants — shared with enqueueing services.
const (
	TaskSendEmail          = "email:send"
	TaskVerifyEmail        = "email:verify"
	TaskResetPassword      = "email:reset_password"
	TaskWorkspaceInvite    = "email:workspace_invite"
	TaskSubscriptionExpiry = "email:subscription_expiry"
)

// ── Generic send task ─────────────────────────────────────────────────────────

type SendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

// ── Typed task payloads ───────────────────────────────────────────────────────

type VerifyEmailPayload struct {
	To         string `json:"to"`
	Name       string `json:"name"`
	VerifyURL  string `json:"verify_url"`
}

type ResetPasswordPayload struct {
	To       string `json:"to"`
	Name     string `json:"name"`
	ResetURL string `json:"reset_url"`
}

type WorkspaceInvitePayload struct {
	To            string `json:"to"`
	InviterName   string `json:"inviter_name"`
	WorkspaceName string `json:"workspace_name"`
	InviteURL     string `json:"invite_url"`
}

type SubscriptionExpiryPayload struct {
	To            string `json:"to"`
	WorkspaceName string `json:"workspace_name"`
	DaysLeft      int    `json:"days_left"`
}

// ── Task constructors ─────────────────────────────────────────────────────────

func NewVerifyEmailTask(p VerifyEmailPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskVerifyEmail, data, asynq.MaxRetry(5)), nil
}

func NewResetPasswordTask(p ResetPasswordPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskResetPassword, data, asynq.MaxRetry(3)), nil
}

func NewWorkspaceInviteTask(p WorkspaceInvitePayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskWorkspaceInvite, data, asynq.MaxRetry(3)), nil
}

func NewSubscriptionExpiryTask(p SubscriptionExpiryPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskSubscriptionExpiry, data, asynq.MaxRetry(2)), nil
}

// ── Handler ───────────────────────────────────────────────────────────────────

// Handler processes email notification tasks.
type Handler struct {
	email  *email.Client
	logger *zap.Logger
}

func NewHandler(emailClient *email.Client, logger *zap.Logger) *Handler {
	return &Handler{email: emailClient, logger: logger}
}

// RegisterMux registers all email task handlers into an Asynq mux.
func RegisterMux(mux *asynq.ServeMux, h *Handler) {
	mux.HandleFunc(TaskSendEmail, h.HandleSend)
	mux.HandleFunc(TaskVerifyEmail, h.HandleVerifyEmail)
	mux.HandleFunc(TaskResetPassword, h.HandleResetPassword)
	mux.HandleFunc(TaskWorkspaceInvite, h.HandleWorkspaceInvite)
	mux.HandleFunc(TaskSubscriptionExpiry, h.HandleSubscriptionExpiry)
}

func (h *Handler) HandleSend(ctx context.Context, t *asynq.Task) error {
	var p SendEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return h.send(ctx, p.To, email.SendParams{Subject: p.Subject, HTML: p.HTML, Text: p.Text})
}

func (h *Handler) HandleVerifyEmail(ctx context.Context, t *asynq.Task) error {
	var p VerifyEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return h.send(ctx, p.To, email.VerifyEmail(p.Name, p.VerifyURL))
}

func (h *Handler) HandleResetPassword(ctx context.Context, t *asynq.Task) error {
	var p ResetPasswordPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return h.send(ctx, p.To, email.ResetPassword(p.Name, p.ResetURL))
}

func (h *Handler) HandleWorkspaceInvite(ctx context.Context, t *asynq.Task) error {
	var p WorkspaceInvitePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return h.send(ctx, p.To, email.WorkspaceInvite(p.InviterName, p.WorkspaceName, p.InviteURL))
}

func (h *Handler) HandleSubscriptionExpiry(ctx context.Context, t *asynq.Task) error {
	var p SubscriptionExpiryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return h.send(ctx, p.To, email.SubscriptionExpiringSoon(p.WorkspaceName, p.DaysLeft))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) send(ctx context.Context, to string, params email.SendParams) error {
	if !h.email.IsConfigured() {
		h.logger.Warn("Resend not configured — email dropped",
			zap.String("to", to),
			zap.String("subject", params.Subject),
		)
		return nil // not an error; avoids retries in dev
	}
	params.To = to
	id, err := h.email.Send(ctx, params)
	if err != nil {
		h.logger.Error("email send failed", zap.String("to", to), zap.Error(err))
		return err
	}
	h.logger.Info("email sent", zap.String("to", to), zap.String("message_id", id))
	return nil
}
