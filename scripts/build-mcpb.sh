#!/bin/sh
# Build a cross-platform Claude Desktop Extension (.mcpb bundle) for pixera-mcp.
#
# Produces ONE bundle that runs on macOS (universal via lipo), Windows (amd64)
# and Linux (amd64); the manifest's platform_overrides select the right binary
# and force stdio transport for Claude Desktop.
#
#   sh scripts/build-mcpb.sh
set -eu

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
BIN="pixera-mcp"
STAGE="build/mcpb"
OUT_DIR="dist"
OUT="${OUT_DIR}/${BIN}-${VERSION}-universal.mcpb"
LDFLAGS="-s -w -X github.com/medcelerate/pixera-mcp/internal/version.Version=${VERSION}"

build() { # GOOS GOARCH OUTFILE
  echo "  building $1/$2"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -ldflags "$LDFLAGS" -o "$3" ./cmd/pixera-mcp
}

rm -rf "$STAGE"
mkdir -p "$STAGE/server" "$OUT_DIR"

echo "Building binaries..."
if command -v lipo >/dev/null 2>&1; then
  build darwin amd64 "$STAGE/server/${BIN}-darwin-amd64"
  build darwin arm64 "$STAGE/server/${BIN}-darwin-arm64"
  lipo -create -output "$STAGE/server/${BIN}-darwin" \
    "$STAGE/server/${BIN}-darwin-amd64" "$STAGE/server/${BIN}-darwin-arm64"
  rm -f "$STAGE/server/${BIN}-darwin-amd64" "$STAGE/server/${BIN}-darwin-arm64"
else
  echo "  warning: lipo not found; shipping a single-arch macOS binary ($(go env GOARCH))"
  build darwin "$(go env GOARCH)" "$STAGE/server/${BIN}-darwin"
fi
build windows amd64 "$STAGE/server/${BIN}-win32.exe"
build linux amd64 "$STAGE/server/${BIN}-linux"

cp mcpb/manifest.json "$STAGE/manifest.json"
[ -f mcpb/icon.png ] && cp mcpb/icon.png "$STAGE/icon.png"

if command -v mcpb >/dev/null 2>&1; then
  mcpb pack "$STAGE" "$OUT"
elif command -v npx >/dev/null 2>&1; then
  npx --yes @anthropic-ai/mcpb pack "$STAGE" "$OUT"
elif command -v zip >/dev/null 2>&1; then
  echo "mcpb CLI not found; packing with zip (manifest not validated)."
  (cd "$STAGE" && zip -qr - .) > "$OUT"
else
  echo "error: need the mcpb CLI, npx, or zip to pack the bundle" >&2
  exit 1
fi

echo "Wrote ${OUT}"
