#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTDIR="${ROOT}/dist"
mkdir -p "${OUTDIR}"

for arch in amd64 arm64; do
    workdir="${OUTDIR}/darwin-${arch}"
    mkdir -p "${workdir}"

    echo "Building Darwin ${arch} binary..."
    GOOS=darwin GOARCH="${arch}" CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "${workdir}/translate-mcp" \
        "${ROOT}/cmd/translate-mcp"

    cp "${ROOT}/README.md" "${ROOT}/LICENSE" "${ROOT}/config.example.yaml" "${workdir}/"

    dmg="${OUTDIR}/translate-mcp_${VERSION}_darwin_${arch}.dmg"
    echo "Building Darwin ${arch} DMG -> ${dmg}..."
    rm -f "${dmg}"
    hdiutil create -volname "translate-mcp" -srcfolder "${workdir}" -format UDZO -o "${dmg}"
done

echo "macOS DMGs built:"
ls -la "${OUTDIR}"/*_darwin_*.dmg
