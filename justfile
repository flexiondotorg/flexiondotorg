# Requires just 1.23.0 or later.
import 'just/loader.just'

# List available recipes
default:
    @just --list

# Render and serve a live Vitrine profile preview at http://127.0.0.1:18474
preview:
    #!/usr/bin/env bash
    set -euo pipefail
    out="$(mktemp -d -t vitrine-preview.XXXXXX)"
    trap 'rm -rf "${out}"' EXIT
    mkdir -p "${out}/assets"
    if [ -z "${GITHUB_TOKEN:-}" ] && command -v gh >/dev/null 2>&1; then
        GITHUB_TOKEN="$(gh auth token 2>/dev/null || true)"
        export GITHUB_TOKEN
    fi
    go run ./cmd/vitrine \
        -config vitrine.yaml \
        -template templates/README.md.tpl \
        -out "${out}/README.md" \
        -assets-dir "${out}/assets" \
        -asset-base-url ./assets \
        -preview-html "${out}/index.html"
    miniserve --index index.html --interfaces 127.0.0.1 --port 18474 "${out}"
