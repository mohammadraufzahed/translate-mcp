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

type deeplProvider struct {
	name       string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func newDeepL(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	apiKey := cfg.String("api_key")
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "https://api.deepl.com"
	}
	return &deeplProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: httpClient(timeout),
	}, nil
}

func (p *deeplProvider) Name() string { return p.name }

func (p *deeplProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	body := map[string]any{
		"text":        []string{req.Text},
		"target_lang": strings.ToUpper(req.TargetLang),
	}
	if req.SourceLang != "" && req.SourceLang != "auto" {
		body["source_lang"] = strings.ToUpper(req.SourceLang)
	}
	if req.Context != "" {
		body["context"] = req.Context
	}
	headers := map[string]string{
		"Authorization": "DeepL-Auth-Key " + p.apiKey,
		"Content-Type":  "application/json",
	}
	data, status, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/v2/translate", headers, body)
	if err != nil {
		if status == 429 {
			return nil, fmt.Errorf("%w: deepl rate limited", common.ErrRateLimited)
		}
		return nil, fmt.Errorf("deepl translate failed: %w", err)
	}
	var resp deeplTranslateResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if len(resp.Translations) == 0 {
		return nil, fmt.Errorf("deepl returned no translations")
	}
	t := resp.Translations[0]
	source := req.SourceLang
	if source == "" || source == "auto" {
		source = strings.ToLower(t.DetectedSourceLanguage)
	}
	return &TranslationResponse{
		Translation: t.Text,
		SourceLang:  common.NormalizeLanguage(source),
		TargetLang:  req.TargetLang,
		Provider:    p.name,
		Model:       "deepl",
		Confidence:  0.92,
	}, nil
}

func (p *deeplProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	body := map[string]any{
		"text":        []string{text},
		"target_lang": "EN",
	}
	headers := map[string]string{
		"Authorization": "DeepL-Auth-Key " + p.apiKey,
		"Content-Type":  "application/json",
	}
	data, _, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/v2/translate", headers, body)
	if err != nil {
		return "", 0, err
	}
	var resp deeplTranslateResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", 0, err
	}
	if len(resp.Translations) == 0 {
		return "", 0, fmt.Errorf("no translation returned")
	}
	return common.NormalizeLanguage(strings.ToLower(resp.Translations[0].DetectedSourceLanguage)), 0.95, nil
}

func (p *deeplProvider) Languages(ctx context.Context) ([]Language, error) {
	headers := map[string]string{"Authorization": "DeepL-Auth-Key " + p.apiKey}
	data, _, err := doJSON(ctx, p.httpClient, "GET", p.baseURL+"/v2/languages?type=target", headers, nil)
	if err != nil {
		return nil, err
	}
	var list []deeplLanguage
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	langs := make([]Language, 0, len(list))
	for _, l := range list {
		langs = append(langs, Language{Code: strings.ToLower(l.Language), Name: l.Name})
	}
	return langs, nil
}

func (p *deeplProvider) Health(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("api key not configured")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/v2/usage", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "DeepL-Auth-Key "+p.apiKey)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("usage endpoint returned %d", resp.StatusCode)
	}
	return nil
}

type deeplTranslateResponse struct {
	Translations []struct {
		Text                   string `json:"text"`
		DetectedSourceLanguage string `json:"detected_source_language"`
	} `json:"translations"`
}

type deeplLanguage struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}
