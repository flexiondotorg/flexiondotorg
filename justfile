# Requires just 1.23.0 or later.
import 'just/loader.just'

# List available recipes
default:
    @just --list

# Render and serve a live Vitrine profile preview at http://127.0.0.1:18474
preview:
    #!/usr/bin/env bash
    set -euo pipefail
    url="http://127.0.0.1:18474"
    if ! command -v miniserve >/dev/null 2>&1; then
        echo "Error: miniserve not found. Enter the Nix dev shell (direnv allow or nix develop) first." >&2
        exit 1
    fi
    out="$(mktemp -d -t vitrine-preview.XXXXXX)"
    trap 'rm -rf "${out}"' EXIT
    mkdir -p "${out}/assets"
    if [ -z "${GITHUB_TOKEN:-}" ] && command -v gh >/dev/null 2>&1; then
        GITHUB_TOKEN="$(gh auth token 2>/dev/null || true)"
        export GITHUB_TOKEN
    fi
    echo "Rendering Vitrine preview..."
    go run ./cmd/vitrine \
        -verbose \
        -config vitrine.yaml \
        -template templates/README.md.tpl \
        -out "${out}/README.md" \
        -assets-dir "${out}/assets" \
        -asset-base-url ./assets \
        -preview-html "${out}/index.html"
    echo ""
    echo "Vitrine preview ready at ${url} (press Ctrl-C to stop)"
    miniserve --index index.html --interfaces 127.0.0.1 --port 18474 "${out}"

# Build and serve a live wimpys.world portfolio site preview at http://127.0.0.1:18475
site:
    #!/usr/bin/env bash
    set -euo pipefail
    url="http://127.0.0.1:18475"
    if ! command -v miniserve >/dev/null 2>&1; then
        echo "Error: miniserve not found. Enter the Nix dev shell (direnv allow or nix develop) first." >&2
        exit 1
    fi
    out="$(mktemp -d -t wimpys-world-site.XXXXXX)"
    trap 'rm -rf "${out}"' EXIT
    if [ -z "${GITHUB_TOKEN:-}" ] && command -v gh >/dev/null 2>&1; then
        GITHUB_TOKEN="$(gh auth token 2>/dev/null || true)"
        export GITHUB_TOKEN
    fi
    echo "Building wimpys.world portfolio site..."
    go run ./cmd/site \
        -verbose \
        -config vitrine.yaml \
        -out "${out}"
    echo ""
    echo "Portfolio site preview ready at ${url} (press Ctrl-C to stop)"
    miniserve --index index.html --interfaces 127.0.0.1 --port 18475 "${out}"
