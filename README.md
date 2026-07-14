# Advanced Translate MCP

A production-grade, AI-powered translation server for the [Model Context Protocol](https://modelcontextprotocol.io). Run it as a single static binary, a Docker container, or a Kubernetes service and connect any MCP-compatible host to translate text, documents, and JSON i18n files with caching, glossaries, translation memory, and multi-provider fallback.

[![CI](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/ci.yml)
[![Release](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/release.yml/badge.svg)](https://github.com/mohammadraufzahed/translate-mcp/releases)
[![Docker](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/docker.yml/badge.svg)](https://github.com/mohammadraufzahed/translate-mcp/pkgs/container/translate-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **MCP-native tools**: `translate`, `detect_language`, `batch_translate`, `translate_document`, `list_languages`, `add_glossary_entry`, `get_glossary`, `search_translation_memory`, `add_translation_memory`.
- **Pluggable translation providers**: OpenAI, Anthropic, DeepL, Google (Gemini), Ollama, LibreTranslate, and any OpenAI-compatible endpoint.
- **Multi-tier cache**: in-memory L1, Redis L2, and SQLite/Postgres L3 with a single key space.
- **Glossary pipeline**: protects terms and brand names by replacing them with placeholders before translation and restoring them after.
- **Translation memory**: fuzzy search over previously stored or approved translations.
- **Resilience**: per-provider concurrency limits, rate limiting, circuit breakers, and configurable fallback chains.
- **Observability**: structured JSON logs, `/health`, and Prometheus `/metrics`.
- **Security**: Bearer token auth, CORS allow-list, input length limits, and request timeouts.
- **Multiple transports**: `stdio` for local clients and **Streamable HTTP** (`/mcp`) for remote access.

## Quick start

### Run from source

```bash
go install github.com/mohammadraufzahed/translate-mcp/cmd/translate-mcp@latest

# stdio mode for Claude Desktop
translate-mcp --transport stdio

# HTTP mode
translate-mcp --transport http --port 8787
```

### Run with Docker

```bash
cp config.example.yaml config.yaml
# edit config.yaml and set your provider API keys

docker build -t translate-mcp .
docker run -p 8787:8787 \
  -e TRANSLATE_MCP_TOKEN=sk-... \
  -e OPENAI_API_KEY=sk-... \
  -v ./config.yaml:/app/config.yaml:ro \
  translate-mcp --config /app/config.yaml --transport http --port 8787
```

### Docker Compose with Redis and Postgres

```bash
cp config.example.yaml config.yaml
docker compose up -d
```

## Configuration

All settings are in `config.yaml`. Environment variables can be referenced as `${VAR}` or `${VAR:-default}`.

```yaml
server:
  transport: http
  host: 0.0.0.0
  port: 8787
  auth:
    type: bearer
    token: ${TRANSLATE_MCP_TOKEN}
  cors:
    allowed_origins: ["*"]

logging:
  format: json
  level: info
  mask_text: true

metrics:
  enabled: true
  path: /metrics

cache:
  default_ttl: 24h
  l1:
    type: memory
    max_entries: 10000
  l2:
    type: redis
    address: redis:6379
  l3:
    type: postgres
    dsn: ${DATABASE_DSN}

providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o
  deepl:
    api_key: ${DEEPL_API_KEY}

translation:
  default_provider: openai
  fallback_chain: [openai, deepl, ollama]
  glossary_preprocessing: true
```

See [`config.example.yaml`](config.example.yaml) for a full example.

## Provider setup

| Provider | Required config | Notes |
|----------|-----------------|-------|
| `openai` | `api_key` | GPT-4o, GPT-4o-mini, etc. |
| `anthropic` | `api_key` | Claude 3.5 Sonnet, etc. |
| `deepl` | `api_key` | Free/pro endpoints supported via `base_url` |
| `google` | `api_key` | Gemini via Generative Language API |
| `ollama` | `base_url` | Local models such as `llama3.1` |
| `libretranslate` | optional `api_key` | Local or hosted LibreTranslate |
| `custom` | `base_url` | OpenAI-compatible chat completions |

## MCP client integration

### Claude Desktop (stdio)

```json
{
  "mcpServers": {
    "translate": {
      "command": "/usr/local/bin/translate-mcp",
      "args": ["--transport", "stdio", "--config", "/etc/translate-mcp/config.yaml"]
    }
  }
}
```

### Cursor / Windsurf / custom clients (HTTP)

```json
{
  "mcpServers": {
    "translate": {
      "url": "http://localhost:8787/mcp",
      "headers": {
        "Authorization": "Bearer sk-your-token"
      }
    }
  }
}
```

## Tools

### `translate`

```json
{
  "text": "Hello world",
  "target_language": "es",
  "source_language": "auto",
  "provider": "openai",
  "tone": "neutral"
}
```

### `batch_translate`

One source to many targets:

```json
{
  "text": "Hello",
  "source_language": "en",
  "targets": ["es", "fr", "de"]
}
```

Many texts to one target:

```json
{
  "items": [{"text": "Hello"}, {"text": "World"}],
  "target_language": "es"
}
```

### `translate_document`

Translates Markdown, JSON i18n, XML, HTML, or plain text while preserving structure.

```json
{
  "content": "{\"greeting\": \"Hello\"}",
  "format": "json",
  "target_language": "es"
}
```

## Deployment

- **Single binary**: download from releases or `go install`.
- **Docker / Docker Compose**: see [`Dockerfile`](Dockerfile) and [`docker-compose.yml`](docker-compose.yml).
- **systemd**: see [`systemd/translate-mcp.service`](systemd/translate-mcp.service).
- **Kubernetes**: see [`kubernetes/`](kubernetes).

## Health and metrics

- `GET /health` returns server status and provider health.
- `GET /metrics` exposes Prometheus metrics:
  - `mcp_requests_total`
  - `mcp_request_duration_seconds`
  - `translate_cache_hits_total`
  - `translate_provider_calls_total`
  - `translate_errors_total`
  - `translate_input_tokens_total`
  - `translate_output_tokens_total`

## Development

```bash
go mod download
go build ./...
go test ./...
go vet ./...
```

## Architecture

```
┌─────────────┐     stdio / HTTP      ┌─────────────────┐
│  MCP Host   │◄─────────────────────►│  translate-mcp  │
└─────────────┘                       └────────┬────────┘
                                             │
    ┌────────┬────────┬────────┬────────┬────┴────┬──────────────┐
    ▼        ▼        ▼        ▼        ▼           ▼
  Tools    Cache    Glossary  Memory   Metrics    Providers
```

The server exposes tools, routes each translation request through the cache, optionally applies glossary masking, queries translation memory, then calls the configured provider chain. Results are cached and, on success, optionally stored in translation memory.

## CI/CD

- **Pull requests** trigger `go vet`, `gofmt`, `go test -race`, cross-compilation, and a Docker build for `linux/amd64` and `linux/arm64`.
- **Merges to `main`** build and push a Docker image to `ghcr.io/mohammadraufzahed/translate-mcp`.
- **Semver tags `v*.*.*`** trigger GoReleaser, which publishes binaries for Linux, macOS, and Windows on amd64 and arm64, plus a GitHub release.

## Releasing

```bash
git checkout main
git pull
git tag v1.0.0
git push origin v1.0.0
```

GoReleaser and the Docker workflow will handle the rest.

## License

[MIT](LICENSE)
