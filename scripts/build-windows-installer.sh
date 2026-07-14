#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTDIR="${ROOT}/dist"
mkdir -p "${OUTDIR}"

for arch in amd64 arm64; do
    workdir="${OUTDIR}/windows-${arch}"
    mkdir -p "${workdir}"

    echo "Building Windows ${arch} binary..."
    GOOS=windows GOARCH="${arch}" CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "${workdir}/translate-mcp.exe" \
        "${ROOT}/cmd/translate-mcp"

    cp "${ROOT}/README.md" "${ROOT}/LICENSE" "${ROOT}/config.example.yaml" "${workdir}/"

    outname="translate-mcp_${VERSION}_windows_${arch}_setup.exe"
    echo "Building Windows ${arch} installer -> ${outname}..."
    makensis \
        -DVERSION="${VERSION}" \
        -DARCH="${arch}" \
        -DOUTFILE="${OUTDIR}/${outname}" \
        -DBINARY="${workdir}/translate-mcp.exe" \
        -DREADME_FILE="${workdir}/README.md" \
        -DLICENSE_FILE="${workdir}/LICENSE" \
        -DCONFIG_FILE="${workdir}/config.example.yaml" \
        "${ROOT}/windows/installer.nsi"
done

echo "Windows installers built:"
ls -la "${OUTDIR}"/*_windows_*_setup.exe
