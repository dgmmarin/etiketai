package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent/prompts"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/storage"
)

// VisionAgent implements agent.VisionAgent using a local Ollama server.
// Used for on-premise deployments where data cannot leave the client's network.
type VisionAgent struct {
	baseURL string
	model   string
	timeout time.Duration
	s3      *storage.S3Client
	http    *http.Client
}

func NewVisionAgent(baseURL, model string, s3 *storage.S3Client) *VisionAgent {
	if model == "" {
		model = "llava:13b"
	}
	timeout := 120 * time.Second
	return &VisionAgent{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		timeout: timeout,
		s3:      s3,
		http:    &http.Client{Timeout: timeout},
	}
}

func (a *VisionAgent) Name() string              { return "ollama" }
func (a *VisionAgent) EstimateCost(_ int64) float64 { return 0.0 } // self-hosted = free

func (a *VisionAgent) ExtractFields(ctx context.Context, req agent.VisionRequest) (*agent.VisionResult, error) {
	start := time.Now()

	imageB64, _, err := a.s3.GetImageBase64(ctx, req.ImageS3Key)
	if err != nil {
		return nil, fmt.Errorf("s3 fetch: %w", err)
	}

	prompt := prompts.VisionPrompt(req.TargetLanguage)

	payload := map[string]any{
		"model":  a.model,
		"prompt": prompt,
		"images": []string{imageB64},
		"stream": false,
		"options": map[string]any{
			"temperature": 0.1, // deterministic output
			"num_ctx":     4096,
		},
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	result, err := parseVisionJSON(ollamaResp.Response)
	if err != nil {
		return nil, fmt.Errorf("parse vision json: %w", err)
	}

	result.ProviderUsed = "ollama/" + a.model
	result.LatencyMS = time.Since(start).Milliseconds()
	return result, nil
}

func parseVisionJSON(raw string) (*agent.VisionResult, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	var result agent.VisionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &result, nil
}
