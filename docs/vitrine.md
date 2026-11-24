# Vitrine design and architecture

Vitrine is a Go command-line interface (`cmd/vitrine`) that builds dynamic README files for GitHub profiles and animated Scalable Vector Graphics (SVG) tiles. Configuration lives in `vitrine.yaml` and layout lives in `templates/README.md.tpl`. No user-specific values sit in Go source code, so any developer can fork the repository or run the binary with another configuration file.

This design replaces `muesli/readme-scribe`, `github-readme-stats`, third-party badge services for core profile tiles, and `.github/workflows/snake.yml`. Every data fetch runs without a Personal Access Token (PAT) and uses only the built-in `GITHUB_TOKEN` plus public web endpoints.

## Package structure

The Go module is `github.com/flexiondotorg/flexiondotorg`. Vitrine splits CLI orchestration from reusable data and rendering packages under `internal/`. The Track 2 portfolio site generator imports the same data packages so both outputs share one source of truth.

| Path | Responsibility | Shared with portfolio site |
| --- | --- | --- |
| `cmd/vitrine/` | Parses CLI flags, loads configuration, runs fetchers, writes dark and light SVG files, and renders `README.md`. | No |
| `internal/config/` | Parses `vitrine.yaml`, resolves built-in theme presets, and applies custom colour overrides. | Yes |
| `internal/discord/` | Fetches live online and member counts for Discord from public endpoints without API secrets. | Optional |
| `internal/github/` | Fetches user, organisation, stats, contribution, release, star, and public sponsor data. | Yes |
| `internal/haberdasher/` | Fetches `haberdasher.yaml`, repository manifests, and configured standalone projects, merges records, and resolves logos. | Yes |
| `internal/rss/` | Parses Really Simple Syndication (RSS) 2.0 feeds and YouTube Atom feeds. | Yes |
| `internal/twitch/` | Checks live or offline channel status from public endpoints without secrets. | Optional |
| `internal/svg/` | Renders modular animated SVG tiles with Catppuccin and preset theme tokens plus reduced-motion rules. | No |
| `internal/render/` | Executes `templates/README.md.tpl` with `<picture>` helpers and writes final Markdown output. | No |

## Zero-PAT data sources

Vitrine gathers all profile data from unauthenticated public endpoints or the standard repository `GITHUB_TOKEN`. No workflow requires `PERSONAL_GITHUB_TOKEN` or `API_TOKEN_GITHUB`.

### GitHub REST API

Vitrine calls only GitHub Representational State Transfer (REST) endpoints because `GITHUB_TOKEN` cannot run user-scoped GraphQL queries across personal profile data.

| Profile data | REST source | Notes |
| --- | --- | --- |
| Stats strip metrics | `/users/{user}`, `/orgs/{org}`, `/users/{user}/repos`, `/orgs/{org}/repos`, `/search/commits`, and `/search/issues` | Computes deduplicated stars across the user, configured organisations, Haberdasher projects, and top-level `projects:` entries, plus commits, pull requests, `involves:` issues, and followers. |
| Community contributions | `/users/{user}/events/public` and `/search/issues` | Collects external repository contributions outside `exclude_owners` within `contributions_window_days` (365 days in `vitrine.yaml`, 180 days by default). |
| Recent releases | `/repos/{owner}/{repo}/releases` | Collects recent releases across configured owners, Haberdasher projects, and top-level `projects:` entries. |
| Starred repositories | `/users/{user}/starred` | Sends `Accept: application/vnd.github.star+json`, excludes owners listed in `exclude_owners`, and renders open by default. |

### Sponsors

GitHub hides sponsor lists from `GITHUB_TOKEN` on the GraphQL API. Vitrine scrapes the public GitHub Sponsors Hypertext Markup Language (HTML) partials directly:

- **Active sponsors**: Fetches `sponsors_partial?filter=active` and renders each active sponsor as a clickable `@handle` link beside the GitHub Sponsors link.
- **Past sponsors**: Fetches `sponsors_partial?filter=inactive` and renders past sponsor handles on a separate understated line as plain unclickable text.
- **Fallback list**: Applies an optional `sponsors_fallback` list in `vitrine.yaml` when an upstream request fails or markup changes.

### Discord live banner

Vitrine fetches live online counts and total member counts for Discord without bot tokens or OAuth credentials. Resolution follows a three-step chain:

1. Query the public Discord invite API (`https://discord.com/api/v9/invites/<code>?with_counts=true`) for online presence and total member counts.
2. Fall back to the Discord server widget endpoint (`https://discord.com/api/guilds/<guild_id>/widget.json`) for live presence counts.
3. Fall back to the Shields JSON endpoint (`https://img.shields.io/discord/<guild_id>.json`) if both Discord endpoints fail.

### Haberdasher and standalone projects

Vitrine reads the curated project list from `haberdasher.yaml` and fetches all eight package repository manifests:

- `https://apt.wimpys.world/haberdasher.json`
- `https://rpm.wimpys.world/haberdasher.json`
- `https://pkg.wimpys.world/haberdasher.json`
- `https://apk.wimpys.world/haberdasher.json`
- `https://flatpak.wimpys.world/haberdasher.json`
- `https://brew.wimpys.world/haberdasher.json`
- `https://scoop.wimpys.world/haberdasher.json`
- `https://nix.wimpys.world/haberdasher.json`

Use the top-level `projects:` list in `vitrine.yaml` to add non-Haberdasher repositories or to override fields on Haberdasher projects. Each entry requires `repo` in `owner/repo` format and accepts optional `logo`, `language`, `homepage`, and `description` overrides.

Vitrine merges `projects:` with Haberdasher entries by case-insensitive repository name without duplicate cards. When `haberdasher.enabled` is `false`, `projects:` still renders the Projects grid standalone.

For each project, Vitrine fetches GitHub REST API metadata (description, star count, primary language, homepage, latest release tag, and release timestamp). Cards without a homepage link to the GitHub repository URL. Vitrine sorts the merged grid descending by star count by default, with latest release date and project name as tie-breakers. Set `haberdasher.sort` in `vitrine.yaml` to `stars` or `release` to control the sort order.

### Logo resolution order

GitHub blocks external image references inside rendered SVGs, so Vitrine embeds every project logo inline during generation. Logo resolution follows an eight-step fallback chain:

1. Fetch `pages/icon.svg` from the project repository and inline its SVG markup.
2. Fetch `pages/icon.png` from the project repository and embed it as a base64 data Uniform Resource Identifier (URI).
3. Fetch `icon.svg` from the repository root and inline its SVG markup.
4. Fetch `icon.png` from the repository root and embed it as a base64 data URI.
5. Resolve a per-project `logo` from `projects:` in `vitrine.yaml` as an HTTP or HTTPS URL, an existing local file, or a repository-relative file path on the default branch.
6. Fetch a configured PNG asset from `png_fallbacks` in `vitrine.yaml` (for example, `assets/ia-get.png` in `wimpysworld/ia-get`) and embed it as a base64 data URI.
7. Apply a per-project entry from `logo_overrides` in `vitrine.yaml`.
8. Generate a placeholder SVG mark (an acoustic waveform for `linuxmatters/jive-*` repositories or an initial monogram) when no logo exists.

### Package repositories shelf

Vitrine builds an eight-tile shelf for `apt`, `rpm`, `pkg`, `apk`, `flatpak`, `brew`, `scoop`, and `nix` on `.wimpys.world`. Each tile links to `https://<host>`, displays an unframed `40x40` host logo from `https://<host>/logo.svg`, and shows the active package count from that host's `haberdasher.json` in a Catppuccin or gilt count badge.

### Feeds and Twitch

RSS, YouTube, and Twitch sources are configurable in `vitrine.yaml`:

- **Podcast (`PODCAST`)**: Reads the Linux Matters RSS 2.0 feed (`https://linuxmatters.sh/episode/index.xml`) and renders episode numbers, durations, relative dates, and subtitles beside the podcast cover image.
- **Blog**: Reads the Wimpy's World RSS 2.0 feed (`https://wimpysworld.com/posts/rss.xml`) and renders post titles, estimated reading times, relative dates, and summaries.
- **YouTube Atom feeds**: Parses `<published>` rather than `<updated>`, so edits to old video metadata do not push archive videos to the top.
- **Twitch status**: Queries `https://decapi.me/twitch/uptime/<channel>` first and falls back to `https://img.shields.io/twitch/status/<channel>.json` without Twitch API credentials.
- **Combined `VIDEOS & STREAMS` section**: Places the Twitch live or offline status button below recent YouTube videos under `VIDEOS & STREAMS`. The heading switches to `VIDEOS` when only YouTube is enabled, or `STREAMS` when only Twitch is enabled.

## Visual design and GitHub SVG rules

GitHub renders SVG files in `README.md` only through `<img>` tags in secure animated mode. That mode disables internal SVG `<a>` links, `:hover` pseudo-classes, JavaScript execution, external image references, and external web fonts.

Vitrine works within those limits by generating dual-theme animated SVG pairs (`-dark.svg` and `-light.svg`) for every tile. `templates/README.md.tpl` wraps each pair in an HTML `<picture>` element inside an outer `<a href="...">` link so GitHub selects the matching theme automatically while every tile stays clickable and self-contained.

### Catppuccin Mocha and Latte palettes

The `flexiondotorg` profile uses **Catppuccin Mocha** for GitHub dark mode (`-dark.svg`) and **Catppuccin Latte** for GitHub light mode (`-light.svg`). Both variants share identical geometry, crisp contrast against GitHub canvases, and clean Catppuccin accent colours.

| Role | Catppuccin Mocha (`-dark.svg`) | Catppuccin Latte (`-light.svg`) |
| --- | --- | --- |
| Primary card canvas | Base (`#1e1e2e`) | Base (`#eff1f5`) |
| Elevated surface | Surface0 (`#313244`) | Mantle (`#e6e9ef`) |
| Primary ink | Text (`#cdd6f4`) | Text (`#4c4f69`) |
| Secondary ink | Subtext1 (`#bac2de`) | Subtext1 (`#5c5f77`) |
| Muted ink | Overlay0 (`#6c7086`) | Overlay0 (`#8c8fa1`) |
| Warm accent gradient | Yellow, Peach, Maroon (`#f9e2af`, `#fab387`, `#eba0ac`) | Yellow, Peach, Maroon (`#df8e1d`, `#fe640b`, `#e64553`) |
| Aurora wave accents | Mauve, Lavender, Blue, Teal (`#cba6f7`, `#b4befe`, `#89b4fa`, `#94e2d5`) | Mauve, Lavender, Blue, Teal (`#8839ef`, `#7287fd`, `#1e66f5`, `#179299`) |
| Positive accent | Green (`#a6e3a1`) | Green (`#40a02b`) |
| Alert or live accent | Red (`#f38ba8`) | Red (`#d20f39`) |
| Info or link accent | Sapphire (`#74c7ec`) | Sapphire (`#209fb5`) |

### Built-in theme presets

Vitrine ships with three built-in presets selectable in `vitrine.yaml`, plus field-level custom hex overrides for any token:

- `catppuccin-mocha` (active dark variant for `flexiondotorg/flexiondotorg`)
- `catppuccin-latte` (active light variant for `flexiondotorg/flexiondotorg`)
- `obsidian-gilt` (optional dark collector-card preset retained for custom profiles)

### Typography, borders, and logos

Every tile uses clean modern geometry and typography:

- **Typography**: Sans-serif headings and body labels specify `'Work Sans'` first with system sans-serif fallbacks (`'Work Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif`). Monospace metrics, version tags, and domains specify `'Fira Code'` first with system monospace fallbacks (`'Fira Code', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace`). Because GitHub blocks external font files inside `<img>` SVGs, local installations show Work Sans and Fira Code while other viewers fall back cleanly to system fonts.
- **Single crisp borders**: Every card, banner, pill, and divider uses a single `1.5px` rounded border. Tiles omit double outlines, triple outlines, and corner studs.
- **Unframed logos**: Project cards display `64x64` project logos directly on the card canvas with no medallion frame. Package repository tiles display `40x40` host logos unframed beside the repository title and count badge.
- **Hero aurora waves**: The hero banner renders animated Catppuccin Harmonic Aurora Waves across the canvas with the title, subtitle, and a `wimpys.world` pill.

### Profile tile layout

The generated `README.md` arranges sections in this order:

1. **Hero banner SVG**: Full-width banner with Catppuccin Harmonic Aurora Waves, title, subtitle (no eyebrow), and `wimpys.world` pill linking to `https://wimpys.world`.
2. **Social navigation pills**: Three larger animated pill tiles for Mastodon, Bluesky, and LinkedIn with authentic brand vector paths.
3. **Short bio**: Concise Markdown introduction covering Quickemu, Linux Matters, Wimpy's World, and Oval Tutu (without an Ubuntu MATE mention).
4. **Discord live banner**: Dynamic community banner showing live online presence and total member count from the invite API, widget, or Shields fallback.
5. **Stats strip**: Five-metric SVG summary showing deduplicated Stars (across configured organisations and Haberdasher), Commits, Pull Requests, `involves:` Issues, and Followers.
6. **Sponsors lines**: Active sponsors as clickable links from `sponsors_partial?filter=active` with the GitHub Sponsors link, plus a second line of unclickable past sponsor names from `sponsors_partial?filter=inactive`.
7. **Projects**: Two-column grid of `408x182` cards combining Haberdasher projects and top-level `projects:` entries, sorted descending by GitHub star count by default (configurable via `haberdasher.sort: stars` or `release`). Each card shows an unframed `64x64` logo, title, up to four lines of description, version pill, primary language, star count, and release age (with no `N/8` host dots).
8. **Package Repositories**: Eight clickable tiles for the `.wimpys.world` package hosts with unframed `40x40` logos and Catppuccin or gilt count badges.
9. **Community Contributions**: External repositories outside `exclude_owners` from the configured `contributions_window_days` window (365 days in `vitrine.yaml`, 180 days by default).
10. **Recent Releases**: Latest releases across configured owners, Haberdasher projects, and top-level `projects:` entries.
11. **Starred**: Open list of recent starred repositories excluding `exclude_owners`.
12. **Podcast (`PODCAST`)**: Linux Matters cover art alongside recent episodes showing episode number, duration, age, and subtitle.
13. **Blog**: Recent posts from the Wimpy's World blog RSS feed showing estimated reading time, age, and summary.
14. **Videos & Streams (`VIDEOS & STREAMS`)**: Recent YouTube videos with the Twitch live or offline status button below (titled `VIDEOS` or `STREAMS` when only one source is enabled).

Vitrine replaces the contribution snake workflow (`.github/workflows/snake.yml`), so the repository drops both that workflow and its `snake` branch.

### Animation and asset rules

Every SVG emitted by `internal/svg` follows five strict rules:

- Animate only GPU-friendly Cascading Style Sheets (CSS) `@keyframes` properties: `transform`, `opacity`, and `stroke-dashoffset`.
- Apply staggered GPU-only top-edge glints across cards, banners, pills, and the package shelf instead of face shimmers. Section dividers use circle ornaments instead of diamonds, and the hero banner animates Catppuccin Harmonic Aurora Waves.
- Include `@media (prefers-reduced-motion: reduce)` in every animated SVG to stop animation and present a calm static frame.
- Embed all logos, icons, and raster art inline as SVG geometry or base64 data URIs.
- Declare `'Work Sans'` and `'Fira Code'` with system font fallbacks instead of external font imports.

## Workflows and organisation sync

Two small workflows keep the personal profile and the `wimpysworld` organisation profile in sync with zero cross-repository tokens.

### Profile update workflow

The workflow in `.github/workflows/vitrine.yml` replaces `readme-scribe.yml` and `snake.yml`:

- **Triggers**: Scheduled cron every 6 hours (`0 */6 * * *`), pushes to `main` that touch `cmd/vitrine/**`, `internal/**`, `templates/README.md.tpl`, `vitrine.yaml`, `go.mod`, `go.sum`, or `.github/workflows/vitrine.yml`, and `workflow_dispatch`.
- **Permissions**: Top-level `permissions: {}` with job-level `contents: write` using only the built-in `GITHUB_TOKEN`.
- **Asset publishing**: Compares generated `-dark.svg` and `-light.svg` files against `origin/assets`. When any SVG differs or `origin/assets` does not exist yet, the workflow force-pushes a single-commit orphan `assets` branch containing only the generated SVG files and records its commit SHA. When no SVG changes, the workflow reuses the existing `origin/assets` commit SHA without pushing.
- **README update**: Substitutes the recorded `assets` commit SHA into `raw.githubusercontent.com` URLs (`https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/<sha>/<tile>-dark.svg` and `-light.svg`) inside `<picture>` tags and commits `README.md` to `main` only when the rendered Markdown changes.

Commit-hashed `raw.githubusercontent.com` URLs solve three problems at once. They bust the GitHub `camo` image proxy cache whenever a tile changes, they keep `main` free of generated SVG churn, and they render identically when `README.md` is mirrored into another repository.

### Mirror workflow in `wimpysworld/.github`

Instead of pushing across repositories with a personal token, `wimpysworld/.github` pulls the rendered profile directly:

- **Triggers**: Scheduled cron every 6 hours offset by 10 minutes (`10 */6 * * *`), plus `workflow_dispatch`.
- **Permissions**: Top-level `permissions: {}` with job-level `contents: write` using the local `GITHUB_TOKEN` of `wimpysworld/.github`.
- **Behaviour**: Fetches `README.md` from `flexiondotorg/flexiondotorg` and commits it to `profile/README.md` only when content differs.
- **Repository boundary**: The workflow adds no community-health files (such as `CONTRIBUTING.md`, `SECURITY.md`, or issue templates) to `wimpysworld/.github`, because organisation `.github` repositories inject those defaults into other repositories under `wimpysworld`.

## Resolved design decisions

All earlier open questions are now settled in implementation:

- **Refresh cadence**: The profile update workflow runs every 6 hours (`0 */6 * * *`), and the `wimpysworld/.github` mirror runs 10 minutes later (`10 */6 * * *`).
- **Assets branch history**: The `assets` branch holds a single orphan commit that updates only when generated SVG files differ from `origin/assets` (or on the first run). Unchanged runs reuse the existing `origin/assets` commit SHA.
- **Social navigation pills**: The navigation row displays three animated pill tiles for Mastodon, Bluesky, and LinkedIn with authentic brand vector paths.
