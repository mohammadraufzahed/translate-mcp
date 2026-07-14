# Architecture

`translate-mcp` is a stateless/semi-stateless MCP tool server written in Go. It translates text through a pluggable provider layer while adding caching, glossary masking, translation memory, and resilience.

## High-level flow

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

1. The MCP host calls a tool via `stdio` or streamable HTTP.
2. The server validates the request.
3. It computes a cache key and checks the cache tier chain.
4. If there is a cache miss, it applies glossary masking and translation-memory search.
5. It calls the primary provider. If that fails, it walks the fallback chain.
6. It restores glossary placeholders, writes the result to cache, and returns JSON.

## Components

### MCP Server

Built with `github.com/mark3labs/mcp-go`.

- `stdio` transport for local clients such as Claude Desktop.
- Streamable HTTP transport at `/mcp` for remote clients such as Cursor and Windsurf.
- Registers tools and handles JSON-RPC tool calls.

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
- OpenRouter
- Anthropic
- DeepL
- Google (Gemini)
- Ollama
- LibreTranslate
- Custom OpenAI-compatible endpoints

Adding a provider means implementing the interface and registering it in the provider factory.

### Cache

The cache is a three-tier chain:

| Tier | Backend | Use case |
|------|---------|----------|
| L1 | In-memory LRU with TTL | Fast exact-hit cache per process |
| L2 | Redis | Shared cache across multiple instances |
| L3 | SQLite / Postgres | Durable translation memory and long-term cache |

The cache key is a SHA-256 hash of text, source/target languages, provider, model, context, tone, and glossary version. This keeps storage keys opaque and avoids leaking source text into cache storage.

### Glossary pipeline

1. Normalize whitespace.
2. Detect glossary terms and replace them with placeholders (`__TERM_N__`).
3. Send masked text to the provider with a context hint and placeholder mapping.
4. Restore placeholders to their target translations.

This prevents drift for names, brands, and domain terms while still letting the LLM translate the surrounding text.

### Translation memory

Stores verified or frequently used translations with fuzzy matching. Before calling a provider, the server searches the memory for a close match and can reuse it, reducing API cost for repetitive content.

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

## Data flow for `translate`

```
MCP request
    │
    ▼
Validate arguments
    │
    ▼
Normalize text and language codes
    │
    ▼
Resolve provider + fallback chain
    │
    ▼
Compute cache key
    │
    ▼
Check L1 → L2 → L3 cache
    │
    ▼
(optional) Search translation memory
    │
    ▼
(optional) Apply glossary masking
    │
    ▼
Call provider (with rate limit, semaphore, circuit breaker)
    │
    ▼
Restore glossary placeholders
    │
    ▼
Store result in cache
    │
    ▼
Return JSON response
```

## Security

- Bearer token authentication on HTTP.
- CORS allow-list.
- Input length and batch-size limits.
- API keys are read from environment variables and never logged.
- Cache keys are hashed so source text is not stored as plain keys.
