package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/common"
	"github.com/mohammadraufzahed/translate-mcp/internal/config"
)

type googleProvider struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func newGoogle(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	apiKey := cfg.String("api_key")
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	model := cfg.String("default_model")
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &googleProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      model,
		httpClient: httpClient(timeout),
	}, nil
}

func (p *googleProvider) Name() string { return p.name }

func (p *googleProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	model := commonFirst(req.Model, p.model)
	masked, mapping := req.Text, map[string]string{}
	if req.Glossary != nil {
		masked, mapping = req.Glossary.Apply(req.Text, req.TargetLang)
	}
	prompt := buildPrompt(req, masked, mapping)
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":      0.3,
			"maxOutputTokens":  outputBudget(masked),
			"responseMimeType": "application/json",
		},
	}
	data, status, err := doJSON(ctx, p.httpClient, "POST", url, nil, body)
	if err != nil {
		if status == 429 {
			return nil, fmt.Errorf("%w: google rate limited", common.ErrRateLimited)
		}
		return nil, fmt.Errorf("google translate failed: %w", err)
	}
	content, err := extractGeminiContent(data)
	if err != nil {
		return nil, err
	}
	result, err := parseLLMResult([]byte(content), req.SourceLang, p.name, model)
	if err != nil {
		return nil, err
	}
	result.TargetLang = req.TargetLang
	if mapping != nil {
		result.Translation = req.Glossary.Restore(result.Translation, mapping)
	}
	return result, nil
}

func (p *googleProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": "Detect the language and return JSON {\"language\":\"<bcp47>\",\"confidence\":0.0-1.0}: " + text}}},
		},
		"generationConfig": map[string]any{
			"temperature":      0.0,
			"maxOutputTokens":  128,
			"responseMimeType": "application/json",
		},
	}
	data, _, err := doJSON(ctx, p.httpClient, "POST", url, nil, body)
	if err != nil {
		return "", 0, err
	}
	content, err := extractGeminiContent(data)
	if err != nil {
		return "", 0, err
	}
	var r struct {
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		return "", 0, err
	}
	return common.NormalizeLanguage(r.Language), r.Confidence, nil
}

func (p *googleProvider) Languages(ctx context.Context) ([]Language, error) {
	return commonLanguageList(), nil
}

func (p *googleProvider) Health(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("api key not configured")
	}
	url := fmt.Sprintf("%s/models?key=%s", p.baseURL, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
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

func extractGeminiContent(data []byte) (string, error) {
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", err
	}
	if len(r.Candidates) == 0 {
		return "", fmt.Errorf("no candidates")
	}
	parts := r.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return "", fmt.Errorf("no content parts")
	}
	return parts[0].Text, nil
}
