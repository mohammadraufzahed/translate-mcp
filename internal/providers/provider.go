package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/mohammadraufzahed/translate-mcp/internal/common"
	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/glossary"
)

type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type TranslationRequest struct {
	Text         string
	SourceLang   string
	TargetLang   string
	Provider     string
	Model        string
	Context      string
	Tone         string
	Alternatives int
	Glossary     *glossary.Glossary
}

type TranslationResponse struct {
	Translation  string   `json:"translation"`
	SourceLang   string   `json:"source_language"`
	TargetLang   string   `json:"target_language"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	Cached       bool     `json:"cached"`
	Confidence   float64  `json:"confidence"`
	Alternatives []string `json:"alternatives"`
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
}

type Translator interface {
	Name() string
	Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error)
	DetectLanguage(ctx context.Context, text string) (string, float64, error)
	Languages(ctx context.Context) ([]Language, error)
	Health(ctx context.Context) error
}

type Factory func(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error)

var registry = map[string]Factory{
	"openai":         newOpenAI,
	"anthropic":      newAnthropic,
	"deepl":          newDeepL,
	"google":         newGoogle,
	"ollama":         newOllama,
	"libretranslate": newLibreTranslate,
	"custom":         newCustom,
}

func Build(name string, cfg config.ProviderConfig, timeout time.Duration) (Translator, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	return f(name, cfg, timeout)
}

func AvailableProviders() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}

func buildPrompt(req TranslationRequest, masked string, mapping map[string]string) string {
	var b stringsBuilder
	b.WriteString("You are a professional translator. Translate the following text accurately into ")
	b.WriteString(common.LanguageName(req.TargetLang))
	b.WriteString(". ")
	if req.Context != "" {
		b.WriteString("Context: ")
		b.WriteString(req.Context)
		b.WriteString(". ")
	}
	if req.Tone != "" && req.Tone != "neutral" {
		b.WriteString("Tone: ")
		b.WriteString(req.Tone)
		b.WriteString(". ")
	}
	if len(mapping) > 0 {
		b.WriteString("Preserve the following placeholders exactly as they appear and do not translate them: ")
		first := true
		for k, v := range mapping {
			if !first {
				b.WriteString("; ")
			}
			b.WriteString(k)
			b.WriteString(" -> ")
			b.WriteString(v)
			first = false
		}
		b.WriteString(". ")
	}
	b.WriteString("Return ONLY a JSON object with keys: translation (string), source_language (string), confidence (number 0-1).")
	if req.Alternatives > 0 {
		b.WriteString(" Also include alternatives (array of strings).")
	}
	b.WriteString("\n\nText: ")
	b.WriteString(masked)
	return b.String()
}

type stringsBuilder struct {
	data []byte
}

func (s *stringsBuilder) WriteString(v string) {
	s.data = append(s.data, v...)
}

func (s *stringsBuilder) String() string {
	return string(s.data)
}
