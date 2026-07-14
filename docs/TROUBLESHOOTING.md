# Troubleshooting

This page lists common issues and how to resolve them.

## Server will not start

### `config error: ...`

The YAML is invalid or a required value is missing. Check `config.example.yaml` and run:

```bash
translate-mcp --config config.yaml
```

Look for indentation mistakes and unsupported `transport` values (`stdio` or `http`).

### `init translator: ...`

A provider could not be built. Make sure each configured provider has the required fields, such as `api_key` or `base_url`.

## Translation fails

### `unknown provider`

The provider name in the request does not match a key under `providers`. Names are case-sensitive.

### `provider unavailable`

All providers in the fallback chain failed. Check:

- API keys are set and valid.
- Network connectivity to the provider.
- Provider rate limits or quota.
- Provider `base_url` is correct.

### `rate limited`

The upstream provider returned HTTP 429. Increase rate-limit headroom or add more providers to the fallback chain.

## Cache issues

### Translations are stale

The cache TTL is controlled by `cache.default_ttl`. Shorten the TTL or restart the server to clear the in-memory L1 cache.

### Redis connection fails

Verify `cache.l2.address` and `cache.l2.password`. If Redis is running in Docker, use the container name as the host, e.g. `redis:6379`.

### Postgres connection fails

Verify `cache.l3.dsn`. Example:

```
postgres://user:password@host:5432/db?sslmode=disable
```

## HTTP issues

### `unauthorized`

The request is missing or has the wrong bearer token. Send:

```bash
curl -H "Authorization: Bearer your-token" http://localhost:8787/mcp
```

### CORS errors

Add your origin to `server.cors.allowed_origins` or set it to `["*"]` for development only.

## MCP client issues

### Claude Desktop does not see tools

- Make sure the `command` path is correct.
- Check Claude Desktop logs for errors.
- Run the command manually to see any startup errors.

### HTTP client cannot connect

- Verify the server is running on the expected `host` and `port`.
- Check firewall rules and that the container port is mapped.

## Performance

### High latency

- Enable L1 and L2 caching.
- Check the fallback chain; a failing provider can cause retries.
- Reduce `translation.max_text_length` or batch size.

### High API costs

- Increase cache TTL.
- Add repetitive translations to translation memory.
- Use glossary preprocessing to avoid re-translating protected terms.

## Still stuck?

Open an issue with:

- The exact error message.
- Your sanitized `config.yaml`.
- Server logs (with `mask_text` enabled if the text is sensitive).
- The MCP request that caused the problem.
