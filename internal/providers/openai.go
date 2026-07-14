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

type openAIProvider struct {
	name         string
	apiKey       string
	baseURL      string
	model        string
	extraHeaders map[string]string
	httpClient   *http.Client
}

func newOpenAI(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	apiKey := cfg.String("api_key")
	if apiKey == "" {
		apiKey = commonFirst(cfg.String("openai_api_key"), cfg.String("apikey"))
	}
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := cfg.String("default_model")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &openAIProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: httpClient(timeout),
	}, nil
}

func (p *openAIProvider) Name() string { return p.name }

func (p *openAIProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	model := commonFirst(req.Model, p.model)
	masked, mapping := "", map[string]string{}
	if req.Glossary != nil {
		masked, mapping = req.Glossary.Apply(req.Text, req.TargetLang)
	} else {
		masked = req.Text
	}

	prompt := buildPrompt(req, masked, mapping)
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful translation assistant. Output only valid JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.3,
		"max_tokens":      outputBudget(masked),
		"response_format": map[string]string{"type": "json_object"},
	}

	headers := p.authHeaders()
	data, status, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/chat/completions", headers, body)
	if err != nil {
		if status == 429 {
			return nil, fmt.Errorf("%w: %s rate limited", common.ErrRateLimited, p.name)
		}
		return nil, fmt.Errorf("%s translate failed: %w", p.name, err)
	}

	var resp openAIChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("%s decode failed: %w", p.name, err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("%s returned no choices", p.name)
	}
	content := resp.Choices[0].Message.Content
	result, err := parseLLMResult([]byte(content), req.SourceLang, p.name, model)
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

func (p *openAIProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a language detector. Output only valid JSON with keys language and confidence."},
			{"role": "user", "content": "Detect the language of this text and return JSON {\"language\":\"<bcp47 code>\",\"confidence\":0.0-1.0}: " + text},
		},
		"temperature":     0.0,
		"max_tokens":      128,
		"response_format": map[string]string{"type": "json_object"},
	}
	headers := p.authHeaders()
	data, _, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/chat/completions", headers, body)
	if err != nil {
		return "", 0, err
	}
	var resp openAIChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", 0, err
	}
	if len(resp.Choices) == 0 {
		return "", 0, fmt.Errorf("no response")
	}
	var r struct {
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &r); err != nil {
		return "", 0, err
	}
	return common.NormalizeLanguage(r.Language), r.Confidence, nil
}

func (p *openAIProvider) Languages(ctx context.Context) ([]Language, error) {
	return commonLanguageList(), nil
}

func (p *openAIProvider) Health(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("api key not configured")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	for k, v := range p.authHeaders() {
		req.Header.Set(k, v)
	}
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

func (p *openAIProvider) authHeaders() map[string]string {
	h := map[string]string{"Authorization": "Bearer " + p.apiKey}
	for k, v := range p.extraHeaders {
		h[k] = v
	}
	return h
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}
