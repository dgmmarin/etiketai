package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agent/prompts"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/storage"
)

const defaultModel = "claude-sonnet-4-6"
const defaultMaxTokens = 2048

// VisionAgent implements agent.VisionAgent using the Claude API.
type VisionAgent struct {
	client    anthropic.Client
	model     string
	maxTokens int64
	s3        *storage.S3Client
}

func NewVisionAgent(apiKey, model string, s3 *storage.S3Client) *VisionAgent {
	if model == "" {
		model = defaultModel
	}
	return &VisionAgent{
		client:    anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:     model,
		maxTokens: defaultMaxTokens,
		s3:        s3,
	}
}

func (a *VisionAgent) Name() string { return "claude" }

func (a *VisionAgent) EstimateCost(imageBytes int64) float64 {
	// claude-sonnet-4-6: ~$3 per 1M input tokens; image ≈ 1000–1500 tokens
	estimatedTokens := float64(imageBytes) / 750.0 // rough approximation
	return estimatedTokens * 3.0 / 1_000_000
}

func (a *VisionAgent) ExtractFields(ctx context.Context, req agent.VisionRequest) (*agent.VisionResult, error) {
	start := time.Now()

	imageB64, mimeType, err := a.s3.GetImageBase64(ctx, req.ImageS3Key)
	if err != nil {
		return nil, fmt.Errorf("s3 fetch: %w", err)
	}

	prompt := prompts.VisionPrompt(req.TargetLanguage)

	// Build Claude API request
	mediaType := toAnthropicMediaType(mimeType)
	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: a.maxTokens,
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					anthropic.NewImageBlockBase64(string(mediaType), imageB64),
					anthropic.NewTextBlock(prompt),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty response")
	}

	text := msg.Content[0].Text
	result, err := parseVisionJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parse vision json: %w", err)
	}

	result.ProviderUsed = "claude/" + a.model
	result.TokensUsed = int(msg.Usage.InputTokens + msg.Usage.OutputTokens)
	result.LatencyMS = time.Since(start).Milliseconds()

	return result, nil
}

// ─── Internal ─────────────────────────────────────────────────────────────────

func parseVisionJSON(raw string) (*agent.VisionResult, error) {
	// Strip markdown code block if present
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result agent.VisionResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from model: %w\nraw: %s", err, raw[:min(len(raw), 200)])
	}
	return &result, nil
}

func toAnthropicMediaType(mimeType string) anthropic.Base64ImageSourceMediaType {
	switch mimeType {
	case "image/png":
		return anthropic.Base64ImageSourceMediaTypeImagePNG
	case "image/gif":
		return anthropic.Base64ImageSourceMediaTypeImageGIF
	case "image/webp":
		return anthropic.Base64ImageSourceMediaTypeImageWebP
	default:
		return anthropic.Base64ImageSourceMediaTypeImageJPEG
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
