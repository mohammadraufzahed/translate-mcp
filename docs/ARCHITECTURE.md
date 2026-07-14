# Architecture

## Overview

`translate-mcp` is a stateless/semi-stateless tool server that implements the Model Context Protocol (MCP). It is designed to be plugged into any MCP host (Claude Desktop, Cursor, Windsurf, custom clients) and provides reliable, domain-aware, cached translation.

```
┌─────────────────────┐      stdio / HTTP       ┌─────────────────────┐
│   MCP Host / AI     │◄───────────────────────►│  translate-mcp      │
│   (Claude, Cursor,  │   JSON-RPC / Streamable │  Go binary/server    │
│    Windsurf, etc.)  │                         │                     │
└─────────────────────┘                         └──────────┬──────────┘
                                                            │
        ┌──────────┬──────────┬──────────┬──────────┬───────┴───────┐
        ▼          ▼          ▼          ▼          ▼               ▼
      Tools      Cache     Glossary   Memory     Metrics        Providers
```

## Components

### MCP Server

- Built with `github.com/mark3labs/mcp-go`.
- Supports `stdio` transport for local clients and **Streamable HTTP** (`/mcp`) for remote access.
- Registers MCP tools and handles tool invocations.

### Provider layer

Every translation engine implements the `Translator` interface:

```go
type Translator interface {
    Name() string
    Translate(ctx context.Context, req TranslationRequest) (*TranslationResponse, error)
    DetectLanguage(ctx context.Context, text string) (string, float64, error)
    Languages(ctx context.Context) ([]Language, error)
    Health(ctx context.Context) error
}
```

Supported providers:

- OpenAI
- Anthropic
- DeepL
- Google (Gemini)
- Ollama
- LibreTranslate
- Custom OpenAI-compatible endpoints

Adding a new provider only requires implementing the interface and registering it in the provider factory.

### Cache

The cache is a three-tier chain:

| Tier | Backend | Use case |
|------|---------|----------|
| L1   | In-memory LRU with TTL | Fast exact-hit cache |
| L2   | Redis | Shared cache across multiple instances |
| L3   | SQLite / Postgres | Durable translation memory and long-term cache |

The cache key is a SHA-256 hash of text, source/target languages, provider, model, context, tone, and glossary version.

### Glossary pipeline

1. Normalize whitespace.
2. Detect glossary terms and replace them with placeholders (`__TERM_N__`).
3. Send masked text to the provider with a context hint and placeholder mapping.
4. Restore placeholders to their target translations.

This prevents drift for names, brands, and domain terms.

### Translation memory

Stores verified or frequently used translations with fuzzy matching. Before calling a provider, the server can search the memory for a close match and reuse it, reducing API cost for repetitive content.

### Resilience

- **Concurrency limits**: per-provider semaphores.
- **Rate limiting**: token-bucket limiter per provider.
- **Circuit breaker**: temporarily disables a provider after repeated failures.
- **Fallback chain**: automatically tries the next configured provider on failure.
- **Timeouts**: configurable request timeout per translation call.

### Observability

- Structured JSON logs via `slog`.
- `/health` endpoint with provider connectivity.
- `/metrics` endpoint with Prometheus counters and histograms.

## Security

- Bearer token authentication on HTTP.
- CORS allow-list.
- Input length and batch-size limits.
- API keys are read from environment variables and never logged.
