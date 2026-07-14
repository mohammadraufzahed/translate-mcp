package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
)

type openrouterProvider struct {
	openAIProvider
}

func newOpenRouter(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	apiKey := cfg.String("api_key")
	if apiKey == "" {
		apiKey = cfg.String("openrouter_api_key")
	}
	baseURL := cfg.String("base_url")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	model := cfg.String("default_model")
	if model == "" {
		model = "openai/gpt-4o-mini"
	}

	extra := map[string]string{}
	if siteURL := cfg.String("site_url"); siteURL != "" {
		extra["HTTP-Referer"] = siteURL
	}
	if siteName := cfg.String("site_name"); siteName != "" {
		extra["X-Title"] = siteName
	}

	if apiKey == "" {
		return nil, fmt.Errorf("openrouter provider requires api_key")
	}

	return &openrouterProvider{
		openAIProvider: openAIProvider{
			name:         name,
			apiKey:       apiKey,
			baseURL:      baseURL,
			model:        model,
			extraHeaders: extra,
			httpClient:   httpClient(timeout),
		},
	}, nil
}
