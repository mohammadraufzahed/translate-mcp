# Getting Started

This guide walks you through running `translate-mcp` locally in a few minutes.

## Prerequisites

- Go 1.25+ (only for source builds) **or** Docker.
- At least one translation provider API key (OpenAI, OpenRouter, Anthropic, DeepL, Google).

## 1. Get the binary

### From a release

Download the archive, `.deb`, `.rpm`, `.apk`, `.pkg.tar.zst`, `.dmg`, or `.exe` for your platform from the [Releases page](https://github.com/mohammadraufzahed/translate-mcp/releases).

### From source

```bash
go install github.com/mohammadraufzahed/translate-mcp/cmd/translate-mcp@latest
```

### With Docker

```bash
docker pull ghcr.io/mohammadraufzahed/translate-mcp:latest
```

## 2. Configure the server

Copy the example configuration:

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` and set at least one provider API key:

```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o-mini
```

You can hard-code values or reference environment variables:

```yaml
openai:
  api_key: ${OPENAI_API_KEY}
  default_model: ${OPENAI_MODEL:-gpt-4o-mini}
```

## 3. Run the server

### stdio mode

Use stdio for local MCP hosts such as Claude Desktop:

```bash
translate-mcp --transport stdio --config config.yaml
```

### HTTP mode

Use HTTP for remote clients and custom integrations:

```bash
translate-mcp --transport http --port 8787 --config config.yaml
```

Test it:

```bash
curl http://localhost:8787/health
```

## 4. Connect a client

### Claude Desktop

Add the server to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "translate": {
      "command": "/usr/local/bin/translate-mcp",
      "args": ["--transport", "stdio", "--config", "/path/to/config.yaml"]
    }
  }
}
```

Restart Claude Desktop. You should see the `translate` tools in the prompt.

### Cursor / Windsurf

Add an MCP server over HTTP:

```json
{
  "mcpServers": {
    "translate": {
      "url": "http://localhost:8787/mcp",
      "headers": {
        "Authorization": "Bearer your-token"
      }
    }
  }
}
```

## 5. Translate something

Use the MCP tool:

```json
{
  "text": "Hello world",
  "target_language": "es",
  "source_language": "auto"
}
```

Expected response:

```json
{
  "translation": "Hola mundo",
  "source_language": "en",
  "target_language": "es",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "confidence": 0.98,
  "cached": false
}
```

## 6. Enable caching (optional)

A cache reduces API calls for repeated text. For a single instance, in-memory L1 is enough. For multiple instances, add Redis L2:

```yaml
cache:
  default_ttl: 24h
  l1:
    type: memory
    max_entries: 10000
  l2:
    type: redis
    address: redis:6379
    password: ${REDIS_PASSWORD}
```

## Next steps

- Read [`CONFIGURATION.md`](CONFIGURATION.md) for every config option.
- Read [`PROVIDERS.md`](PROVIDERS.md) for provider-specific setup.
- Read [`DEPLOYMENT.md`](DEPLOYMENT.md) for production hosting.
