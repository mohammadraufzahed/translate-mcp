# Contributing

Thank you for your interest in contributing to `translate-mcp`. This project follows standard Go conventions and uses GitHub pull requests.

## Getting started

1. Fork the repository and clone your fork.
2. Install Go 1.25 or later.
3. Run `go mod download`.
4. Build with `go build ./...`.

## Code style

- Run `gofmt -w .` before committing.
- Run `go vet ./...`.
- Keep packages focused and interfaces small.
- Prefer explicit errors and structured logging.

## Testing

```bash
go test ./...
```

Add tests for new providers, cache backends, and tool handlers.

## Pull requests

1. Create a feature branch from `main`.
2. Make focused, well-scoped changes.
3. Update documentation if the public interface or configuration changes.
4. Ensure `go vet ./...` and `go test ./...` pass.
5. Open a pull request with a clear description and motivation.

## Code of conduct

Be respectful and constructive. We welcome contributors of all backgrounds and experience levels.
