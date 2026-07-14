package translator

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/cache"
	"github.com/mohammadraufzahed/translate-mcp/internal/common"
	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/glossary"
	"github.com/mohammadraufzahed/translate-mcp/internal/memory"
	"github.com/mohammadraufzahed/translate-mcp/internal/metrics"
	"github.com/mohammadraufzahed/translate-mcp/internal/providers"
	"github.com/mohammadraufzahed/translate-mcp/internal/rate"
)

type mockProvider struct {
	name         string
	translations map[string]string
	lang         string
	fails        bool
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Translate(_ context.Context, req providers.TranslationRequest) (*providers.TranslationResponse, error) {
	if m.fails {
		return nil, errors.New("mock failure")
	}
	text := req.Text
	if req.Glossary != nil {
		masked, mapping := req.Glossary.Apply(text, req.TargetLang)
		text = masked
		// simulated "provider" translation: translate placeholders back.
		if tr, ok := m.translations[text]; ok {
			text = tr
		} else {
			text = "translated:" + text
		}
		text = req.Glossary.Restore(text, mapping)
	} else {
		if tr, ok := m.translations[text]; ok {
			text = tr
		} else {
			text = "translated:" + text
		}
	}
	return &providers.TranslationResponse{
		Translation: text,
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
		Provider:    m.name,
		Confidence:  0.9,
	}, nil
}

func (m *mockProvider) DetectLanguage(_ context.Context, text string) (string, float64, error) {
	return m.lang, 0.9, nil
}

func (m *mockProvider) Languages(_ context.Context) ([]providers.Language, error) {
	return []providers.Language{{Code: "en", Name: "English"}}, nil
}

func (m *mockProvider) Health(_ context.Context) error {
	if m.fails {
		return errors.New("unhealthy")
	}
	return nil
}

func newTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Transport: "stdio"},
		Cache: config.CacheConfig{
			DefaultTTL: "1h",
			L1:         config.CacheTierConfig{Type: "memory", MaxEntries: 100},
		},
		Translation: config.TranslationConfig{
			DefaultProvider:       "mock",
			MaxTextLength:         10000,
			MaxBatchItems:         10,
			RequestTimeout:        "60s",
			FallbackChain:         []string{"fallback"},
			GlossaryPreprocessing: true,
		},
	}
}

func newTestService(t *testing.T, cfg *config.Config, provs map[string]providers.Translator) *Service {
	s := &Service{
		cfg:       cfg,
		gloss:     glossary.New(),
		mem:       memory.NewInMemory(),
		providers: provs,
		metrics:   metrics.New(),
		breaker:   make(map[string]*rate.CircuitBreaker),
		semaphore: make(map[string]chan struct{}),
		ratelimit: rate.NewLimiter(1000, 100),
		timeout:   time.Minute,
	}
	for name := range provs {
		s.breaker[name] = rate.NewCircuitBreaker(5, 30*time.Second)
		s.semaphore[name] = make(chan struct{}, 5)
	}
	c, err := cache.Build(cfg.Cache, nil)
	if err != nil {
		t.Fatalf("cache build: %v", err)
	}
	s.cache = c
	return s
}

func TestTranslate(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{
		name:         "mock",
		translations: map[string]string{"hello": "hola"},
		lang:         "en",
	}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})

	resp, err := s.Translate(context.Background(), providers.TranslationRequest{
		Text:       "hello",
		TargetLang: "es",
	})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if resp.Translation != "hola" {
		t.Errorf("translation = %q", resp.Translation)
	}
	if resp.SourceLang != "en" {
		t.Errorf("source lang = %q", resp.SourceLang)
	}
	if resp.Provider != "mock" {
		t.Errorf("provider = %q", resp.Provider)
	}
}

func TestTranslateEmpty(t *testing.T) {
	cfg := newTestConfig()
	s := newTestService(t, cfg, map[string]providers.Translator{})
	_, err := s.Translate(context.Background(), providers.TranslationRequest{Text: "", TargetLang: "es"})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestTranslateCache(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{name: "mock", lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})

	ctx := context.Background()
	_, _ = s.Translate(ctx, providers.TranslationRequest{Text: "hello", TargetLang: "es"})
	resp, err := s.Translate(ctx, providers.TranslationRequest{Text: "hello", TargetLang: "es"})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if !resp.Cached {
		t.Error("expected second request to be cached")
	}
}

func TestTranslateFallback(t *testing.T) {
	cfg := newTestConfig()
	primary := &mockProvider{name: "mock", fails: true, lang: "en"}
	fallback := &mockProvider{name: "fallback", translations: map[string]string{"hello": "salut"}, lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": primary, "fallback": fallback})

	resp, err := s.Translate(context.Background(), providers.TranslationRequest{Text: "hello", TargetLang: "fr"})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if resp.Provider != "fallback" {
		t.Errorf("expected fallback provider, got %q", resp.Provider)
	}
	if resp.Translation != "salut" {
		t.Errorf("translation = %q", resp.Translation)
	}
}

func TestTranslateMaxLength(t *testing.T) {
	cfg := newTestConfig()
	cfg.Translation.MaxTextLength = 2
	s := newTestService(t, cfg, map[string]providers.Translator{})
	_, err := s.Translate(context.Background(), providers.TranslationRequest{Text: "hello", TargetLang: "es"})
	if !errors.Is(err, common.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchTranslateOneToMany(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{name: "mock", lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})

	resps, err := s.BatchTranslate(context.Background(), BatchRequest{
		Mode:       BatchOneToMany,
		Text:       "hello",
		SourceLang: "en",
		Targets:    []string{"es", "fr"},
	})
	if err != nil {
		t.Fatalf("BatchTranslate: %v", err)
	}
	if len(resps) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resps))
	}
}

func TestBatchTranslateManyToOne(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{name: "mock", lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})

	resps, err := s.BatchTranslate(context.Background(), BatchRequest{
		Mode:       BatchManyToOne,
		SourceLang: "en",
		TargetLang: "es",
		Items: []BatchItem{
			{Text: "hello"},
			{Text: "world"},
		},
	})
	if err != nil {
		t.Fatalf("BatchTranslate: %v", err)
	}
	if len(resps) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resps))
	}
}

func TestBatchTooManyItems(t *testing.T) {
	cfg := newTestConfig()
	cfg.Translation.MaxBatchItems = 1
	s := newTestService(t, cfg, map[string]providers.Translator{})
	_, err := s.BatchTranslate(context.Background(), BatchRequest{
		Mode:    BatchOneToMany,
		Targets: []string{"es", "fr"},
	})
	if err == nil {
		t.Fatal("expected error for too many items")
	}
}

func TestTranslateDocumentJSON(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{name: "mock", lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})

	out, err := s.TranslateDocument(context.Background(), `{"greeting":"hello"}`, "json", "es", "en", "", "", "", "")
	if err != nil {
		t.Fatalf("TranslateDocument: %v", err)
	}
	var data map[string]string
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if !strings.HasPrefix(data["greeting"], "translated:") {
		t.Errorf("expected translated greeting, got %q", data["greeting"])
	}
}

func TestListLanguages(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{name: "mock", lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})

	langs, err := s.ListLanguages(context.Background(), "")
	if err != nil {
		t.Fatalf("ListLanguages: %v", err)
	}
	if len(langs) == 0 {
		t.Fatal("expected languages")
	}
}

func TestHealth(t *testing.T) {
	cfg := newTestConfig()
	healthy := &mockProvider{name: "healthy"}
	unhealthy := &mockProvider{name: "unhealthy", fails: true}
	s := newTestService(t, cfg, map[string]providers.Translator{"healthy": healthy, "unhealthy": unhealthy})

	status := s.Health(context.Background())
	if status["healthy"] != "healthy" {
		t.Errorf("healthy status = %q", status["healthy"])
	}
	if status["unhealthy"] == "healthy" {
		t.Errorf("unhealthy status = %q", status["unhealthy"])
	}
}

func TestGlossaryPreprocessing(t *testing.T) {
	cfg := newTestConfig()
	mock := &mockProvider{name: "mock", lang: "en"}
	s := newTestService(t, cfg, map[string]providers.Translator{"mock": mock})
	s.gloss.Add(glossary.Entry{SourceTerm: "MCP", TargetLang: "es", Translation: "MCP", CaseSensitive: true})

	resp, err := s.Translate(context.Background(), providers.TranslationRequest{
		Text:       "Welcome to MCP",
		SourceLang: "en",
		TargetLang: "es",
	})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if !strings.Contains(resp.Translation, "MCP") {
		t.Errorf("expected preserved term MCP, got %q", resp.Translation)
	}
	if strings.Contains(resp.Translation, "__TERM_") {
		t.Errorf("placeholder leaked into output: %q", resp.Translation)
	}
}
