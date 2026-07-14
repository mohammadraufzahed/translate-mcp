# Contributing

Thank you for your interest in contributing to `translate-mcp`. This project follows standard Go conventions and uses GitHub pull requests.

## Getting started

1. Fork the repository and clone your fork.
2. Install Go 1.25 or later.
3. Run `go mod download`.
4. Build with `go build ./...`.
5. Copy `config.example.yaml` to `config.yaml` for local testing.

## Code style

- Run `gofmt -w .` before committing.
- Run `go vet ./...`.
- Keep packages focused and interfaces small.
- Prefer explicit errors and structured logging.
- Do not log or print API keys.

## Testing

```bash
go test -race ./...
```

Add tests for new providers, cache backends, and tool handlers. Use `httptest` for provider HTTP clients and mock dependencies where possible.

## Documentation

If your change affects the public interface, configuration, or deployment, update the relevant files:

- [`README.md`](README.md)
- [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)
- [`docs/PROVIDERS.md`](docs/PROVIDERS.md)
- [`docs/TOOLS.md`](docs/TOOLS.md)
- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)

## Pull requests

1. Create a feature branch from `main`.
2. Make focused, well-scoped changes.
3. Update documentation if the public interface or configuration changes.
4. Ensure `gofmt -w .`, `go vet ./...`, and `go test -race ./...` pass.
5. Open a pull request with a clear description and motivation.

## Release process

Maintainers push semver tags to trigger the release workflow:

```bash
git checkout main
git pull
git tag v1.0.0
git push origin v1.0.0
```

GoReleaser and the Docker workflow handle binaries, installers, and container images.

## Code of conduct

Be respectful and constructive. We welcome contributors of all backgrounds and experience levels.
