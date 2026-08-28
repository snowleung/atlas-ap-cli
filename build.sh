#!/usr/bin/env bash

set -euo pipefail

ARCH="arm64"
OUT="dist"
VERSION="dev"

usage() {
    cat <<'EOF'
Usage: ./build.sh [-arch arm64|amd64] [-out directory] [-version version]

Builds the atlas-ap-remote macOS command-line binary.
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -arch)
            [[ $# -ge 2 ]] || { echo "missing value for -arch" >&2; exit 2; }
            ARCH="$2"
            shift 2
            ;;
        -out)
            [[ $# -ge 2 ]] || { echo "missing value for -out" >&2; exit 2; }
            OUT="$2"
            shift 2
            ;;
        -version)
            [[ $# -ge 2 ]] || { echo "missing value for -version" >&2; exit 2; }
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "unknown option: $1" >&2
            usage >&2
            exit 2
            ;;
    esac
done

case "$ARCH" in
    arm64|amd64) ;;
    *)
        echo "unsupported architecture: $ARCH (use arm64 or amd64)" >&2
        exit 2
        ;;
esac

mkdir -p "$OUT"
OUTPUT="$OUT/atlas-ap-remote"

echo "Building darwin/$ARCH -> $OUTPUT"
GOOS=darwin GOARCH="$ARCH" CGO_ENABLED=0 \
    go build -trimpath \
    -ldflags "-s -w -X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=$VERSION" \
    -o "$OUTPUT" ./cmd/atlas-ap-remote

chmod +x "$OUTPUT"
echo "Built $OUTPUT (version: $VERSION)"
