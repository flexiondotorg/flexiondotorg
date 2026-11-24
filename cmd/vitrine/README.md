# Vitrine

Vitrine is a Go command-line interface (CLI) that builds dynamic README files for GitHub profiles and animated Scalable Vector Graphics (SVG) tiles. Every tile comes in dark and light variants that follow GitHub colour settings automatically.

Vitrine needs zero long-lived Personal Access Tokens (PATs). It reads GitHub Representational State Transfer (REST) endpoints with the built-in `GITHUB_TOKEN`, alongside unauthenticated public endpoints for sponsors, Discord, feeds, and Twitch.

## What Vitrine builds

Vitrine generates dual `-dark.svg` and `-light.svg` files for each enabled section and wraps them in Hypertext Markup Language (HTML) `<picture>` tags inside `README.md`:

- Hero banner with animated Catppuccin aurora waves, title, subtitle, and link pill.
- Social navigation pills for links such as Mastodon, Bluesky, and LinkedIn.
- Discord community banner with live online and total member counts.
- Five-metric GitHub statistics strip for stars, commits, pull requests, issues, and followers.
- Animated GitHub Sponsors banner (`sponsors-dark.svg` and `sponsors-light.svg`) with floating Catppuccin hearts, wrapped active and past sponsor lists, and a `Sponsor on GitHub` call-to-action pill.
- Two-column Projects grid with inline logos, descriptions, release tags, primary languages, star counts, and release ages.
- Optional Package Repositories shelf from Haberdasher manifests.
- Dynamic lists for community contributions, recent releases, starred repositories, podcasts, blog posts, YouTube videos, and Twitch live status.

## Quick start

Run Vitrine from the repository root with the Go version declared in [go.mod](../../go.mod):

```bash
go run ./cmd/vitrine \
  -config vitrine.yaml \
  -template templates/README.md.tpl \
  -out README.md \
  -assets-dir assets/generated \
  -asset-base-url assets/generated
```

Set `GITHUB_TOKEN` in your environment to avoid low unauthenticated GitHub rate limits.

To preview both GitHub Dark and GitHub Light themes locally in a browser, run:

```bash
just preview
```

This command writes output into a temporary directory and serves the preview at `http://127.0.0.1:18474`.

## CLI flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `-config` | `vitrine.yaml` | Path to the YAML configuration file. |
| `-template` | `templates/README.md.tpl` | Path to the Go text template for `README.md`. |
| `-out` | `README.md` | Output path for the rendered Markdown file. |
| `-assets-dir` | `assets/generated` | Output directory for generated SVG tiles. |
| `-asset-base-url` | `output.asset_base_url` | Base URL or relative path prefix for `<picture>` image URLs. |
| `-preview-html` | `""` | Optional output path for a local HTML preview page for dark and light themes. |
| `-verbose` | `false` | Log fetch and render progress to standard error. |
| `-version` | `false` | Print the binary version and exit. |

## Configuration (`vitrine.yaml`)

All profile content lives in `vitrine.yaml` with zero user-specific values in Go source code. Copy [vitrine.yaml](../../vitrine.yaml) and edit each section for your profile.

### Profile and social links

Set your identity, hero banner text, Markdown bio, and social pills under `profile:`:

```yaml
profile:
  name: "Your Name"
  handle: "your-handle"
  hero_title: "YOUR NAME"
  hero_subtitle: "Short tagline for the hero banner."
  hero_url: "https://example.com"
  hero_cta: "example.com"
  bio: "Short Markdown bio paragraph."
  links:
    - label: "Mastodon"
      slug: "mastodon"
      url: "https://mastodon.social/@you"
      icon: "mastodon"
```

### Theme presets and overrides

Vitrine includes two built-in Catppuccin presets: `catppuccin-mocha` (dark default) and `catppuccin-latte` (light default). You can override any palette token with a `#RRGGBB` hex colour under `theme.overrides`:

```yaml
theme:
  dark_preset: catppuccin-mocha
  light_preset: catppuccin-latte
  overrides:
    accent_primary: "#cba6f7"
```

Supported override keys are `base`, `mantle`, `surface0`, `surface1`, `text`, `subtext`, `muted`, `accent_primary`, `accent_secondary`, `accent_tertiary`, `accent_pink`, `accent_green`, `accent_red`, `accent_peach`, `accent_yellow`, and `accent_sapphire`.

### Discord live banner

Set `discord.enabled: true` with either `invite_code` or `guild_id` and a join `url`. Vitrine queries the public Discord invite API first, then the server widget endpoint, then Shields JSON, with no bot token.

### GitHub statistics, exclusions, and sponsors

Set `github.username` plus optional `orgs` to aggregate deduplicated stars, commits, pull requests, issues, and followers. Add your own user and organisation names to `exclude_owners` so Community Contributions and Starred list only external repositories.

Vitrine reads active and past GitHub Sponsors from the public HTML partials on GitHub Sponsors. Set `github.sponsors_url` and an optional `github.sponsors_fallback` list for resilience.

### Projects and optional Haberdasher

List featured repositories under top-level `projects:` in `owner/repo` format. Each entry accepts optional `logo`, `language`, `homepage`, and `description` overrides:

```yaml
projects:
  - repo: your-org/your-project
    language: Go
    logo: pages/icon.svg
```

Vitrine fetches repository metadata from GitHub REST endpoints. It embeds logos inline from `pages/icon.svg`, `pages/icon.png`, `icon.svg`, `icon.png`, your `logo` override, or a monogram. If you do not publish packages through Haberdasher, keep `haberdasher.enabled: false`. Set `haberdasher.sort` to `stars` (default) or `release` to control project card order.

### Feeds and Twitch

Configure optional Really Simple Syndication (RSS) 2.0 and Atom feeds under `feeds:`:

- `feeds.podcast`: RSS 2.0 podcast feed with optional `cover_url` artwork and optional `links:` subscribe pills (`apple-podcasts`, `spotify`, `youtube`, `rss`, or any supported navigation icon).
- `feeds.blog`: RSS 2.0 blog feed with estimated reading times and summaries.
- `feeds.youtube`: YouTube channel Atom feed. Vitrine parses `<published>` so metadata edits on old videos do not reorder the list.

Any feed under `feeds:` accepts an optional `links:` list with the same `label`, `slug`, `url`, and `icon` fields as `profile.links`. Vitrine generates matching `44px` Catppuccin pills prefixed by the feed name (for example, `podcast-apple-podcasts-dark.svg` and `podcast-apple-podcasts-light.svg`).

Set `twitch.enabled: true` with `twitch.channel` to show a live or offline Twitch button under `VIDEOS & STREAMS`. Vitrine checks `decapi.me` first and falls back to Shields JSON without Twitch API credentials.

## Automated updates with GitHub Actions

Use [.github/workflows/vitrine.yml](../../.github/workflows/vitrine.yml) to refresh your profile automatically on a 6-hour schedule and on configuration changes.

The workflow grants only `contents: write` on the built-in `GITHUB_TOKEN` and needs no extra secrets:

1. Runs `go run ./cmd/vitrine` into a temporary directory.
2. Compares generated SVGs against `origin/assets`. When any SVG changes, it force-pushes an orphan `assets` branch with one commit.
3. Rewrites image URLs in `README.md` with the pinned `assets` commit SHA so GitHub's `camo` proxy refreshes changed tiles immediately.
4. Commits `README.md` to `main` only when the rendered Markdown changes.

See [docs/vitrine.md](../../docs/vitrine.md) for the full architecture, palette tables, SVG animation rules, and organisation mirror workflow.
