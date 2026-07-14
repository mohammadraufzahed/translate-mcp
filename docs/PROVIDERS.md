# Provider Setup

`translate-mcp` supports multiple translation backends. Add each provider under the `providers` section in `config.yaml`.

## Provider selection rules

1. If the MCP request includes `provider`, that provider is used.
2. Otherwise the `translation.default_provider` value is used.
3. If the request includes `model`, it overrides the provider's `default_model`.
4. If translation fails, the server tries the next provider in `translation.fallback_chain`.

## OpenAI

```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o-mini
```

| Field | Description |
|-------|-------------|
| `api_key` | OpenAI API key |
| `default_model` | Any chat model, e.g. `gpt-4o`, `gpt-4o-mini` |
| `base_url` | Optional custom base URL |

Default base URL: `https://api.openai.com/v1`

## OpenRouter

OpenRouter exposes an OpenAI-compatible API and supports models from multiple hosts.

```yaml
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
    default_model: openai/gpt-4o-mini
    site_url: https://github.com/mohammadraufzahed/translate-mcp
    site_name: translate-mcp
```

| Field | Description |
|-------|-------------|
| `api_key` | OpenRouter API key |
| `default_model` | OpenRouter model ID, e.g. `openai/gpt-4o-mini`, `anthropic/claude-3.5-sonnet`, `google/gemini-flash-1.5` |
| `base_url` | Optional custom base URL (default `https://openrouter.ai/api/v1`) |
| `site_url` | Sent as `HTTP-Referer` header |
| `site_name` | Sent as `X-Title` header |

## Anthropic

```yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    default_model: claude-3-5-sonnet-20241022
```

| Field | Description |
|-------|-------------|
| `api_key` | Anthropic API key |
| `default_model` | Claude model ID |
| `base_url` | Optional custom base URL |

## DeepL

```yaml
providers:
  deepl:
    api_key: ${DEEPL_API_KEY}
    base_url: https://api.deepl.com
```

| Field | Description |
|-------|-------------|
| `api_key` | DeepL API key |
| `base_url` | `https://api.deepl.com` (pro) or `https://api-free.deepl.com` (free) |

## Google (Gemini)

```yaml
providers:
  google:
    api_key: ${GOOGLE_API_KEY}
    default_model: gemini-1.5-flash
```

| Field | Description |
|-------|-------------|
| `api_key` | Google Generative Language API key |
| `default_model` | Gemini model ID |

## Ollama

For local or self-hosted models.

```yaml
providers:
  ollama:
    base_url: http://localhost:11434
    default_model: llama3.1
```

No `api_key` is required by default. Make sure the model is pulled and running:

```bash
ollama run llama3.1
```

## LibreTranslate

```yaml
providers:
  libretranslate:
    base_url: https://libretranslate.com
    api_key: ${LIBRETRANSLATE_API_KEY}
```

`api_key` is optional for public instances that do not require one.

## Custom OpenAI-compatible endpoint

Use any endpoint that implements `/chat/completions`.

```yaml
providers:
  custom:
    base_url: http://localhost:8000/v1
    api_key: ${CUSTOM_API_KEY}
    default_model: default
```

## Fallback chain example

```yaml
translation:
  default_provider: openai
  fallback_chain: [openai, openrouter, deepl, ollama]
```

If `openai` fails because of a rate limit or outage, the server automatically tries `openrouter`, then `deepl`, then `ollama`.

## Health checks

Each provider is checked by calling its models or health endpoint. Results appear in `GET /health`:

```json
{
  "status": "ok",
  "providers": {
    "openai": "healthy",
    "deepl": "unhealthy: api key not configured"
  }
}
```
