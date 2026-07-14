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

type libreTranslateProvider struct {
	name       string
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func newLibreTranslate(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "https://libretranslate.com"
	}
	apiKey := cfg.String("api_key")
	return &libreTranslateProvider{
		name:       name,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: httpClient(timeout),
	}, nil
}

func (p *libreTranslateProvider) Name() string { return p.name }

func (p *libreTranslateProvider) Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error) {
	source := req.SourceLang
	if source == "" || source == "auto" {
		source = "auto"
	}
	body := map[string]any{
		"q":       req.Text,
		"source":  source,
		"target":  req.TargetLang,
		"format":  "text",
		"api_key": p.apiKey,
	}
	if req.Alternatives > 0 {
		body["alternatives"] = req.Alternatives
	}
	data, status, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/translate", nil, body)
	if err != nil {
		if status == 429 {
			return nil, fmt.Errorf("%w: libretranslate rate limited", common.ErrRateLimited)
		}
		return nil, fmt.Errorf("libretranslate failed: %w", err)
	}
	var resp libreTranslateResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	sourceOut := req.SourceLang
	if sourceOut == "" || sourceOut == "auto" {
		if resp.DetectedLanguage != nil {
			sourceOut = resp.DetectedLanguage.Language
		} else {
			sourceOut = "auto"
		}
	}
	return &TranslationResponse{
		Translation:  resp.TranslatedText,
		SourceLang:   common.NormalizeLanguage(sourceOut),
		TargetLang:   req.TargetLang,
		Provider:     p.name,
		Model:        "libretranslate",
		Confidence:   0.75,
		Alternatives: resp.Alternatives,
	}, nil
}

func (p *libreTranslateProvider) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	body := map[string]any{
		"q":       text,
		"api_key": p.apiKey,
	}
	data, _, err := doJSON(ctx, p.httpClient, "POST", p.baseURL+"/detect", nil, body)
	if err != nil {
		return "", 0, err
	}
	var list []struct {
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return "", 0, err
	}
	if len(list) == 0 {
		return "", 0, fmt.Errorf("no detection result")
	}
	return common.NormalizeLanguage(list[0].Language), list[0].Confidence, nil
}

func (p *libreTranslateProvider) Languages(ctx context.Context) ([]Language, error) {
	data, _, err := doJSON(ctx, p.httpClient, "GET", p.baseURL+"/languages", nil, nil)
	if err != nil {
		return nil, err
	}
	var list []struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	langs := make([]Language, 0, len(list))
	for _, l := range list {
		langs = append(langs, Language{Code: l.Code, Name: l.Name})
	}
	return langs, nil
}

func (p *libreTranslateProvider) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/languages", nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("languages endpoint returned %d", resp.StatusCode)
	}
	return nil
}

type libreTranslateResponse struct {
	TranslatedText   string   `json:"translatedText"`
	Alternatives     []string `json:"alternatives"`
	DetectedLanguage *struct {
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	} `json:"detectedLanguage"`
}
