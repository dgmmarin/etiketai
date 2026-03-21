package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CallLogEntry struct {
	WorkspaceID  string
	LabelID      string
	AgentType    string
	Provider     string
	Model        string
	TokensInput  int
	TokensOutput int
	CostUSD      float64
	LatencyMS    int
	Success      bool
	ErrorMessage string
	CalledAt     time.Time
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
