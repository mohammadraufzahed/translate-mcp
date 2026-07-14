# Deployment Guide

This guide covers running `translate-mcp` in production with Docker, Docker Compose, systemd, and Kubernetes.

## General recommendations

- Keep the bearer token in a secret manager or environment variable, never in Git.
- Do not expose `/metrics` to the public internet without authentication or a reverse proxy.
- Use Redis L2 when running multiple replicas so caches stay coherent.
- Use Postgres L3 for durable translation memory and long-term cache.
- Set a fallback chain of at least two providers for high availability.

## Docker

```bash
docker run -d \
  --name translate-mcp \
  -p 8787:8787 \
  -e TRANSLATE_MCP_TOKEN=${TRANSLATE_MCP_TOKEN} \
  -e OPENAI_API_KEY=${OPENAI_API_KEY} \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/mohammadraufzahed/translate-mcp:1.0.3 \
  --config /app/config.yaml --transport http
```

The image is distroless and runs as an unprivileged user. Only the HTTP port needs to be exposed.

## Docker Compose

```yaml
services:
  translate-mcp:
    image: ghcr.io/mohammadraufzahed/translate-mcp:1.0.3
    ports:
      - "8787:8787"
    environment:
      - TRANSLATE_MCP_TOKEN=${TRANSLATE_MCP_TOKEN}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - DEEPL_API_KEY=${DEEPL_API_KEY}
      - DATABASE_DSN=${DATABASE_DSN:-postgres://mcp:secret@postgres:5432/mcp?sslmode=disable}
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    depends_on:
      - redis
      - postgres
    command: ["--config", "/app/config.yaml", "--transport", "http", "--port", "8787"]

  redis:
    image: redis:7-alpine
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: mcp
      POSTGRES_USER: mcp
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

Start it:

```bash
docker compose up -d
```

## systemd

1. Create a user and config directory:

   ```bash
   sudo useradd -r -s /sbin/nologin translate-mcp
   sudo mkdir -p /etc/translate-mcp
   sudo cp config.yaml /etc/translate-mcp/config.yaml
   sudo chmod 600 /etc/translate-mcp/config.yaml
   sudo chown -R translate-mcp:translate-mcp /etc/translate-mcp
   ```

2. Create an environment file at `/etc/translate-mcp/env`:

   ```bash
   TRANSLATE_MCP_TOKEN=sk-...
   OPENAI_API_KEY=sk-...
   ```

3. Copy the binary to `/usr/local/bin/translate-mcp`.

4. Copy `systemd/translate-mcp.service` to `/etc/systemd/system/translate-mcp.service`:

   ```ini
   [Unit]
   Description=Advanced Translate MCP Server
   After=network.target

   [Service]
   Type=simple
   ExecStart=/usr/local/bin/translate-mcp --config /etc/translate-mcp/config.yaml
   Restart=on-failure
   EnvironmentFile=/etc/translate-mcp/env
   User=translate-mcp
   Group=translate-mcp

   [Install]
   WantedBy=multi-user.target
   ```

5. Enable and start:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now translate-mcp
   sudo systemctl status translate-mcp
   ```

## Kubernetes

The `kubernetes/` directory contains example manifests.

1. Create a secret from `kubernetes/secret.example.yaml`:

   ```bash
   cp kubernetes/secret.example.yaml kubernetes/secret.yaml
   # edit the values
   kubectl apply -f kubernetes/secret.yaml
   ```

2. Apply the configmap and deployment:

   ```bash
   kubectl apply -f kubernetes/configmap.yaml
   kubectl apply -f kubernetes/deployment.yaml
   ```

3. Expose the service. The included `deployment.yaml` creates a ClusterIP service; add an `Ingress` or `LoadBalancer` as needed.

### Pinning the image tag

The example manifests use `ghcr.io/mohammadraufzahed/translate-mcp:latest`. For production, pin a specific version:

```yaml
image: ghcr.io/mohammadraufzahed/translate-mcp:1.0.3
```

## Reverse proxy

For HTTPS and path-based routing, put `translate-mcp` behind Nginx, Traefik, or Caddy. Example Nginx snippet:

```nginx
server {
    listen 443 ssl;
    server_name translate.example.com;

    location / {
        proxy_pass http://localhost:8787;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Scaling

- The binary is stateless. Scale horizontally behind a load balancer.
- Use Redis L2 so all instances share a cache.
- Keep the Postgres L3 DSN the same across replicas.
- `/metrics` is useful for auto-scaling based on request latency.

## Security checklist

- [ ] Bearer token is set and rotated regularly.
- [ ] Provider API keys are in environment variables or a secret store.
- [ ] `/metrics` is not publicly accessible.
- [ ] CORS `allowed_origins` is restricted to known origins.
- [ ] The host only binds to the intended interface.
- [ ] Logs do not contain sensitive text when `mask_text` is enabled.
