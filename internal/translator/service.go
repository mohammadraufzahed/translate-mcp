package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
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

type Service struct {
	cfg       *config.Config
	cache     cache.Cache
	gloss     *glossary.Glossary
	mem       memory.Manager
	providers map[string]providers.Translator
	metrics   *metrics.Metrics
	breaker   map[string]*rate.CircuitBreaker
	semaphore map[string]chan struct{}
	ratelimit *rate.Limiter
	timeout   time.Duration
}

func New(cfg *config.Config) (*Service, error) {
	timeout, err := time.ParseDuration(cfg.Translation.RequestTimeout)
	if err != nil {
		timeout = 60 * time.Second
	}
	s := &Service{
		cfg:       cfg,
		gloss:     glossary.New(),
		mem:       memory.NewInMemory(),
		providers: make(map[string]providers.Translator),
		metrics:   metrics.New(),
		breaker:   make(map[string]*rate.CircuitBreaker),
		semaphore: make(map[string]chan struct{}),
		ratelimit: rate.NewLimiter(100, 20),
		timeout:   timeout,
	}
	for name, pc := range cfg.Providers {
		t, err := providers.Build(name, pc, timeout)
		if err != nil {
			return nil, fmt.Errorf("build provider %s: %w", name, err)
		}
		s.providers[name] = t
		s.breaker[name] = rate.NewCircuitBreaker(5, 30*time.Second)
		s.semaphore[name] = make(chan struct{}, 5)
	}
	s.cache, err = cache.Build(cfg.Cache, nil)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error {
	return s.cache.Close()
}

func (s *Service) Metrics() *metrics.Metrics    { return s.metrics }
func (s *Service) Cache() cache.Cache           { return s.cache }
func (s *Service) Glossary() *glossary.Glossary { return s.gloss }
func (s *Service) Memory() memory.Manager       { return s.mem }

func (s *Service) Translate(ctx context.Context, req providers.TranslationRequest) (*providers.TranslationResponse, error) {
	text := common.NormalizeText(req.Text)
	if text == "" {
		return nil, fmt.Errorf("%w: text is empty", common.ErrInvalidInput)
	}
	if len([]rune(text)) > s.cfg.Translation.MaxTextLength {
		return nil, fmt.Errorf("%w: text exceeds max length", common.ErrInvalidInput)
	}

	req.Text = text
	providerName := commonFirst(req.Provider, s.cfg.Translation.DefaultProvider)
	chain := s.resolveFallback(providerName)

	if req.SourceLang == "" || req.SourceLang == "auto" {
		detected, _, err := s.DetectLanguage(ctx, text)
		if err == nil && detected != "" && detected != "auto" {
			req.SourceLang = detected
		} else {
			req.SourceLang = "auto"
		}
	}
	req.SourceLang = common.NormalizeLanguage(req.SourceLang)
	req.TargetLang = common.NormalizeLanguage(req.TargetLang)

	model := ""
	if pc, ok := s.cfg.Providers[providerName]; ok {
		model = pc.String("default_model")
	}
	model = commonFirst(req.Model, model)

	var gl *glossary.Glossary
	if s.cfg.Translation.GlossaryPreprocessing {
		gl = s.gloss
	}
	cacheKey := cache.Key(req.Text, req.SourceLang, req.TargetLang, providerName, model, req.Context, req.Tone, gl.Version())

	timer := s.metrics.RequestDuration.WithLabelValues("translate")
	start := time.Now()
	defer func() { timer.Observe(time.Since(start).Seconds()) }()

	if item, hit, _ := s.cache.Get(ctx, cacheKey); hit {
		s.metrics.CacheHitsTotal.WithLabelValues("exact").Inc()
		resp := &item.Response
		resp.Cached = true
		return resp, nil
	}

	var lastErr error
	for _, p := range chain {
		resp, err := s.callProvider(ctx, p, req, gl)
		if err == nil {
			resp.Cached = false
			s.metrics.ProviderCallsTotal.WithLabelValues(p.Name()).Inc()
			s.metrics.InputTokensTotal.WithLabelValues(p.Name()).Add(float64(common.EstimateTokens(req.Text)))
			s.metrics.OutputTokensTotal.WithLabelValues(p.Name()).Add(float64(common.EstimateTokens(resp.Translation)))
			_ = s.cache.Set(ctx, cacheKey, &cache.Item{Response: *resp, CreatedAt: time.Now()}, cache.ParseTTL(s.cfg.Cache.DefaultTTL, 24*time.Hour))
			return resp, nil
		}
		lastErr = err
		s.metrics.ErrorsTotal.WithLabelValues(p.Name()).Inc()
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("%w: no provider available", common.ErrProviderUnavailable)
}

func (s *Service) callProvider(ctx context.Context, p providers.Translator, req providers.TranslationRequest, gl *glossary.Glossary) (*providers.TranslationResponse, error) {
	cb := s.breaker[p.Name()]
	if !cb.Allow() {
		return nil, fmt.Errorf("%w: circuit breaker open for %s", common.ErrProviderUnavailable, p.Name())
	}
	if err := s.ratelimit.Wait(ctx, p.Name()); err != nil {
		return nil, err
	}

	sem := s.semaphore[p.Name()]
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	r := req
	r.Glossary = gl
	resp, err := p.Translate(ctx, r)
	if err != nil {
		cb.RecordFailure()
		return nil, err
	}
	cb.RecordSuccess()
	if resp.SourceLang == "" || resp.SourceLang == "auto" {
		resp.SourceLang = req.SourceLang
	}
	return resp, nil
}

func (s *Service) resolveFallback(name string) []providers.Translator {
	out := make([]providers.Translator, 0)
	if p, ok := s.providers[name]; ok {
		out = append(out, p)
	}
	for _, n := range s.cfg.Translation.FallbackChain {
		if n == name {
			continue
		}
		if p, ok := s.providers[n]; ok {
			out = append(out, p)
		}
	}
	return out
}

func (s *Service) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	text = common.NormalizeText(text)
	chain := s.resolveFallback(s.cfg.Translation.DefaultProvider)
	for _, p := range chain {
		lang, conf, err := p.DetectLanguage(ctx, text)
		if err == nil && lang != "" {
			return common.NormalizeLanguage(lang), conf, nil
		}
	}
	return "", 0, fmt.Errorf("%w: language detection failed", common.ErrProviderUnavailable)
}

func (s *Service) ListLanguages(ctx context.Context, provider string) ([]providers.Language, error) {
	if provider == "" {
		provider = s.cfg.Translation.DefaultProvider
	}
	p, ok := s.providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: provider %s", common.ErrNotFound, provider)
	}
	return p.Languages(ctx)
}

type BatchMode int

const (
	BatchOneToMany BatchMode = iota
	BatchManyToOne
)

type BatchItem struct {
	Text       string `json:"text"`
	TargetLang string `json:"target_language"`
}

type BatchRequest struct {
	Mode       BatchMode
	Text       string
	SourceLang string
	TargetLang string
	Targets    []string
	Items      []BatchItem
	Provider   string
	Model      string
	Context    string
	Tone       string
}

func (s *Service) BatchTranslate(ctx context.Context, req BatchRequest) ([]*providers.TranslationResponse, error) {
	s.metrics.RequestsTotal.WithLabelValues("batch_translate").Inc()
	if req.Mode == BatchOneToMany {
		if len(req.Targets) == 0 {
			return nil, fmt.Errorf("%w: targets required", common.ErrInvalidInput)
		}
		if len(req.Targets) > s.cfg.Translation.MaxBatchItems {
			return nil, fmt.Errorf("%w: too many targets", common.ErrInvalidInput)
		}
		results := make([]*providers.TranslationResponse, len(req.Targets))
		var wg sync.WaitGroup
		errCh := make(chan error, len(req.Targets))
		for i, target := range req.Targets {
			wg.Add(1)
			go func(idx int, tgt string) {
				defer wg.Done()
				r := providers.TranslationRequest{
					Text:         req.Text,
					SourceLang:   req.SourceLang,
					TargetLang:   tgt,
					Provider:     req.Provider,
					Model:        req.Model,
					Context:      req.Context,
					Tone:         req.Tone,
					Alternatives: 0,
				}
				resp, err := s.Translate(ctx, r)
				if err != nil {
					errCh <- err
					return
				}
				results[idx] = resp
			}(i, target)
		}
		wg.Wait()
		close(errCh)
		if len(errCh) > 0 {
			return nil, <-errCh
		}
		return results, nil
	}

	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: items required", common.ErrInvalidInput)
	}
	if len(req.Items) > s.cfg.Translation.MaxBatchItems {
		return nil, fmt.Errorf("%w: too many items", common.ErrInvalidInput)
	}
	results := make([]*providers.TranslationResponse, len(req.Items))
	var wg sync.WaitGroup
	errCh := make(chan error, len(req.Items))
	for i, item := range req.Items {
		wg.Add(1)
		go func(idx int, it BatchItem) {
			defer wg.Done()
			target := it.TargetLang
			if target == "" {
				target = req.TargetLang
			}
			r := providers.TranslationRequest{
				Text:         it.Text,
				SourceLang:   req.SourceLang,
				TargetLang:   target,
				Provider:     req.Provider,
				Model:        req.Model,
				Context:      req.Context,
				Tone:         req.Tone,
				Alternatives: 0,
			}
			resp, err := s.Translate(ctx, r)
			if err != nil {
				errCh <- err
				return
			}
			results[idx] = resp
		}(i, item)
	}
	wg.Wait()
	close(errCh)
	if len(errCh) > 0 {
		return nil, <-errCh
	}
	return results, nil
}

func (s *Service) TranslateDocument(ctx context.Context, content, format, targetLang, sourceLang, provider, model, contextHint, tone string) (string, error) {
	s.metrics.RequestsTotal.WithLabelValues("translate_document").Inc()
	switch format {
	case "json":
		return s.translateJSON(ctx, content, targetLang, sourceLang, provider, model, contextHint, tone)
	case "html", "xml":
		return s.translateTagged(ctx, content, targetLang, sourceLang, provider, model, contextHint, tone)
	case "markdown", "plain":
		return s.translateMarkdown(ctx, content, targetLang, sourceLang, provider, model, contextHint, tone)
	default:
		return "", fmt.Errorf("%w: unsupported format %s", common.ErrInvalidInput, format)
	}
}

var codeFenceRE = regexp.MustCompile("(?s)(```(?:[^\n]*\n)?(.*?)```)")
var tagRE = regexp.MustCompile(`(<[^>]+>)`)

func (s *Service) translateMarkdown(ctx context.Context, content, targetLang, sourceLang, provider, model, contextHint, tone string) (string, error) {
	segments := codeFenceRE.Split(content, -1)
	matches := codeFenceRE.FindAllString(content, -1)
	out := make([]string, 0, len(segments))
	for i, seg := range segments {
		if seg != "" {
			translated, err := s.Translate(ctx, providers.TranslationRequest{
				Text:       seg,
				SourceLang: sourceLang,
				TargetLang: targetLang,
				Provider:   provider,
				Model:      model,
				Context:    contextHint,
				Tone:       tone,
			})
			if err != nil {
				return "", err
			}
			out = append(out, translated.Translation)
		}
		if i < len(matches) {
			out = append(out, matches[i])
		}
	}
	return strings.Join(out, "\n\n"), nil
}

func (s *Service) translateTagged(ctx context.Context, content, targetLang, sourceLang, provider, model, contextHint, tone string) (string, error) {
	parts := tagRE.Split(content, -1)
	matches := tagRE.FindAllString(content, -1)
	out := make([]string, 0, len(parts))
	for i, part := range parts {
		if part != "" && !strings.HasPrefix(part, "<") {
			translated, err := s.Translate(ctx, providers.TranslationRequest{
				Text:       part,
				SourceLang: sourceLang,
				TargetLang: targetLang,
				Provider:   provider,
				Model:      model,
				Context:    contextHint,
				Tone:       tone,
			})
			if err != nil {
				return "", err
			}
			out = append(out, translated.Translation)
		} else {
			out = append(out, part)
		}
		if i < len(matches) {
			out = append(out, matches[i])
		}
	}
	return strings.Join(out, ""), nil
}

func (s *Service) translateJSON(ctx context.Context, content, targetLang, sourceLang, provider, model, contextHint, tone string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("%w: invalid json: %v", common.ErrInvalidInput, err)
	}
	if err := s.walkJSON(ctx, &data, targetLang, sourceLang, provider, model, contextHint, tone); err != nil {
		return "", err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Service) walkJSON(ctx context.Context, v *any, targetLang, sourceLang, provider, model, contextHint, tone string) error {
	switch val := (*v).(type) {
	case string:
		if val == "" {
			return nil
		}
		resp, err := s.Translate(ctx, providers.TranslationRequest{
			Text:       val,
			SourceLang: sourceLang,
			TargetLang: targetLang,
			Provider:   provider,
			Model:      model,
			Context:    contextHint,
			Tone:       tone,
		})
		if err != nil {
			return err
		}
		*v = resp.Translation
	case map[string]any:
		for k, vv := range val {
			if err := s.walkJSON(ctx, &vv, targetLang, sourceLang, provider, model, contextHint, tone); err != nil {
				return err
			}
			val[k] = vv
		}
	case []any:
		for i := range val {
			if err := s.walkJSON(ctx, &val[i], targetLang, sourceLang, provider, model, contextHint, tone); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) Health(ctx context.Context) map[string]string {
	status := make(map[string]string)
	for name, p := range s.providers {
		if err := p.Health(ctx); err != nil {
			status[name] = "unhealthy: " + err.Error()
		} else {
			status[name] = "healthy"
		}
	}
	return status
}

func commonFirst(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
