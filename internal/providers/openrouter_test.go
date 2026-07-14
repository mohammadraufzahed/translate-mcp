package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
)

func TestOpenRouterTranslate(t *testing.T) {
	var gotModel string
	var gotBody map[string]any
	var gotHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotHeaders = r.Header
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		gotModel, _ = gotBody["model"].(string)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"translation":"hola","source_language":"en","confidence":0.95}`}},
			},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 2},
		})
	}))
	defer srv.Close()

	cfg := config.ProviderConfig{
		"api_key":       "test-key",
		"base_url":      srv.URL,
		"default_model": "openai/gpt-4o-mini",
		"site_url":      "https://example.com",
		"site_name":     "test-app",
	}
	p, err := newOpenRouter("openrouter", cfg, 5*time.Second)
	if err != nil {
		t.Fatalf("newOpenRouter: %v", err)
	}

	resp, err := p.Translate(context.Background(), TranslationRequest{
		Text:       "hello",
		SourceLang: "en",
		TargetLang: "es",
	})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}

	if resp.Translation != "hola" {
		t.Errorf("translation = %q, want hola", resp.Translation)
	}
	if resp.Provider != "openrouter" {
		t.Errorf("provider = %q, want openrouter", resp.Provider)
	}
	if gotModel != "openai/gpt-4o-mini" {
		t.Errorf("model = %q, want openai/gpt-4o-mini", gotModel)
	}
	if auth := gotHeaders.Get("Authorization"); auth != "Bearer test-key" {
		t.Errorf("Authorization = %q", auth)
	}
	if gotHeaders.Get("HTTP-Referer") != "https://example.com" {
		t.Errorf("HTTP-Referer = %q", gotHeaders.Get("HTTP-Referer"))
	}
	if gotHeaders.Get("X-Title") != "test-app" {
		t.Errorf("X-Title = %q", gotHeaders.Get("X-Title"))
	}
}

func TestOpenRouterMissingAPIKey(t *testing.T) {
	cfg := config.ProviderConfig{}
	if _, err := newOpenRouter("openrouter", cfg, 5*time.Second); err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestOpenRouterDetectLanguage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"language":"es","confidence":0.97}`}},
			},
		})
	}))
	defer srv.Close()

	cfg := config.ProviderConfig{"api_key": "k", "base_url": srv.URL}
	p, err := newOpenRouter("openrouter", cfg, 5*time.Second)
	if err != nil {
		t.Fatalf("newOpenRouter: %v", err)
	}

	lang, conf, err := p.DetectLanguage(context.Background(), "hola")
	if err != nil {
		t.Fatalf("DetectLanguage: %v", err)
	}
	if lang != "es" {
		t.Errorf("language = %q, want es", lang)
	}
	if conf != 0.97 {
		t.Errorf("confidence = %v, want 0.97", conf)
	}
}

func TestOpenRouterHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := config.ProviderConfig{"api_key": "k", "base_url": srv.URL}
	p, err := newOpenRouter("openrouter", cfg, 5*time.Second)
	if err != nil {
		t.Fatalf("newOpenRouter: %v", err)
	}

	if err := p.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestOpenRouterRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	cfg := config.ProviderConfig{"api_key": "k", "base_url": srv.URL}
	p, err := newOpenRouter("openrouter", cfg, 5*time.Second)
	if err != nil {
		t.Fatalf("newOpenRouter: %v", err)
	}

	_, err = p.Translate(context.Background(), TranslationRequest{
		Text:       "hello",
		SourceLang: "en",
		TargetLang: "es",
	})
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}
