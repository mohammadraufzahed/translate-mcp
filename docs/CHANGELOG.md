# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Docker `Build and push` workflow now runs only on semver release tags to reduce CI cost.
- Docker build disables provenance and SBOM attestations to keep GHCR package tags clean.
- Comprehensive documentation rewrite: new README, configuration reference, provider guide, tool reference, deployment guide, architecture docs, troubleshooting, and changelog.

## [1.0.3] - 2026-07-14

### Added

- OpenRouter provider support with OpenAI-compatible API, configurable model IDs, and `HTTP-Referer`/`X-Title` headers.

### Changed

- `openAIProvider` now supports extra provider-specific headers and provider-agnostic error messages.

## [1.0.2] - 2026-07-14

### Changed

- Restricted Docker `Build and push` to release tags only.

## [1.0.1] - 2026-07-14

### Added

- Installable release formats via GoReleaser nFPM and custom CI jobs:
  - `.deb`, `.rpm`, `.apk`, Arch Linux `.pkg.tar.zst`
  - macOS `.dmg`
  - Windows NSIS `.exe` installers

### Changed

- Updated release pipeline to include Windows installer and macOS DMG build jobs.

## [1.0.0] - 2026-07-14

### Added

- Initial release of `translate-mcp`.
- MCP tool server with `stdio` and Streamable HTTP transports.
- Translation providers: OpenAI, Anthropic, DeepL, Google, Ollama, LibreTranslate, custom OpenAI-compatible.
- Multi-tier cache: L1 memory, L2 Redis, L3 SQL.
- Glossary masking and restoration.
- Translation memory with fuzzy search.
- Rate limiting, circuit breakers, and fallback chains.
- Prometheus metrics and health endpoint.
- Bearer token auth and CORS support.
- CI/CD with GitHub Actions, GoReleaser, Docker publishing, and Dependabot.
