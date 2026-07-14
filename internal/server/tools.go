package server

import (
	"context"
	"fmt"

	mcp "github.com/mark3labs/mcp-go/mcp"

	"github.com/mohammadraufzahed/translate-mcp/internal/common"
	"github.com/mohammadraufzahed/translate-mcp/internal/glossary"
	"github.com/mohammadraufzahed/translate-mcp/internal/memory"
	"github.com/mohammadraufzahed/translate-mcp/internal/providers"
	"github.com/mohammadraufzahed/translate-mcp/internal/translator"
)

func (s *Server) registerTools() {
	s.mcp.AddTool(translateTool(), s.handleTranslate)
	s.mcp.AddTool(detectTool(), s.handleDetect)
	s.mcp.AddTool(batchTool(), s.handleBatch)
	s.mcp.AddTool(documentTool(), s.handleDocument)
	s.mcp.AddTool(listLanguagesTool(), s.handleListLanguages)
	s.mcp.AddTool(addGlossaryTool(), s.handleAddGlossary)
	s.mcp.AddTool(getGlossaryTool(), s.handleGetGlossary)
	s.mcp.AddTool(addMemoryTool(), s.handleAddMemory)
	s.mcp.AddTool(searchMemoryTool(), s.handleSearchMemory)
}

func translateTool() mcp.Tool {
	return mcp.NewTool("translate",
		mcp.WithDescription("Translate text into a target language. Supports glossaries, context hints, and caching."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to translate")),
		mcp.WithString("target_language", mcp.Required(), mcp.Description("BCP-47 code, e.g. 'es', 'zh-CN', 'fr-FR'")),
		mcp.WithString("source_language", mcp.Description("Source BCP-47 code or 'auto'"), mcp.DefaultString("auto")),
		mcp.WithString("provider", mcp.Description("Preferred engine: openai, anthropic, deepl, google, ollama, libretranslate, custom")),
		mcp.WithString("model", mcp.Description("Specific model ID")),
		mcp.WithString("context", mcp.Description("Domain/tone hint, e.g. 'medical', 'software UI'.")),
		mcp.WithString("tone", mcp.Description("Tone: formal, informal, neutral"), mcp.DefaultString("neutral")),
		mcp.WithBoolean("use_cache", mcp.Description("Use cache"), mcp.DefaultBool(true)),
		mcp.WithNumber("alternatives", mcp.Description("Number of alternative translations to return"), mcp.DefaultNumber[int](0)),
	)
}

type translateArgs struct {
	Text           string `json:"text"`
	TargetLanguage string `json:"target_language"`
	SourceLanguage string `json:"source_language"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Context        string `json:"context"`
	Tone           string `json:"tone"`
	UseCache       bool   `json:"use_cache"`
	Alternatives   int    `json:"alternatives"`
}

func (s *Server) handleTranslate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("translate").Inc()
	var args translateArgs
	if err := req.BindArguments(&args); err != nil {
		return errorResult(fmt.Errorf("%w: %v", common.ErrInvalidInput, err)), nil
	}
	if args.UseCache {
		_ = req.GetBool("use_cache", true)
	}
	if args.SourceLanguage == "" {
		args.SourceLanguage = "auto"
	}
	if args.Tone == "" {
		args.Tone = "neutral"
	}
	resp, err := s.svc.Translate(ctx, providers.TranslationRequest{
		Text:         args.Text,
		SourceLang:   args.SourceLanguage,
		TargetLang:   args.TargetLanguage,
		Provider:     args.Provider,
		Model:        args.Model,
		Context:      args.Context,
		Tone:         args.Tone,
		Alternatives: args.Alternatives,
	})
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(resp)
}

func detectTool() mcp.Tool {
	return mcp.NewTool("detect_language",
		mcp.WithDescription("Detect the language of a text."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to analyze")),
		mcp.WithString("provider", mcp.Description("Provider to use")),
	)
}

func (s *Server) handleDetect(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("detect_language").Inc()
	text := req.GetString("text", "")
	if text == "" {
		return errorResult(fmt.Errorf("%w: text required", common.ErrInvalidInput)), nil
	}
	lang, conf, err := s.svc.DetectLanguage(ctx, text)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(map[string]any{
		"language":   lang,
		"confidence": conf,
	})
}

func batchTool() mcp.Tool {
	return mcp.NewTool("batch_translate",
		mcp.WithDescription("Translate a text into multiple target languages or many source strings into one target."),
		mcp.WithString("text", mcp.Description("Source text for one-to-many mode")),
		mcp.WithString("source_language", mcp.Description("Source BCP-47 code or 'auto'"), mcp.DefaultString("auto")),
		mcp.WithString("target_language", mcp.Description("Target BCP-47 code for many-to-one mode")),
		mcp.WithArray("targets", mcp.Description("List of target language codes"), mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("items", mcp.Description("List of objects with 'text' and optional 'target_language'"), mcp.Items(map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}, "target_language": map[string]any{"type": "string"}}})),
		mcp.WithString("provider", mcp.Description("Provider to use")),
		mcp.WithString("model", mcp.Description("Model ID")),
		mcp.WithString("context", mcp.Description("Context hint")),
		mcp.WithString("tone", mcp.Description("Tone")),
	)
}

func (s *Server) handleBatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("batch_translate").Inc()
	text := req.GetString("text", "")
	source := req.GetString("source_language", "auto")
	target := req.GetString("target_language", "")
	targets := req.GetStringSlice("targets", nil)
	provider := req.GetString("provider", "")
	model := req.GetString("model", "")
	contextHint := req.GetString("context", "")
	tone := req.GetString("tone", "")

	args := translator.BatchRequest{
		SourceLang: source,
		TargetLang: target,
		Targets:    targets,
		Provider:   provider,
		Model:      model,
		Context:    contextHint,
		Tone:       tone,
	}
	if text != "" && len(targets) > 0 {
		args.Mode = translator.BatchOneToMany
		args.Text = text
	} else if rawItems, ok := req.GetArguments()["items"].([]any); ok && len(rawItems) > 0 {
		args.Mode = translator.BatchManyToOne
		args.Items = make([]translator.BatchItem, 0, len(rawItems))
		for _, ri := range rawItems {
			m, ok := ri.(map[string]any)
			if !ok {
				continue
			}
			it := translator.BatchItem{}
			if v, ok := m["text"].(string); ok {
				it.Text = v
			}
			if v, ok := m["target_language"].(string); ok {
				it.TargetLang = v
			}
			args.Items = append(args.Items, it)
		}
	} else {
		return errorResult(fmt.Errorf("%w: provide text+targets or items", common.ErrInvalidInput)), nil
	}

	results, err := s.svc.BatchTranslate(ctx, args)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(results)
}

func documentTool() mcp.Tool {
	return mcp.NewTool("translate_document",
		mcp.WithDescription("Translate a document while preserving format (markdown, json, xml, html, plain)."),
		mcp.WithString("content", mcp.Required(), mcp.Description("Document content")),
		mcp.WithString("format", mcp.Required(), mcp.Description("Format: markdown, json, xml, html, plain"), mcp.Enum("markdown", "json", "xml", "html", "plain")),
		mcp.WithString("target_language", mcp.Required(), mcp.Description("Target BCP-47 code")),
		mcp.WithString("source_language", mcp.Description("Source BCP-47 code or 'auto'"), mcp.DefaultString("auto")),
		mcp.WithString("provider", mcp.Description("Provider to use")),
		mcp.WithString("model", mcp.Description("Model ID")),
		mcp.WithString("context", mcp.Description("Context hint")),
		mcp.WithString("tone", mcp.Description("Tone")),
	)
}

func (s *Server) handleDocument(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("translate_document").Inc()
	content := req.GetString("content", "")
	format := req.GetString("format", "")
	target := req.GetString("target_language", "")
	source := req.GetString("source_language", "auto")
	if content == "" || format == "" || target == "" {
		return errorResult(fmt.Errorf("%w: content, format and target_language required", common.ErrInvalidInput)), nil
	}
	out, err := s.svc.TranslateDocument(ctx, content, format, target, source,
		req.GetString("provider", ""), req.GetString("model", ""), req.GetString("context", ""), req.GetString("tone", ""))
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(map[string]any{
		"translation":     out,
		"format":          format,
		"target_language": target,
	})
}

func listLanguagesTool() mcp.Tool {
	return mcp.NewTool("list_languages",
		mcp.WithDescription("List supported languages for a provider."),
		mcp.WithString("provider", mcp.Description("Provider name")),
	)
}

func (s *Server) handleListLanguages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("list_languages").Inc()
	provider := req.GetString("provider", "")
	langs, err := s.svc.ListLanguages(ctx, provider)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(langs)
}

func addGlossaryTool() mcp.Tool {
	return mcp.NewTool("add_glossary_entry",
		mcp.WithDescription("Add a terminology entry to the glossary."),
		mcp.WithString("source_term", mcp.Required(), mcp.Description("Source term")),
		mcp.WithString("target_language", mcp.Required(), mcp.Description("Target language code")),
		mcp.WithString("translation", mcp.Required(), mcp.Description("Translation of the term")),
		mcp.WithString("context", mcp.Description("Domain context")),
		mcp.WithBoolean("case_sensitive", mcp.Description("Case sensitive matching"), mcp.DefaultBool(false)),
	)
}

func (s *Server) handleAddGlossary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("add_glossary_entry").Inc()
	entry := glossary.Entry{
		SourceTerm:    req.GetString("source_term", ""),
		TargetLang:    req.GetString("target_language", ""),
		Translation:   req.GetString("translation", ""),
		Context:       req.GetString("context", ""),
		CaseSensitive: req.GetBool("case_sensitive", false),
	}
	if entry.SourceTerm == "" || entry.TargetLang == "" || entry.Translation == "" {
		return errorResult(fmt.Errorf("%w: source_term, target_language and translation required", common.ErrInvalidInput)), nil
	}
	s.svc.Glossary().Add(entry)
	return jsonResult(map[string]any{"status": "added"})
}

func getGlossaryTool() mcp.Tool {
	return mcp.NewTool("get_glossary",
		mcp.WithDescription("Get all glossary entries, optionally filtered by target language."),
		mcp.WithString("target_language", mcp.Description("Target language filter")),
	)
}

func (s *Server) handleGetGlossary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("get_glossary").Inc()
	lang := req.GetString("target_language", "")
	entries := s.svc.Glossary().Get(lang)
	return jsonResult(map[string]any{"entries": entries})
}

func addMemoryTool() mcp.Tool {
	return mcp.NewTool("add_translation_memory",
		mcp.WithDescription("Store a verified translation in the translation memory."),
		mcp.WithString("source_text", mcp.Required(), mcp.Description("Source text")),
		mcp.WithString("target_text", mcp.Required(), mcp.Description("Target text")),
		mcp.WithString("source_language", mcp.Required(), mcp.Description("Source language code")),
		mcp.WithString("target_language", mcp.Required(), mcp.Description("Target language code")),
		mcp.WithString("domain", mcp.Description("Domain tag")),
		mcp.WithString("project", mcp.Description("Project tag")),
	)
}

func (s *Server) handleAddMemory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("add_translation_memory").Inc()
	entry := memory.Entry{
		SourceText: req.GetString("source_text", ""),
		TargetText: req.GetString("target_text", ""),
		SourceLang: req.GetString("source_language", ""),
		TargetLang: req.GetString("target_language", ""),
		Domain:     req.GetString("domain", ""),
		Project:    req.GetString("project", ""),
	}
	if entry.SourceText == "" || entry.TargetText == "" || entry.SourceLang == "" || entry.TargetLang == "" {
		return errorResult(fmt.Errorf("%w: source_text, target_text, source_language, target_language required", common.ErrInvalidInput)), nil
	}
	if err := s.svc.Memory().Store(ctx, entry); err != nil {
		return errorResult(err), nil
	}
	return jsonResult(map[string]any{"status": "added"})
}

func searchMemoryTool() mcp.Tool {
	return mcp.NewTool("search_translation_memory",
		mcp.WithDescription("Search the translation memory for similar previous translations."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Source text to search")),
		mcp.WithString("source_language", mcp.Description("Source language filter")),
		mcp.WithString("target_language", mcp.Description("Target language filter")),
		mcp.WithNumber("threshold", mcp.Description("Similarity threshold 0-1"), mcp.DefaultNumber[float64](0.8)),
	)
}

func (s *Server) handleSearchMemory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.svc.Metrics().RequestsTotal.WithLabelValues("search_translation_memory").Inc()
	text := req.GetString("text", "")
	source := req.GetString("source_language", "")
	target := req.GetString("target_language", "")
	threshold := req.GetFloat("threshold", 0.8)
	if threshold == 0 {
		threshold = 0.8
	}
	results, err := s.svc.Memory().Search(ctx, text, source, target, threshold)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(map[string]any{"results": results})
}
