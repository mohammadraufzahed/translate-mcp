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

type ollamaProvider struct {
	name       string
	baseURL    string
	model      string
	httpClient *http.Client
}

func newOllama(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := cfg.String("default_model")
	if model == "" {
		model = "llama3.1"
	}
	return &ollamaProvider{
		name:       name,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      model,
		httpClient: httpClient(timeout),
	}, nil
}

func (p *ollamaProvider) Name() string { return p.name }

func (p *ollamaProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	model := commonFirst(req.Model, p.model)
	masked, mapping := req.Text, map[string]string{}
	if req.Glossary != nil {
		masked, mapping = req.Glossary.Apply(req.Text, req.TargetLang)
	}
	prompt := buildPrompt(req, masked, mapping)
	body := map[string]any{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   false,
		"options": map[string]any{
			"temperature": 0.3,
		},
	}
	data, _, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/api/chat", nil, body)
	if err != nil {
		return nil, fmt.Errorf("ollama translate failed: %w", err)
	}
	var resp ollamaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	content := resp.Message.Content
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

func (p *ollamaProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	body := map[string]any{
		"model":    p.model,
		"messages": []map[string]string{{"role": "user", "content": "Detect the language and return JSON {\"language\":\"<bcp47>\",\"confidence\":0.0-1.0}: " + text}},
		"stream":   false,
		"options":  map[string]any{"temperature": 0.0},
	}
	data, _, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/api/chat", nil, body)
	if err != nil {
		return "", 0, err
	}
	var resp ollamaResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", 0, err
	}
	var r struct {
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(resp.Message.Content), &r); err != nil {
		return "", 0, err
	}
	return common.NormalizeLanguage(r.Language), r.Confidence, nil
}

func (p *ollamaProvider) Languages(ctx context.Context) ([]Language, error) {
	return commonLanguageList(), nil
}

func (p *ollamaProvider) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("api/tags returned %d", resp.StatusCode)
	}
	return nil
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}
