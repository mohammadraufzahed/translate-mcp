package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/common"
	"github.com/mohammadraufzahed/translate-mcp/internal/config"
)

type customProvider struct {
	openAIProvider
}

func newCustom(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	apiKey := cfg.String("api_key")
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		return nil, fmt.Errorf("custom provider requires base_url")
	}
	model := cfg.String("default_model")
	if model == "" {
		model = "default"
	}
	return &customProvider{
		openAIProvider: openAIProvider{
			name:       name,
			apiKey:     apiKey,
			baseURL:    strings.TrimSuffix(baseURL, "/"),
			model:      model,
			httpClient: httpClient(timeout),
		},
	}, nil
}

func (p *customProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	model := commonFirst(req.Model, p.model)
	masked, mapping := req.Text, map[string]string{}
	if req.Glossary != nil {
		masked, mapping = req.Glossary.Apply(req.Text, req.TargetLang)
	}
	prompt := buildPrompt(req, masked, mapping)
	body := map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.3,
		"max_tokens":  outputBudget(masked),
	}
	headers := map[string]string{}
	if p.apiKey != "" {
		headers["Authorization"] = "Bearer " + p.apiKey
	}
	data, status, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/chat/completions", headers, body)
	if err != nil {
		if status == 429 {
			return nil, fmt.Errorf("%w: custom rate limited", common.ErrRateLimited)
		}
		return nil, fmt.Errorf("custom translate failed: %w", err)
	}
	var resp openAIChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("custom returned no choices")
	}
	result, err := parseLLMResult([]byte(resp.Choices[0].Message.Content), req.SourceLang, p.name, model)
	if err != nil {
		return nil, err
	}
	result.TargetLang = req.TargetLang
	if mapping != nil {
		result.Translation = req.Glossary.Restore(result.Translation, mapping)
	}
	if resp.Usage != nil {
		result.InputTokens = resp.Usage.PromptTokens
		result.OutputTokens = resp.Usage.CompletionTokens
	}
	return result, nil
}

func (p *customProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	return "", 0, fmt.Errorf("custom provider does not support detect language")
}

func (p *customProvider) Languages(ctx context.Context) ([]Language, error) {
	return commonLanguageList(), nil
}

func (p *customProvider) Health(ctx context.Context) error {
	return nil
}
