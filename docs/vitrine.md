# Vitrine design and architecture

Vitrine is a Go CLI (`cmd/vitrine`) that builds dynamic README files for GitHub profiles and animated Scalable Vector Graphics (SVG) tiles. Configuration lives in `vitrine.yaml` and layout lives in `templates/README.md.tpl`. No user-specific values sit in Go source code, so any developer can fork the repository or run the binary with a different configuration file.

This design replaces `muesli/readme-scribe`, `github-readme-stats`, third-party badge services for core profile tiles, and `.github/workflows/snake.yml`. Every data fetch runs without a Personal Access Token (PAT) and uses only the built-in `GITHUB_TOKEN` plus public web endpoints.

## Package structure

The Go module is `github.com/flexiondotorg/flexiondotorg`. Vitrine splits CLI orchestration from reusable data and rendering packages under `internal/`. The Track 2 portfolio site generator imports the same data packages so both outputs share one source of truth.

| Path | Responsibility | Shared with portfolio site |
| --- | --- | --- |
| `cmd/vitrine/` | Parses CLI flags, loads configuration, runs fetchers, writes SVG files, and renders `README.md`. | No |
| `internal/config/` | Parses `vitrine.yaml`, resolves built-in theme presets, and applies custom colour overrides. | Yes |
| `internal/github/` | Fetches user, organisation, event, pull request, release, star, and public sponsor data. | Yes |
| `internal/haberdasher/` | Fetches `haberdasher.yaml` and eight repository manifests, merges project records, and resolves logos. | Yes |
| `internal/rss/` | Parses Really Simple Syndication (RSS) 2.0 feeds and YouTube Atom feeds. | Yes |
| `internal/twitch/` | Checks live or offline channel status from public endpoints without secrets. | Optional |
| `internal/svg/` | Renders modular animated SVG tiles with theme tokens and reduced-motion rules. | No |
| `internal/render/` | Executes `templates/README.md.tpl` with template helpers and writes final Markdown output. | No |

## Zero-PAT data sources

Vitrine gathers all profile data using unauthenticated public endpoints or the standard repository `GITHUB_TOKEN`. No workflow requires `PERSONAL_GITHUB_TOKEN` or `API_TOKEN_GITHUB`.

### GitHub REST API

Vitrine calls only GitHub Representational State Transfer (REST) endpoints because `GITHUB_TOKEN` cannot run user-scoped GraphQL queries across personal profile data.

| Profile data | REST source | Notes |
| --- | --- | --- |
| User and organisation stats | `/users/{user}`, `/orgs/{org}`, and `/users/{user}/repos` | Aggregates public repository counts, followers, and stars. |
| Recent contributions | `/users/{user}/events/public` | Filters public events to deduplicated repository contributions. |
| Recent pull requests | `/search/issues` | Queries authored public pull requests sorted by creation date. |
| Recent releases | `/repos/{owner}/{repo}/releases` | Collects recent releases across configured owners and Haberdasher projects. |
| Recent starred repositories | `/users/{user}/starred` | Sends `Accept: application/vnd.github.star+json` to obtain `starred_at` timestamps. |

### Sponsors

GitHub hides sponsor lists from `GITHUB_TOKEN` on the GraphQL API. Vitrine fetches the public `https://github.com/sponsors/<user>` page and parses visible sponsor handles and links from its HTML. An optional static fallback list in `vitrine.yaml` covers transient fetch failures or upstream markup changes. The profile renders sponsors as a single understated line rather than a banner.

### Haberdasher curated projects

Vitrine reads the curated project list from `haberdasher.yaml` and fetches all eight package repository manifests:

- `https://apt.wimpys.world/haberdasher.json`
- `https://rpm.wimpys.world/haberdasher.json`
- `https://pkg.wimpys.world/haberdasher.json`
- `https://apk.wimpys.world/haberdasher.json`
- `https://flatpak.wimpys.world/haberdasher.json`
- `https://brew.wimpys.world/haberdasher.json`
- `https://scoop.wimpys.world/haberdasher.json`
- `https://nix.wimpys.world/haberdasher.json`

For each curated project, Vitrine enriches the entry with metadata from the GitHub REST API (description, star count, primary language, latest release tag, and release timestamp). Vitrine sorts the project cards in descending order by latest release date on GitHub so the most recently shipped software appears first.

### Logo resolution order

GitHub blocks external image references inside rendered SVGs, so Vitrine embeds every project logo inline during generation. Logo resolution follows a four-step fallback chain:

1. Fetch `pages/icon.svg` from the project repository and inline its SVG markup.
2. Fall back to a configured PNG asset from the repository (for example, `assets/ia-get.png` in `wimpysworld/ia-get`) and embed it as a base64 data Uniform Resource Identifier (URI).
3. Apply a logo override per project declared in `vitrine.yaml`.
4. Generate a placeholder SVG mark in Obsidian and Gilt (used for `linuxmatters/jive-*` repositories until dedicated logos exist).

### Package repositories shelf

Vitrine builds an eight-tile shelf for `apt`, `rpm`, `pkg`, `apk`, `flatpak`, `brew`, `scoop`, and `nix` on `.wimpys.world`. Each tile links to `https://<host>` and embeds the host logo fetched from `https://<host>/logo.svg` alongside the active package count read from that host's `haberdasher.json`.

### Feeds and Twitch

RSS, YouTube, and Twitch sources are opt-in in `vitrine.yaml`:

- **RSS 2.0 feeds**: Configured for the Linux Matters podcast (`https://linuxmatters.sh/episode/index.xml`) and the Wimpy's World blog (`https://wimpysworld.com/posts/rss.xml`).
- **YouTube Atom feeds**: Parsed using the `<published>` element rather than `<updated>`, so edits to old video descriptions do not push archive videos to the top.
- **Twitch status**: Uses `https://decapi.me/twitch/uptime/<channel>` as the primary check and `https://img.shields.io/twitch/status/<channel>.json` as the fallback, requiring no Twitch API credentials. Vitrine renders dedicated online and offline button tiles from this status.

Vitrine implements and prototypes both the YouTube and Twitch integrations so any profile can toggle them in `vitrine.yaml`. Both sections stay inactive on Martin's profile configuration.

## Visual design and GitHub SVG rules

GitHub renders SVG files in `README.md` only through `<img>` tags in browser secure animated mode. That mode disables internal SVG `<a>` links, `:hover` pseudo-classes, JavaScript execution, external image references, and external web fonts.

Vitrine works within those limits by generating modular animated SVG tiles and wrapping each `<img>` in an outer HTML `<a href="...">` element in `README.md`. Every tile stays self-contained, clickable as a whole unit, and safe under GitHub's content security policy.

### Obsidian and Gilt palette

The `flexiondotorg` profile uses the **Obsidian and Gilt** palette in both GitHub light and dark modes. Dark collector card stock with an ivory outer rim retains crisp contrast against white and dark GitHub backgrounds without dual light and dark SVG sets.

| Role | Name | Hex values |
| --- | --- | --- |
| Primary card stock | Obsidian | `#121216` |
| Elevated surface | Obsidian elevated | `#19191f` |
| Outer rim and primary ink | Ivory | `#f4efe4` |
| Secondary ink | Stone | `#b3aa99` |
| Muted ink | Ash | `#7d7668` |
| Gilt gold gradient (3-stop) | Gilt | `#f0d890`, `#cfa84f`, `#7a5e25` |
| Positive accent | Emerald | `#4fb885` |
| Alert or live accent | Ruby | `#c9545c` |
| Info or link accent | Sapphire | `#6c9bc9` |

### Built-in theme presets

Vitrine ships with three built-in presets selectable in `vitrine.yaml`, plus field-level custom hex overrides for any token:

- `obsidian-gilt` (default for `flexiondotorg/flexiondotorg`)
- `catppuccin-mocha`
- `catppuccin-latte`

### Profile tile layout

The generated `README.md` arranges sections in this order:

1. **Hero banner SVG**: Full-width animated nameplate that links to `https://wimpys.world`.
2. **Navigation pills**: Compact animated pill tiles linking to key sites and social profiles.
3. **Short bio**: Concise Markdown introduction.
4. **Haberdasher project cards**: Two-column wrapping grid of project tiles sorted descending by latest release date on GitHub.
5. **Package repositories shelf**: Eight clickable tiles for the `.wimpys.world` package repositories.
6. **Stats strip**: Self-rendered SVG summary replacing `github-readme-stats`.
7. **Sponsors line**: One understated Markdown line listing current sponsors and the sponsorship link.
8. **Recent activity**: Recent project contributions and recent releases visible by default, with recent pull requests and recent starred repositories tucked inside collapsible `<details>` sections.
9. **Linux Matters podcast**: Cover art alongside recent episodes from the RSS feed.
10. **Blog**: Recent posts from the blog RSS feed, followed by optional YouTube and Twitch sections when enabled in `vitrine.yaml`.

Vitrine replaces the contribution snake workflow (`.github/workflows/snake.yml`), so the repository drops both that workflow and its `snake` branch.

### Animation and asset rules

Every SVG emitted by `internal/svg` follows four strict rules:

- Animate only GPU-friendly Cascading Style Sheets (CSS) `@keyframes` properties: `transform`, `opacity`, and `stroke-dashoffset`.
- Include `@media (prefers-reduced-motion: reduce)` in every animated SVG to stop animation and present a calm static frame.
- Embed all logos, icons, and raster art inline as SVG geometry or base64 data URIs.
- Use system font stacks or vector `<path>` wordmarks instead of external font imports.

## Workflows and organisation sync

Two small workflows keep the personal profile and the `wimpysworld` organisation profile in sync with zero cross-repository tokens.

### Profile update workflow

The workflow in `flexiondotorg/flexiondotorg` replaces `readme-scribe.yml` and `snake.yml`:

- **Triggers**: Scheduled cron every 6 hours (or 12 hours), pushes to `main` touching Vitrine source, `vitrine.yaml`, or `templates/README.md.tpl`, and `workflow_dispatch`.
- **Permissions**: `contents: write` using only the built-in `GITHUB_TOKEN`.
- **Asset publishing**: Pushes generated SVG tiles to the `assets` branch and records the resulting commit SHA.
- **README update**: Renders `README.md` with commit-hashed `raw.githubusercontent.com` URLs (`https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/<sha>/<tile>.svg`) and commits to `main` only when the rendered Markdown changes.

Commit-hashed `raw.githubusercontent.com` URLs solve three problems at once. They bust the GitHub `camo` image proxy cache whenever a tile changes, they keep `main` free of generated SVG churn, and they render identically when `README.md` is mirrored into another repository.

### Mirror workflow in `wimpysworld/.github`

Instead of pushing across repositories with a personal token, `wimpysworld/.github` pulls the rendered profile directly:

- **Triggers**: Scheduled cron offset 10 minutes after the `flexiondotorg/flexiondotorg` workflow, plus `workflow_dispatch`.
- **Permissions**: `contents: write` using the local `GITHUB_TOKEN` of `wimpysworld/.github`.
- **Behaviour**: Fetches `README.md` from `flexiondotorg/flexiondotorg` and commits it to `profile/README.md` only when content differs.
- **Repository boundary**: The workflow adds no community-health files (such as `CONTRIBUTING.md`, `SECURITY.md`, or issue templates) to `wimpysworld/.github`, because organisation `.github` repositories inject those defaults into other repositories under `wimpysworld`.

## Open questions

- **Refresh cadence**: Choose between a 6-hour and a 12-hour cron schedule for the profile update workflow.
- **Assets branch history**: Decide whether the `assets` branch keeps incremental commits or force-pushes a single orphan commit on each run while updating the commit SHA in `README.md`.
- **Navigation pills**: Finalise the exact set of pill links displayed under the hero banner.
