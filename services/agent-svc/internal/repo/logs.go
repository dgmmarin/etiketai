package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CallLogEntry struct {
	WorkspaceID  string    `json:"workspace_id"`
	LabelID      string    `json:"label_id"`
	AgentType    string    `json:"agent_type"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	TokensInput  int       `json:"tokens_input"`
	TokensOutput int       `json:"tokens_output"`
	CostUSD      float64   `json:"cost_usd"`
	LatencyMS    int       `json:"latency_ms"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CalledAt     time.Time `json:"called_at"`
}

type CallLogRepo struct {
	db *pgxpool.Pool
}

func NewCallLogRepo(db *pgxpool.Pool) *CallLogRepo {
	return &CallLogRepo{db: db}
}

func (r *CallLogRepo) Insert(ctx context.Context, e CallLogEntry) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO agent_call_logs (
			workspace_id, label_id, agent_type, provider, model,
			tokens_input, tokens_output, cost_usd, latency_ms,
			success, error_message, called_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`,
		e.WorkspaceID, e.LabelID, e.AgentType, e.Provider, e.Model,
		nilZero(e.TokensInput), nilZero(e.TokensOutput), e.CostUSD, e.LatencyMS,
		e.Success, nilIfEmptyStr(e.ErrorMessage), e.CalledAt,
	)
	return err
}

func (r *CallLogRepo) ListByWorkspace(ctx context.Context, workspaceID string, limit int) ([]CallLogEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT workspace_id, label_id, agent_type, provider, COALESCE(model,''),
		       COALESCE(tokens_input,0), COALESCE(tokens_output,0), cost_usd, latency_ms,
		       success, COALESCE(error_message,''), called_at
		FROM agent_call_logs
		WHERE workspace_id = $1
		ORDER BY called_at DESC
		LIMIT $2
	`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CallLogEntry
	for rows.Next() {
		var e CallLogEntry
		if err := rows.Scan(
			&e.WorkspaceID, &e.LabelID, &e.AgentType, &e.Provider, &e.Model,
			&e.TokensInput, &e.TokensOutput, &e.CostUSD, &e.LatencyMS,
			&e.Success, &e.ErrorMessage, &e.CalledAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func nilZero(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func nilIfEmptyStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
