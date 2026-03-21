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
)

// TranslationAgent implements agent.TranslationAgent using the Claude API.
type TranslationAgent struct {
	client    anthropic.Client
	model     string
	maxTokens int64
}

func NewTranslationAgent(apiKey, model string) *TranslationAgent {
	if model == "" {
		model = defaultModel
	}
	return &TranslationAgent{
		client:    anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:     model,
		maxTokens: 4096,
	}
}

func (a *TranslationAgent) Name() string { return "claude" }
func (a *TranslationAgent) SupportedLanguages() []string {
	return []string{"*"} // Claude supports all languages
}

func (a *TranslationAgent) Translate(ctx context.Context, req agent.TranslRequest) (*agent.TranslResult, error) {
	start := time.Now()

	// Serialize fields to JSON for the prompt
	fieldsJSON, err := json.Marshal(req.Fields)
	if err != nil {
		return nil, fmt.Errorf("marshal fields: %w", err)
	}

	systemPrompt := prompts.TranslationSystemPrompt()
	userPrompt := prompts.TranslationUserPrompt(string(fieldsJSON), string(req.Category), req.SourceLang)

	msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: a.maxTokens,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{
			{
				Role:    anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(userPrompt)},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api: %w", err)
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty response")
	}

	text := strings.TrimSpace(msg.Content[0].Text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 2 {
			text = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var translated map[string]*string
	if err := json.Unmarshal([]byte(text), &translated); err != nil {
		return nil, fmt.Errorf("parse translation json: %w", err)
	}

	return &agent.TranslResult{
		Translated:   translated,
		ProviderUsed: "claude/" + a.model,
		TokensUsed:   int(msg.Usage.InputTokens + msg.Usage.OutputTokens),
		LatencyMS:    time.Since(start).Milliseconds(),
	}, nil
}
