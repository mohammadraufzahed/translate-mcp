# Configuration Reference

`translate-mcp` is configured through a YAML file. Environment variables are expanded at load time with `${VAR}` or `${VAR:-default}` syntax.

## File location

The file path is passed with `--config`:

```bash
translate-mcp --config /etc/translate-mcp/config.yaml
```

If `--config` is omitted, the server starts with sensible defaults but has no providers configured.

## Top-level sections

| Section | Description |
|---------|-------------|
| `server` | Transport, host, port, auth, and CORS |
| `logging` | Log format and level |
| `metrics` | Prometheus `/metrics` endpoint |
| `cache` | L1/L2/L3 cache configuration |
| `providers` | Provider API keys and defaults |
| `translation` | Default provider, fallback chain, limits |
| `vector` | Semantic translation memory (reserved for future use) |

## `server`

```yaml
server:
  transport: http   # stdio | http
  host: 0.0.0.0
  port: 8787
  auth:
    type: bearer
    token: ${TRANSLATE_MCP_TOKEN}
  cors:
    allowed_origins: ["*"]
```

| Field | Default | Description |
|-------|---------|-------------|
| `transport` | `stdio` | `stdio` for local MCP hosts, `http` for remote |
| `host` | `0.0.0.0` | HTTP bind host |
| `port` | `8787` | HTTP bind port |
| `auth.type` | `bearer` | Currently only `bearer` is supported |
| `auth.token` | `""` | Bearer token. Empty means no auth. |
| `cors.allowed_origins` | `[]` | Origins allowed by CORS. `["*"]` allows all. |

Command-line flags override these values:

```bash
translate-mcp --transport http --host 127.0.0.1 --port 8080
```

## `logging`

```yaml
logging:
  format: json   # json | text
  level: info    # debug | info | warn | error
  mask_text: true
```

When `mask_text` is `true`, source text is omitted from logs to reduce accidental PII leakage.

## `metrics`

```yaml
metrics:
  enabled: true
  path: /metrics
```

The `/metrics` endpoint is exposed on the same HTTP port. It is not protected by bearer auth so a Prometheus scraper can reach it. Run a reverse proxy or firewall it in production.

## `cache`

```yaml
cache:
  default_ttl: 24h
  l1:
    type: memory
    max_entries: 10000
  l2:
    type: redis
    address: redis:6379
    password: ${REDIS_PASSWORD:-}
    ttl: 24h
  l3:
    type: postgres
    dsn: ${DATABASE_DSN}
    ttl: 720h
```

### Tiers

| Tier | Type | Purpose |
|------|------|---------|
| L1 | `memory` | Fast per-process LRU cache |
| L2 | `redis` | Shared cache across containers/hosts |
| L3 | `postgres` or `sqlite` | Durable long-term cache and translation memory |

Only the tiers you define are used. A typical production setup uses L1 + L2 + L3.

### Cache key

The cache key is a SHA-256 hash of:

- normalized source text
- source and target language codes
- provider and model
- context and tone
- glossary version

This makes exact matches fast and avoids leaking text in cache storage.

## `providers`

Each key is the provider name used in MCP requests. See [`PROVIDERS.md`](PROVIDERS.md) for provider-specific details.

```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o
  deepl:
    api_key: ${DEEPL_API_KEY}
    base_url: https://api.deepl.com
```

Common provider fields:

| Field | Description |
|-------|-------------|
| `api_key` | Provider API key |
| `base_url` | Custom endpoint base URL |
| `default_model` | Model used when the request does not specify one |

## `translation`

```yaml
translation:
  default_provider: openai
  max_text_length: 10000
  max_batch_items: 50
  request_timeout: 60s
  fallback_chain: [openai, deepl, ollama]
  glossary_preprocessing: true
```

| Field | Default | Description |
|-------|---------|-------------|
| `default_provider` | `openai` | Provider used when the request does not specify one |
| `max_text_length` | `10000` | Maximum rune length of a single translation request |
| `max_batch_items` | `50` | Maximum targets or items in a batch request |
| `request_timeout` | `60s` | Per-translation request timeout |
| `fallback_chain` | `[]` | Ordered list of backup providers |
| `glossary_preprocessing` | `true` | Whether to apply glossary masking before translation |

## `vector`

Reserved for future semantic translation memory and embedding-based matching.

```yaml
vector:
  enabled: false
  provider: openai
  embedding_model: text-embedding-3-small
  similarity_threshold: 0.85
```

## Environment variables

Any YAML string can reference environment variables:

```yaml
token: ${TRANSLATE_MCP_TOKEN}
model: ${OPENAI_MODEL:-gpt-4o-mini}
```

- `${VAR}` is replaced with the value of `VAR` or an empty string.
- `${VAR:-default}` is replaced with `VAR` if set, otherwise `default`.

Secrets should always be loaded from environment variables or a secret manager, never committed to the config file.

## Full example

See [`config.example.yaml`](../config.example.yaml) in the repository root.
