# flexiondotorg

This repository builds the `flexiondotorg` GitHub profile README through `cmd/vitrine` ([docs/vitrine.md](docs/vitrine.md)) and the `https://wimpys.world` portfolio site ([docs/portfolio-site.md](docs/portfolio-site.md)).

## Setup

- Enter the Nix development shell with `direnv allow`, or run `nix develop` when `direnv` is absent.
- Stage any new file under `nix/` or `flake.nix` with `git add` before flake evaluation, because Nix flakes ignore untracked files.
- Run `direnv reload` after Nix file changes, because `.envrc` enables `nix_direnv_manual_reload`.
- Use the Nix-provided `playwright-mcp` server (`.mcp.json`, headless Chromium) against `just site` to inspect site output.

## Build and test

Run every command from the repository root inside the Nix development shell:

- Build Go binaries into `bin/`: `just build`
- Run `go vet` and atomic-coverage tests: `just test`
- Run race-detector Go tests: `go test -race ./...`
- Check workflows, `go mod tidy`, formatting, `golangci-lint`, `govulncheck`, and `gocyclo`: `just lint`
- Tidy `go.mod` and format Go source with `gofumpt` in place: `just lint correct`
- Validate GitHub Actions workflows only: `actionlint`
- Build and preview the portfolio site at `http://127.0.0.1:18475`: `just site`
- Render and preview the Vitrine profile at `http://127.0.0.1:18474`: `just preview`
- Audit repository health files and GitHub settings against `.tailor.yml`: `tailor measure`
- Preview Tailor template changes without file writes: `tailor baste`

## Code style

- Format Go code with `gofumpt` via `just lint correct`, not plain `gofmt`.
- Satisfy `.golangci.yml` checks in all Go packages, including `noctx` (pass `context.Context` to HTTP requests), `bodyclose`, `errorlint`, `gosec`, `revive`, and `staticcheck`.
- Keep all repository code and documentation compatible with the `BlueOak-1.0.0` licence.

## PR and commit conventions

- Write commit messages in Conventional Commits 1.0.0 format.
- Do not add AI identities to `Signed-off-by` or `Co-authored-by` commit trailers.
- Run `just test`, `just lint`, and `actionlint` before you push changes.
- Merge pull requests with squash or rebase only, because `.tailor.yml` disables merge commits.

## Architecture notes

- Put the reusable profile README generator CLI in `cmd/vitrine/` and place shared fetchers, renderers, and domain packages in `internal/`.
- Keep the profile README refresh workflow and the `https://wimpys.world` build-and-deploy workflow in separate GitHub Actions files.
- Store generated profile README SVGs on the dedicated `assets` branch, never on `main`.
- Do not commit dynamically generated `https://wimpys.world` site HTML to the repository. Build and deploy it in GitHub Actions.
- Put custom build, test, or lint hooks in `just/project/build.sh`, `just/project/test.sh`, or `just/project/lint.sh`. Do not edit Tailor-managed `just/tailor.just`.
- Update `.tailor.yml` instead of files marked `alteration: always` (`SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, and Tailor-managed workflows).

## Companion repository

This repository pairs with the blog repository (`https://blog.wimpys.world`, currently `flexiondotorg/flexiondotorg.github.io`, hosted on Cloudflare Workers Static Assets).

- Keep shared chrome identical across `wimpys.world` (`internal/site/web/`) and `blog.wimpys.world`: sticky frosted header (`min(1320px, 100%)`, `1.5px` aurora bottom hairline, `40px` `W` badge, `Projects`/`Links`/`Media`/`Blog` nav, `184px x 36px` right utility slot, three-way theme switch), three-column frosted footer (`6px` CSS circle pip, copyright, stack credit, `Sponsor ♥`), `scrollbar-gutter: stable`, and `overflow-x: clip`.
- Keep Catppuccin Mocha and Latte tokens, `Work Sans` and `Fira Code` fonts, and the `localStorage` theme key (`wimpys-world-theme`) aligned across both sites.
- Keep the WebGL Balatro swirl clock `(Date.now() / 1000) % (200 * Math.PI)` and static reduced-motion phase `42.0` identical across both sites so cross-site navigation does not reset the background.
- `vitrine.yaml` consumes `https://blog.wimpys.world/posts/rss.xml` for the GitHub profile README and `wimpys.world` Media cards. Do not hardcode legacy `wimpysworld.com` hostnames in `internal/site/web/app.js`.
- Whenever you change the shared header, footer, palette tokens, theme key, swirl clock, or blog link targets here, update the companion blog repository or explicitly flag the matching edit.

## Security and secrets

- Require zero long-lived Personal Access Tokens (PATs) and zero third-party API secrets for automated workflows.
- Authenticate GitHub Actions with `GITHUB_TOKEN` only. Set top-level `permissions: {}` and grant minimal per-job permissions (the repository default is `read`).
- Pin GitHub Actions `uses:` references to full commit SHAs with version comments.
- Call GitHub REST endpoints only. Never call user-scoped GraphQL fields (`viewer`, `contributionsCollection`, `repositoriesContributedTo`, `sponsorshipsAsMaintainer`), because `GITHUB_TOKEN` cannot query them.
- Scrape GitHub Sponsors data from the public sponsors HTML page instead of authenticated GraphQL APIs.
- Treat `wimpysworld/.github` as a pull-mirror only. Never add organisation community-health defaults there, because they override repository health files.

## Gotchas

- Keep `ruleset.rules.pull_request.enabled: false` in `.tailor.yml`. GitHub rejects `GITHUB_TOKEN` in `bypass_actors` with HTTP 422, which blocks scheduled `README.md` commits to `main`.
- Keep `.goreleaser.yaml`, `.github/workflows/build-go.yml`, and `Dockerfile` set to `alteration: never` in `.tailor.yml` so Tailor does not scaffold unused release artefacts.
- Set `pages.enabled: false` in `.tailor.yml` before you switch to a custom scheduled Pages workflow. While `pages.enabled: true`, Tailor enforces an exact match on `.github/workflows/tailor-pages.yml`.
- Render profile README SVGs for `<img>` display only. Omit internal `<a href>`, `:hover`, `<script>`, external URLs, and external fonts.
- Wrap each clickable SVG tile in an outer HTML `<a>` tag in `README.md`, inline all logos, and honour `prefers-reduced-motion`.
- Reference `assets` branch SVGs in `README.md` with commit-hashed URLs so GitHub's `camo` proxy busts cached images on every update.
- Parse `<published>` instead of `<updated>` from YouTube Atom feeds, because YouTube bumps `<updated>` on non-publish metadata edits.
- Fetch live Twitch status from `decapi.me` as the primary source and fall back to Shields JSON to avoid Twitch API secrets.
