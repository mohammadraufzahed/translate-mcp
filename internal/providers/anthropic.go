package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/common"
	"github.com/mohammadraufzahed/translate-mcp/internal/config"
)

type anthropicProvider struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func newAnthropic(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	apiKey := cfg.String("api_key")
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	model := cfg.String("default_model")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &anthropicProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: httpClient(timeout),
	}, nil
}

func (p *anthropicProvider) Name() string { return p.name }

func (p *anthropicProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	model := commonFirst(req.Model, p.model)
	masked, mapping := req.Text, map[string]string{}
	if req.Glossary != nil {
		masked, mapping = req.Glossary.Apply(req.Text, req.TargetLang)
	}
	prompt := buildPrompt(req, masked, mapping)
	body := map[string]any{
		"model":      model,
		"max_tokens": outputBudget(masked),
		"system":     "You are a helpful translation assistant. Output only valid JSON.",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}
	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
		"content-type":      "application/json",
	}
	data, status, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/messages", headers, body)
	if err != nil {
		if status == 429 {
			return nil, fmt.Errorf("%w: anthropic rate limited", common.ErrRateLimited)
		}
		return nil, fmt.Errorf("anthropic translate failed: %w", err)
	}
	var resp anthropicResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("anthropic returned no content")
	}
	result, err := parseLLMResult([]byte(resp.Content[0].Text), req.SourceLang, p.name, model)
	if err != nil {
		return nil, err
	}
	result.TargetLang = req.TargetLang
	if mapping != nil {
		result.Translation = req.Glossary.Restore(result.Translation, mapping)
	}
	if resp.Usage != nil {
		result.InputTokens = resp.Usage.InputTokens
		result.OutputTokens = resp.Usage.OutputTokens
	}
	return result, nil
}

func (p *anthropicProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	body := map[string]any{
		"model":      p.model,
		"max_tokens": 128,
		"system":     "You are a language detector. Output only valid JSON with keys language and confidence.",
		"messages": []map[string]string{
			{"role": "user", "content": "Detect the language and return JSON {\"language\":\"<bcp47>\",\"confidence\":0.0-1.0}: " + text},
		},
	}
	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
		"content-type":      "application/json",
	}
	data, _, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/messages", headers, body)
	if err != nil {
		return "", 0, err
	}
	var resp anthropicResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", 0, err
	}
	if len(resp.Content) == 0 {
		return "", 0, fmt.Errorf("no response")
	}
	var r struct {
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(resp.Content[0].Text), &r); err != nil {
		return "", 0, err
	}
	return common.NormalizeLanguage(r.Language), r.Confidence, nil
}

func (p *anthropicProvider) Languages(ctx context.Context) ([]Language, error) {
	return commonLanguageList(), nil
}

func (p *anthropicProvider) Health(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("api key not configured")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("models endpoint returned %d", resp.StatusCode)
	}
	return nil
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage *struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
}
