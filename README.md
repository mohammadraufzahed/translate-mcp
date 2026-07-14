# translate-mcp

A production-grade, model-agnostic translation server for the [Model Context Protocol (MCP)](https://modelcontextprotocol.io). Run it as a single static binary, a Docker container, or a Kubernetes service and connect any MCP-compatible host — Claude Desktop, Cursor, Windsurf, custom clients — to translate text, documents, and JSON i18n files with caching, glossaries, translation memory, and multi-provider fallback.

[![CI](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/ci.yml)
[![Release](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/release.yml/badge.svg)](https://github.com/mohammadraufzahed/translate-mcp/releases)
[![Docker](https://github.com/mohammadraufzahed/translate-mcp/actions/workflows/docker.yml/badge.svg)](https://github.com/mohammadraufzahed/translate-mcp/pkgs/container/translate-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

- [Features](#features)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Providers](#providers)
- [MCP tools](#mcp-tools)
- [Document translation](#document-translation)
- [Deployment](#deployment)
- [Observability](#observability)
- [Development](#development)
- [Architecture](#architecture)
- [License](#license)

## Features

- **MCP-native tools**: `translate`, `detect_language`, `batch_translate`, `translate_document`, `list_languages`, `add_glossary_entry`, `get_glossary`, `add_translation_memory`, `search_translation_memory`.
- **Pluggable translation providers**: OpenAI, OpenRouter, Anthropic, DeepL, Google (Gemini), Ollama, LibreTranslate, and any OpenAI-compatible endpoint.
- **Multi-tier cache**: L1 in-memory LRU, L2 Redis, L3 SQL (SQLite/Postgres) with a single key space.
- **Glossary pipeline**: protects terms and brand names by replacing them with placeholders before translation and restoring them after.
- **Translation memory**: fuzzy search over previously stored or approved translations.
- **Resilience**: per-provider concurrency limits, rate limiting, circuit breakers, and configurable fallback chains.
- **Observability**: structured JSON logs, `/health`, and Prometheus `/metrics`.
- **Security**: Bearer token auth, CORS allow-list, input length limits, and request timeouts.
- **Multiple transports**: `stdio` for local clients and **Streamable HTTP** (`/mcp`) for remote access.
- **Installable releases**: pre-built binaries plus `.deb`, `.rpm`, `.apk`, Arch `.pkg.tar.zst`, `.dmg`, and NSIS `.exe` installers on every release.

## Installation

### Pre-built binaries

Download the latest release for your platform from the [Releases page](https://github.com/mohammadraufzahed/translate-mcp/releases).

### Package managers

After each release, installable artifacts are published:

| Format | Install command |
|--------|-----------------|
| Debian/Ubuntu | `sudo dpkg -i translate-mcp_*.deb` |
| RHEL/CentOS/Fedora | `sudo rpm -i translate-mcp-*.rpm` |
| Alpine | `sudo apk add --allow-untrusted translate-mcp_*.apk` |
| Arch Linux | `sudo pacman -U translate-mcp-*.pkg.tar.zst` |
| macOS | Open the `.dmg` and copy `translate-mcp` to `/usr/local/bin` |
| Windows | Run the `*_setup.exe` installer |

### Go install

```bash
go install github.com/mohammadraufzahed/translate-mcp/cmd/translate-mcp@latest
```

### Docker

```bash
docker pull ghcr.io/mohammadraufzahed/translate-mcp:latest
```

## Quick start

1. Copy the example configuration:

   ```bash
   cp config.example.yaml config.yaml
   ```

2. Edit `config.yaml` and set at least one provider API key, for example `OPENAI_API_KEY`.

3. Run the server:

   ```bash
   # stdio mode for Claude Desktop
   translate-mcp --transport stdio

   # HTTP mode
   translate-mcp --transport http --port 8787
   ```

4. With HTTP mode, test the health endpoint:

   ```bash
   curl http://localhost:8787/health
   ```

### Run with Docker

```bash
cp config.example.yaml config.yaml
# edit config.yaml and set your provider API keys

docker run -p 8787:8787 \
  -e TRANSLATE_MCP_TOKEN=sk-... \
  -e OPENAI_API_KEY=sk-... \
  -v ./config.yaml:/app/config.yaml:ro \
  ghcr.io/mohammadraufzahed/translate-mcp:latest \
  --config /app/config.yaml --transport http --port 8787
```

### Docker Compose with Redis and Postgres

```bash
cp config.example.yaml config.yaml
# edit config.yaml and set your provider API keys
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
    password: ${REDIS_PASSWORD}
  l3:
    type: postgres
    dsn: ${DATABASE_DSN}

providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    default_model: openai/gpt-4o-mini
    site_url: https://github.com/mohammadraufzahed/translate-mcp
    site_name: translate-mcp
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    default_model: claude-3-5-sonnet-20241022
  deepl:
    api_key: ${DEEPL_API_KEY}
    base_url: https://api.deepl.com
  google:
    api_key: ${GOOGLE_API_KEY}
    default_model: gemini-1.5-flash
  ollama:
    base_url: http://localhost:11434
    default_model: llama3.1
  libretranslate:
    base_url: https://libretranslate.com
    api_key: ${LIBRETRANSLATE_API_KEY}
  custom:
    base_url: http://localhost:8000/v1
    default_model: default

translation:
  default_provider: openai
  max_text_length: 10000
  max_batch_items: 50
  request_timeout: 60s
  fallback_chain: [openai, deepl, ollama]
  glossary_preprocessing: true

vector:
  enabled: false
  provider: openai
  embedding_model: text-embedding-3-small
  similarity_threshold: 0.85
```

See [`config.example.yaml`](config.example.yaml) and [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) for the complete reference.

## Providers

| Provider | Required config | Notes |
|----------|-----------------|-------|
| `openai` | `api_key` | GPT-4o, GPT-4o-mini, etc. |
| `openrouter` | `api_key` | Any OpenRouter model (e.g. `openai/gpt-4o-mini`, `anthropic/claude-3.5-sonnet`) |
| `anthropic` | `api_key` | Claude 3.5 Sonnet, etc. |
| `deepl` | `api_key` | Free/pro endpoints supported via `base_url` |
| `google` | `api_key` | Gemini via Generative Language API |
| `ollama` | `base_url` | Local models such as `llama3.1` |
| `libretranslate` | optional `api_key` | Local or hosted LibreTranslate |
| `custom` | `base_url` | OpenAI-compatible chat completions |

See [`docs/PROVIDERS.md`](docs/PROVIDERS.md) for setup examples, model IDs, and region-specific endpoints.

## MCP tools

### `translate`

Translate text into a target language.

```json
{
  "text": "Hello world",
  "target_language": "es",
  "source_language": "auto",
  "provider": "openai",
  "tone": "neutral"
}
```

Parameters:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | yes | Text to translate |
| `target_language` | string | yes | BCP-47 code, e.g. `es`, `zh-CN`, `fr-FR` |
| `source_language` | string | no | Source BCP-47 code or `auto` (default `auto`) |
| `provider` | string | no | Provider name (defaults to `translation.default_provider`) |
| `model` | string | no | Specific model ID |
| `context` | string | no | Domain/tone hint, e.g. `medical`, `software UI` |
| `tone` | string | no | `formal`, `informal`, or `neutral` (default `neutral`) |
| `use_cache` | boolean | no | Use cache (default `true`) |
| `alternatives` | number | no | Number of alternative translations to return (default `0`) |

### `detect_language`

Detect the language of a text.

```json
{
  "text": "Hola mundo"
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
  "items": [
    {"text": "Hello"},
    {"text": "World"}
  ],
  "target_language": "es",
  "source_language": "en"
}
```

### `translate_document`

Translate Markdown, JSON i18n, XML, HTML, or plain text while preserving structure.

```json
{
  "content": "{\"greeting\": \"Hello\"}",
  "format": "json",
  "target_language": "es",
  "source_language": "en"
}
```

### `add_glossary_entry` / `get_glossary`

Add and retrieve terminology. The server masks source terms before translation and restores target translations after.

```json
{
  "source_term": "OpenAI",
  "target_language": "es",
  "translation": "OpenAI",
  "case_sensitive": false
}
```

### `add_translation_memory` / `search_translation_memory`

Store and search previously translated segments.

See [`docs/TOOLS.md`](docs/TOOLS.md) for the full tool schema and response formats.

## Document translation

`translate_document` preserves the original structure:

- **JSON**: recursively translates all string values, preserving keys.
- **HTML/XML**: preserves tags and translates text nodes.
- **Markdown**: preserves code fences and translates prose segments.
- **Plain**: translates the whole text.

Example JSON i18n:

```json
{
  "content": "{\"nav\":{\"home\":\"Home\",\"about\":\"About\"}}",
  "format": "json",
  "target_language": "es"
}
```

Result:

```json
{
  "nav": {
    "home": "Inicio",
    "about": "Acerca de"
  }
}
```

## MCP client integration

### Claude Desktop (stdio)

Edit `claude_desktop_config.json`:

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

## Deployment

- **Single binary**: download from [Releases](https://github.com/mohammadraufzahed/translate-mcp/releases) or `go install`.
- **Docker / Docker Compose**: see [`Dockerfile`](Dockerfile) and [`docker-compose.yml`](docker-compose.yml).
- **systemd**: see [`systemd/translate-mcp.service`](systemd/translate-mcp.service).
- **Kubernetes**: see [`kubernetes/`](kubernetes).

See [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) for production deployment hardening, secrets management, and horizontal scaling.

## Observability

- `GET /health` returns server status and provider health:

  ```json
  {
    "status": "ok",
    "version": "1.0.0",
    "providers": {
      "openai": "healthy",
      "deepl": "unhealthy: api key not configured"
    }
  }
  ```

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
go test -race ./...
go vet ./...
gofmt -w .
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for internals.

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

The server exposes MCP tools, routes each translation request through the cache, optionally applies glossary masking, queries translation memory, then calls the configured provider chain. Results are cached and, on success, optionally stored in translation memory.

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for a deep dive.

## CI/CD

- **Pull requests** trigger `go vet`, `gofmt`, `go test -race`, cross-compilation, and a Docker build.
- **Merges to `main`** run the CI pipeline and any outstanding workflow dispatch tasks.
- **Semver tags `v*.*.*`** trigger GoReleaser, which publishes binaries for Linux, macOS, and Windows on amd64 and arm64 plus `.deb`, `.rpm`, `.apk`, Arch packages, `.dmg`, and `.exe` installers. The same tag triggers the Docker build, which pushes `ghcr.io/mohammadraufzahed/translate-mcp` with tags `latest`, `{{version}}`, and `{{major}}.{{minor}}`.

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
