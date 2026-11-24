package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/haberdasher"
	"github.com/flexiondotorg/flexiondotorg/internal/render"
	"github.com/flexiondotorg/flexiondotorg/internal/svg"
)

func TestVersionFlagSkipsNetwork(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"-version"}, runtimeDeps{
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("run -version error: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != version {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout.String()), version)
	}
}

func TestRunEndToEndOfflineFixtures(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

	var (
		searchMu      sync.Mutex
		searchQueries []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/exampleuser":
			writeTestJSON(t, w, map[string]any{"login": "exampleuser", "followers": 42, "public_repos": 12})
		case "/users/exampleuser/repos":
			writeTestJSON(t, w, []map[string]any{
				{"full_name": "exampleuser/tool", "name": "tool", "owner": map[string]string{"login": "exampleuser"}, "description": "Example CLI", "html_url": "https://github.com/exampleuser/tool", "stargazers_count": 125, "pushed_at": fixedNow.Add(-24 * time.Hour)},
			})
		case "/search/commits":
			writeTestJSON(t, w, map[string]any{"total_count": 3210})
		case "/search/issues":
			q := r.URL.Query().Get("q")
			searchMu.Lock()
			searchQueries = append(searchQueries, q)
			searchMu.Unlock()
			if strings.Contains(q, "type:issue") {
				writeTestJSON(t, w, map[string]any{"total_count": 19, "items": []any{}})
				return
			}
			writeTestJSON(t, w, map[string]any{
				"total_count": 7,
				"items": []map[string]any{
					{"title": "Add tests", "html_url": "https://github.com/external/repo/pull/1", "repository_url": "https://api.github.com/repos/external/repo", "state": "open", "created_at": fixedNow.Add(-48 * time.Hour), "updated_at": fixedNow.Add(-24 * time.Hour)},
				},
			})
		case "/users/exampleuser/events/public":
			writeTestJSON(t, w, []map[string]any{
				{"type": "PushEvent", "repo": map[string]string{"name": "exampleuser/tool"}, "created_at": fixedNow.Add(-24 * time.Hour)},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "external/repo"}, "created_at": fixedNow.Add(-12 * time.Hour)},
			})
		case "/repos/external/repo":
			writeTestJSON(t, w, map[string]any{
				"full_name": "external/repo", "name": "repo", "owner": map[string]string{"login": "external"},
				"description": "External upstream project", "html_url": "https://github.com/external/repo", "stargazers_count": 500,
			})
		case "/repos/exampleuser/tool":
			writeTestJSON(t, w, map[string]any{
				"full_name": "exampleuser/tool", "name": "tool", "owner": map[string]string{"login": "exampleuser"},
				"description": "Example CLI", "html_url": "https://github.com/exampleuser/tool", "stargazers_count": 125,
			})
		case "/repos/exampleuser/tool/releases/latest":
			writeTestJSON(t, w, map[string]any{
				"tag_name": "v1.0.0", "html_url": "https://github.com/exampleuser/tool/releases/tag/v1.0.0", "published_at": fixedNow.Add(-24 * time.Hour),
			})
		case "/users/exampleuser/starred":
			writeTestJSON(t, w, []map[string]any{
				{"starred_at": fixedNow.Add(-48 * time.Hour), "repo": map[string]any{"full_name": "other/project", "html_url": "https://github.com/other/project", "description": "Neat project", "stargazers_count": 13647}},
			})
		case "/sponsors/exampleuser/sponsors_partial":
			if r.URL.Query().Get("filter") == "inactive" {
				_, _ = w.Write([]byte(`<html><img alt="@alice"><img alt="@bob"></html>`))
				return
			}
			_, _ = w.Write([]byte(`<html></html>`))
		case "/discord/invites/testcode":
			writeTestJSON(t, w, map[string]any{
				"guild":                      map[string]string{"name": "Example Discord"},
				"approximate_presence_count": 180,
				"approximate_member_count":   1450,
			})
		case "/twitch/uptime/examplechannel":
			_, _ = w.Write([]byte("examplechannel is offline"))
		case "/podcast.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel><item><title>Episode 94: Linux Matters</title><link>https://linuxmatters.sh/episode/94</link><pubDate>Mon, 23 Nov 2026 08:00:00 +0000</pubDate><itunes:episode>94</itunes:episode><itunes:duration>35:00</itunes:duration><itunes:subtitle>Weekly Linux news.</itunes:subtitle></item></channel></rss>`))
		case "/blog.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><item><title>First Post</title><link>https://example.com/posts/1</link><pubDate>Mon, 23 Nov 2026 10:00:00 +0000</pubDate></item></channel></rss>`))
		case "/youtube.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom"><entry><title>First Video</title><link rel="alternate" href="https://youtube.com/watch?v=abc123"/><published>2026-11-23T09:00:00+00:00</published></entry></feed>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "vitrine.yaml")
	cfgYAML := `profile:
  name: "Example Dev"
  handle: "exampleuser"
  eyebrow: ""
  hero_title: "EXAMPLE WORLD"
  hero_subtitle: "Open-source tools"
  hero_url: "https://example.com"
  hero_cta: "example.com"
  bio: "Building open-source software."
  links:
    - label: "Mastodon"
      slug: "mastodon"
      url: "https://example.social/@dev"
      icon: "mastodon"
discord:
  enabled: true
  name: "Example Discord"
  tagline: "Friendly community hub"
  url: "https://example.com/discord"
  invite_code: "testcode"
  guild_id: "123456"
  invite_base_url: "` + srv.URL + `/discord/invites"
  widget_base_url: "` + srv.URL + `/discord/guilds"
  shields_base_url: "` + srv.URL + `/discord/shields"
theme:
  dark_preset: "catppuccin-mocha"
  light_preset: "catppuccin-latte"
github:
  username: "exampleuser"
  exclude_owners:
    - "exampleuser"
  max_items: 5
  contributions_window_days: 90
haberdasher:
  enabled: false
feeds:
  podcast:
    enabled: true
    title: "Linux Matters Podcast"
    url: "` + srv.URL + `/podcast.xml"
    site_url: "https://linuxmatters.sh"
    cover_url: "https://example.com/cover.png"
    max_items: 3
    links:
      - label: "Apple Podcasts"
        slug: "apple-podcasts"
        url: "https://geo.itunes.apple.com/podcast/feed/id1682797246"
        icon: "apple-podcasts"
      - label: "Spotify"
        slug: "spotify"
        url: "https://open.spotify.com/show/388W00MbvHGI69accqEHQK"
        icon: "spotify"
      - label: "YouTube"
        slug: "youtube"
        url: "https://www.youtube.com/@Linux-Matters"
        icon: "youtube"
      - label: "RSS"
        slug: "rss"
        url: "https://linuxmatters.sh/episode/index.xml"
        icon: "rss"
  blog:
    enabled: true
    title: "Blog"
    url: "` + srv.URL + `/blog.xml"
    site_url: "https://example.com"
    max_items: 3
  youtube:
    enabled: true
    title: "YouTube"
    url: "` + srv.URL + `/youtube.xml"
    site_url: "https://youtube.com/@example"
    max_items: 3
twitch:
  enabled: true
  channel: "examplechannel"
  url: "https://twitch.tv/examplechannel"
  primary_base_url: "` + srv.URL + `/twitch/uptime"
  fallback_base_url: "` + srv.URL + `/twitch/shields"
output:
  asset_base_url: "./assets"
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	tmplPath := filepath.Join("..", "..", "templates", "README.md.tpl")
	outPath := filepath.Join(dir, "README.md")
	assetsDir := filepath.Join(dir, "assets")
	previewPath := filepath.Join(dir, "preview.html")

	ghClient := &gh.Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		RawBaseURL: srv.URL,
		WebBaseURL: srv.URL,
	}

	beforeRun := time.Now().UTC()
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{
		"-config", cfgPath,
		"-template", tmplPath,
		"-out", outPath,
		"-assets-dir", assetsDir,
		"-preview-html", previewPath,
	}, runtimeDeps{
		now:          func() time.Time { return fixedNow },
		httpClient:   srv.Client(),
		githubClient: ghClient,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	afterRun := time.Now().UTC()
	if err != nil {
		t.Fatalf("run error: %v (stderr: %s)", err, stderr.String())
	}

	cutoffBefore := "updated:>=" + beforeRun.AddDate(0, 0, -90).Format("2006-01-02")
	cutoffAfter := "updated:>=" + afterRun.AddDate(0, 0, -90).Format("2006-01-02")
	sawWindowCutoff := false
	searchMu.Lock()
	for _, q := range searchQueries {
		if strings.Contains(q, cutoffBefore) || strings.Contains(q, cutoffAfter) {
			sawWindowCutoff = true
			break
		}
	}
	searchMu.Unlock()
	if !sawWindowCutoff {
		t.Errorf("expected /search/issues contribution query with 90-day cutoff (%s or %s), got queries: %v", cutoffBefore, cutoffAfter, searchQueries)
	}

	readmeBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	readme := string(readmeBytes)
	for _, want := range []string{
		`<picture><source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg"><img src="./assets/hero-light.svg" alt="EXAMPLE WORLD" width="840"></picture>`,
		`./assets/nav-mastodon-dark.svg`,
		`./assets/discord-dark.svg`,
		`./assets/stats-dark.svg`,
		`<a href="https://github.com/sponsors/exampleuser"><picture><source media="(prefers-color-scheme: dark)" srcset="./assets/sponsors-dark.svg"><img src="./assets/sponsors-light.svg" alt="Sponsor on GitHub" width="840"></picture></a>`,
		`./assets/section-podcast-dark.svg`,
		`<br clear="right" />`,
		`<a href="https://geo.itunes.apple.com/podcast/feed/id1682797246"><picture><source media="(prefers-color-scheme: dark)" srcset="./assets/podcast-apple-podcasts-dark.svg"><img src="./assets/podcast-apple-podcasts-light.svg" alt="Apple Podcasts"></picture></a>`,
		`<a href="https://open.spotify.com/show/388W00MbvHGI69accqEHQK"><picture><source media="(prefers-color-scheme: dark)" srcset="./assets/podcast-spotify-dark.svg"><img src="./assets/podcast-spotify-light.svg" alt="Spotify"></picture></a>`,
		`<a href="https://www.youtube.com/@Linux-Matters"><picture><source media="(prefers-color-scheme: dark)" srcset="./assets/podcast-youtube-dark.svg"><img src="./assets/podcast-youtube-light.svg" alt="YouTube"></picture></a>`,
		`<a href="https://linuxmatters.sh/episode/index.xml"><picture><source media="(prefers-color-scheme: dark)" srcset="./assets/podcast-rss-dark.svg"><img src="./assets/podcast-rss-light.svg" alt="RSS"></picture></a>`,
		`./assets/section-contributions-dark.svg`,
		`./assets/section-releases-dark.svg`,
		`./assets/section-starred-dark.svg`,
		`./assets/section-videos-dark.svg`,
		`alt="VIDEOS &amp; STREAMS"`,
		`./assets/twitch-status-dark.svg`,
		`[**external/repo**](https://github.com/external/repo)`,
		`[**other/project**](https://github.com/other/project) · ` + "`★ 13,647`",
		`**[First Post](https://example.com/posts/1)** (today)`,
		`[First Video](https://youtube.com/watch?v=abc123) (today)`,
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README missing %q:\n%s", want, readme)
		}
	}

	ordered := []string{
		`./assets/discord-dark.svg`,
		`./assets/stats-dark.svg`,
		`./assets/sponsors-dark.svg`,
		`./assets/section-podcast-dark.svg`,
		`<br clear="right" />`,
		`./assets/podcast-apple-podcasts-dark.svg`,
		`./assets/podcast-spotify-dark.svg`,
		`./assets/podcast-youtube-dark.svg`,
		`./assets/podcast-rss-dark.svg`,
		`./assets/section-releases-dark.svg`,
		`./assets/section-contributions-dark.svg`,
		`./assets/section-starred-dark.svg`,
		`./assets/section-blog-dark.svg`,
		`./assets/section-videos-dark.svg`,
		`[First Video](https://youtube.com/watch?v=abc123)`,
		`./assets/twitch-status-dark.svg`,
	}
	prevIdx := -1
	for _, marker := range ordered {
		idx := strings.Index(readme, marker)
		if idx <= prevIdx {
			t.Errorf("README ordering error: %q (pos %d) must follow previous marker (pos %d)", marker, idx, prevIdx)
		}
		prevIdx = idx
	}

	for _, unwanted := range []string{
		`<details>`,
		`Recent pull requests`,
		`@alice`,
		`exampleuser/tool</a>`,
		`section-youtube`,
		`section-twitch`,
	} {
		if strings.Contains(readme, unwanted) {
			t.Errorf("README must not contain %q:\n%s", unwanted, readme)
		}
	}

	expectedAssets := []string{
		"hero-light.svg", "hero-dark.svg",
		"nav-mastodon-light.svg", "nav-mastodon-dark.svg",
		"discord-light.svg", "discord-dark.svg",
		"stats-light.svg", "stats-dark.svg",
		"sponsors-light.svg", "sponsors-dark.svg",
		"section-podcast-light.svg", "section-podcast-dark.svg",
		"podcast-apple-podcasts-light.svg", "podcast-apple-podcasts-dark.svg",
		"podcast-spotify-light.svg", "podcast-spotify-dark.svg",
		"podcast-youtube-light.svg", "podcast-youtube-dark.svg",
		"podcast-rss-light.svg", "podcast-rss-dark.svg",
		"section-contributions-light.svg", "section-contributions-dark.svg",
		"section-releases-light.svg", "section-releases-dark.svg",
		"section-starred-light.svg", "section-starred-dark.svg",
		"section-blog-light.svg", "section-blog-dark.svg",
		"section-videos-light.svg", "section-videos-dark.svg",
		"twitch-status-light.svg", "twitch-status-dark.svg",
		"twitch-live-demo-light.svg", "twitch-live-demo-dark.svg",
		"twitch-offline-demo-light.svg", "twitch-offline-demo-dark.svg",
	}
	for _, asset := range expectedAssets {
		if _, err := os.Stat(filepath.Join(assetsDir, asset)); err != nil {
			t.Errorf("expected generated asset %s: %v", asset, err)
		}
	}
	for _, forbidden := range []string{"section-youtube-light.svg", "section-twitch-light.svg"} {
		if _, err := os.Stat(filepath.Join(assetsDir, forbidden)); !os.IsNotExist(err) {
			t.Errorf("unexpected legacy divider asset %s exists", forbidden)
		}
	}
	heroDarkSVG, err := os.ReadFile(filepath.Join(assetsDir, "hero-dark.svg"))
	if err != nil {
		t.Fatalf("read hero-dark.svg: %v", err)
	}
	if !strings.Contains(string(heroDarkSVG), "#1e1e2e") {
		t.Errorf("hero-dark.svg missing Catppuccin Mocha base #1e1e2e")
	}
	heroLightSVG, err := os.ReadFile(filepath.Join(assetsDir, "hero-light.svg"))
	if err != nil {
		t.Fatalf("read hero-light.svg: %v", err)
	}
	if !strings.Contains(string(heroLightSVG), "#eff1f5") {
		t.Errorf("hero-light.svg missing Catppuccin Latte base #eff1f5")
	}

	videosSVG, err := os.ReadFile(filepath.Join(assetsDir, "section-videos-dark.svg"))
	if err != nil {
		t.Fatalf("read section-videos-dark.svg: %v", err)
	}
	if !strings.Contains(string(videosSVG), "VIDEOS &amp; STREAMS") {
		t.Errorf("section-videos-dark.svg missing VIDEOS &amp; STREAMS heading:\n%s", string(videosSVG))
	}

	sponsorsDarkSVG, err := os.ReadFile(filepath.Join(assetsDir, "sponsors-dark.svg"))
	if err != nil {
		t.Fatalf("read sponsors-dark.svg: %v", err)
	}
	for _, want := range []string{"#1e1e2e", `stroke="#45475a" stroke-width="1.5"`, "alice, bob", "Sponsor on GitHub"} {
		if !strings.Contains(string(sponsorsDarkSVG), want) {
			t.Errorf("sponsors-dark.svg missing %q", want)
		}
	}
	sponsorsLightSVG, err := os.ReadFile(filepath.Join(assetsDir, "sponsors-light.svg"))
	if err != nil {
		t.Fatalf("read sponsors-light.svg: %v", err)
	}
	for _, want := range []string{"#eff1f5", `stroke="#bcc0cc" stroke-width="1.5"`, "alice, bob", "Sponsor on GitHub"} {
		if !strings.Contains(string(sponsorsLightSVG), want) {
			t.Errorf("sponsors-light.svg missing %q", want)
		}
	}

	previewBytes, err := os.ReadFile(previewPath)
	if err != nil {
		t.Fatalf("read preview.html: %v", err)
	}
	preview := string(previewBytes)
	darkIdx := strings.Index(preview, `id="github-dark"`)
	lightIdx := strings.Index(preview, `id="github-light"`)
	if darkIdx < 0 || lightIdx < 0 || lightIdx <= darkIdx {
		t.Fatalf("preview.html missing github-dark/github-light panels")
	}
	darkPanel := preview[darkIdx:lightIdx]
	lightPanel := preview[lightIdx:]

	if strings.Contains(darkPanel, "-light.svg") || strings.Contains(darkPanel, "<picture>") {
		t.Errorf("dark preview panel must display -dark.svg directly without <picture> or -light.svg:\n%s", darkPanel)
	}
	if !strings.Contains(darkPanel, `./assets/hero-dark.svg`) || !strings.Contains(darkPanel, `./assets/sponsors-dark.svg`) || !strings.Contains(darkPanel, `./assets/podcast-apple-podcasts-dark.svg`) || !strings.Contains(darkPanel, `./assets/twitch-live-demo-dark.svg`) {
		t.Errorf("dark preview panel missing -dark.svg assets")
	}
	if strings.Contains(lightPanel, "-dark.svg") || strings.Contains(lightPanel, "<picture>") {
		t.Errorf("light preview panel must display -light.svg directly without <picture> or -dark.svg:\n%s", lightPanel)
	}
	if !strings.Contains(lightPanel, `./assets/hero-light.svg`) || !strings.Contains(lightPanel, `./assets/sponsors-light.svg`) || !strings.Contains(lightPanel, `./assets/podcast-apple-podcasts-light.svg`) || !strings.Contains(lightPanel, `./assets/twitch-offline-demo-light.svg`) {
		t.Errorf("light preview panel missing -light.svg assets")
	}

	podcastPillChecks := []struct {
		file       string
		accent     string
		pathPrefix string
	}{
		{"podcast-apple-podcasts-dark.svg", "#cba6f7", "M5.34 0A5.328 5.328"},
		{"podcast-spotify-dark.svg", "#a6e3a1", "M12 0C5.4 0 0 5.4"},
		{"podcast-youtube-dark.svg", "#f38ba8", "M23.498 6.186a3.016"},
		{"podcast-rss-dark.svg", "#fab387", "M19.199 24C19.199 13.467"},
		{"podcast-spotify-light.svg", "#40a02b", "M12 0C5.4 0 0 5.4"},
	}
	for _, pc := range podcastPillChecks {
		raw, err := os.ReadFile(filepath.Join(assetsDir, pc.file))
		if err != nil {
			t.Fatalf("read %s: %v", pc.file, err)
		}
		svgText := string(raw)
		if !strings.Contains(svgText, `stroke="`+pc.accent+`" stroke-width="1.5"`) {
			t.Errorf("%s missing accent border %s", pc.file, pc.accent)
		}
		if !strings.Contains(svgText, pc.pathPrefix) {
			t.Errorf("%s missing icon path %q", pc.file, pc.pathPrefix)
		}
	}
}

func TestMarkdownToHTMLAndPreviewTheme(t *testing.T) {
	t.Parallel()

	md := `<p align="center"><picture><source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg"><img src="./assets/hero-light.svg" alt="Hero" width="840"></picture></p>

### Recent releases

- [**org/repo**](https://example.com) (**v1.0**, *today*) with ` + "`go`" + `
  - Nested subtitle
`
	got := markdownToHTML(md)
	for _, want := range []string{
		`<picture><source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg"><img src="./assets/hero-light.svg" alt="Hero" width="840"></picture>`,
		`<h3>Recent releases</h3>`,
		`<ul>`,
		`<a href="https://example.com"><strong>org/repo</strong></a>`,
		`<strong>v1.0</strong>`,
		`<em>today</em>`,
		`<code>go</code>`,
		"<ul>\n<li>Nested subtitle</li>\n</ul>\n</li>",
		`</ul>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("markdownToHTML missing %q:\n%s", want, got)
		}
	}

	dark := resolvePreviewTheme(got, true)
	if !strings.Contains(dark, `<img src="./assets/hero-dark.svg" alt="Hero" width="840">`) || strings.Contains(dark, "<picture>") {
		t.Errorf("resolvePreviewTheme(dark) unexpected:\n%s", dark)
	}
	light := resolvePreviewTheme(got, false)
	if !strings.Contains(light, `<img src="./assets/hero-light.svg" alt="Hero" width="840">`) || strings.Contains(light, "<picture>") {
		t.Errorf("resolvePreviewTheme(light) unexpected:\n%s", light)
	}
}

func TestVideosSectionTitleAndDivider(t *testing.T) {
	t.Parallel()

	theme := config.Theme{
		DarkPreset:  config.PresetCatppuccinMocha,
		LightPreset: config.PresetCatppuccinLatte,
	}
	darkPalette, err := config.ResolveDarkPalette(theme)
	if err != nil {
		t.Fatalf("ResolveDarkPalette error: %v", err)
	}
	lightPalette, err := config.ResolveLightPalette(theme)
	if err != nil {
		t.Fatalf("ResolveLightPalette error: %v", err)
	}
	sr := svgRenderers{
		light: svg.NewRenderer(toSVGPalette(lightPalette)),
		dark:  svg.NewRenderer(toSVGPalette(darkPalette)),
	}

	cases := []struct {
		name           string
		youtubeEnabled bool
		twitchEnabled  bool
		wantHeading    string
		wantSVGText    string
	}{
		{
			name:           "both enabled",
			youtubeEnabled: true,
			twitchEnabled:  true,
			wantHeading:    "VIDEOS & STREAMS",
			wantSVGText:    "VIDEOS &amp; STREAMS",
		},
		{
			name:           "youtube only",
			youtubeEnabled: true,
			twitchEnabled:  false,
			wantHeading:    "VIDEOS",
			wantSVGText:    "VIDEOS",
		},
		{
			name:           "twitch only",
			youtubeEnabled: false,
			twitchEnabled:  true,
			wantHeading:    "STREAMS",
			wantSVGText:    "STREAMS",
		},
		{
			name:           "neither enabled",
			youtubeEnabled: false,
			twitchEnabled:  false,
			wantHeading:    "",
			wantSVGText:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := videosSectionTitle(tc.youtubeEnabled, tc.twitchEnabled); got != tc.wantHeading {
				t.Fatalf("videosSectionTitle(%v, %v) = %q, want %q", tc.youtubeEnabled, tc.twitchEnabled, got, tc.wantHeading)
			}

			cfg := &config.Config{
				Feeds: config.FeedsConfig{
					YouTube: &config.FeedConfig{Enabled: tc.youtubeEnabled},
				},
				Twitch: config.TwitchConfig{Enabled: tc.twitchEnabled},
			}
			data := &render.RenderData{
				SectionBases: make(map[string]string),
				SectionFiles: make(map[string]string),
			}
			written := make(map[string]string)
			writeAsset := func(name string, b []byte) error {
				written[name] = string(b)
				return nil
			}

			if err := renderStatsAndDividers(sr, cfg, writeAsset, data); err != nil {
				t.Fatalf("renderStatsAndDividers error: %v", err)
			}
			if data.VideosHeading != tc.wantHeading {
				t.Errorf("VideosHeading = %q, want %q", data.VideosHeading, tc.wantHeading)
			}
			if tc.wantHeading == "" {
				if len(written) != 0 {
					t.Errorf("expected no divider assets when neither enabled, wrote %v", written)
				}
				return
			}
			if data.SectionBases["videos"] != "section-videos" || data.SectionFiles["videos"] != "section-videos-light.svg" {
				t.Errorf("unexpected SectionBases/Files for videos: %v / %v", data.SectionBases, data.SectionFiles)
			}
			if _, ok := data.SectionBases["youtube"]; ok {
				t.Errorf("unexpected separate youtube section divider")
			}
			if _, ok := data.SectionBases["twitch"]; ok {
				t.Errorf("unexpected separate twitch section divider")
			}
			for _, file := range []string{"section-videos-light.svg", "section-videos-dark.svg"} {
				content, ok := written[file]
				if !ok {
					t.Fatalf("missing rendered file %s", file)
				}
				if !strings.Contains(content, tc.wantSVGText) {
					t.Errorf("%s missing %q:\n%s", file, tc.wantSVGText, content)
				}
			}
		})
	}
}

func TestToSVGPaletteAndStaggeredDelays(t *testing.T) {
	t.Parallel()

	darkCfg, err := config.ResolveDarkPalette(config.Theme{DarkPreset: config.PresetCatppuccinMocha})
	if err != nil {
		t.Fatalf("ResolveDarkPalette error: %v", err)
	}
	lightCfg, err := config.ResolveLightPalette(config.Theme{LightPreset: config.PresetCatppuccinLatte})
	if err != nil {
		t.Fatalf("ResolveLightPalette error: %v", err)
	}

	darkPal := toSVGPalette(darkCfg)
	lightPal := toSVGPalette(lightCfg)
	if darkPal != svg.CatppuccinMochaPalette() {
		t.Errorf("toSVGPalette(mocha) = %+v, want %+v", darkPal, svg.CatppuccinMochaPalette())
	}
	if lightPal != svg.CatppuccinLattePalette() {
		t.Errorf("toSVGPalette(latte) = %+v, want %+v", lightPal, svg.CatppuccinLattePalette())
	}

	sr := svgRenderers{
		light: svg.NewRenderer(lightPal),
		dark:  svg.NewRenderer(darkPal),
	}
	cat := &haberdasher.Catalogue{
		Projects: []haberdasher.Project{
			{Name: "alpha", Slug: "alpha", Description: "First project", RepoURL: "https://github.com/org/alpha"},
			{Name: "beta", Slug: "beta", Description: "Second project", RepoURL: "https://github.com/org/beta"},
		},
		Hosts: []haberdasher.PackageHost{
			{Name: "apt", Host: "apt.wimpys.world", URL: "https://apt.wimpys.world", PackageCount: 4},
			{Name: "rpm", Host: "rpm.wimpys.world", URL: "https://rpm.wimpys.world", PackageCount: 4},
		},
	}
	data := &render.RenderData{}
	written := make(map[string]string)
	writeAsset := func(name string, b []byte) error {
		written[name] = string(b)
		return nil
	}
	if err := renderCatalogueTiles(sr, cat, time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC), writeAsset, data); err != nil {
		t.Fatalf("renderCatalogueTiles error: %v", err)
	}

	if !strings.Contains(written["project-beta-dark.svg"], "0.55s") {
		t.Errorf("project-beta-dark.svg missing 0.55s staggered delay")
	}
	if !strings.Contains(written["host-rpm-dark.svg"], "0.45s") {
		t.Errorf("host-rpm-dark.svg missing 0.45s staggered delay")
	}
	if !strings.Contains(written["project-alpha-dark.svg"], "#1e1e2e") {
		t.Errorf("project-alpha-dark.svg missing Mocha base #1e1e2e")
	}
	if !strings.Contains(written["project-alpha-light.svg"], "#eff1f5") {
		t.Errorf("project-alpha-light.svg missing Latte base #eff1f5")
	}
}

func TestRunProjectsStandaloneAndMerged(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	pngBytes := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("nix-logo")...)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/haberdasher.yaml":
			_, _ = w.Write([]byte("sources: [wimpysworld/tailor]\n"))
		case "/hosts/apt.wimpys.world/haberdasher.json":
			writeTestJSON(t, w, map[string]any{
				"assets": []map[string]any{
					{"identity": map[string]any{"name": "tailor"}, "source": map[string]any{"repository": "wimpysworld/tailor"}},
				},
				"homepages": map[string]string{
					"wimpysworld/tailor": "https://tailor.wimpys.world/",
				},
			})
		case "/hosts/apt.wimpys.world/logo.svg":
			_, _ = w.Write([]byte(`<svg id="apt"></svg>`))
		case "/users/exampleuser":
			writeTestJSON(t, w, map[string]any{"login": "exampleuser", "followers": 10, "public_repos": 1})
		case "/users/exampleuser/repos":
			writeTestJSON(t, w, []map[string]any{})
		case "/search/commits":
			writeTestJSON(t, w, map[string]any{"total_count": 50})
		case "/search/issues":
			writeTestJSON(t, w, map[string]any{"total_count": 0, "items": []any{}})
		case "/users/exampleuser/events/public":
			writeTestJSON(t, w, []any{})
		case "/users/exampleuser/starred":
			writeTestJSON(t, w, []any{})
		case "/sponsors/exampleuser/sponsors_partial":
			_, _ = w.Write([]byte(`<html></html>`))
		case "/repos/wimpysworld/tailor":
			writeTestJSON(t, w, map[string]any{
				"full_name": "wimpysworld/tailor", "name": "tailor", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Tailor tool", "homepage": "https://tailor.wimpys.world/", "html_url": "https://github.com/wimpysworld/tailor",
				"default_branch": "main", "language": "Go", "stargazers_count": 100,
			})
		case "/repos/wimpysworld/tailor/releases/latest":
			writeTestJSON(t, w, map[string]any{
				"tag_name": "v0.6.6", "html_url": "https://github.com/wimpysworld/tailor/releases/tag/v0.6.6", "published_at": fixedNow.Add(-48 * time.Hour),
			})
		case "/repos/wimpysworld/nix-config":
			writeTestJSON(t, w, map[string]any{
				"full_name": "wimpysworld/nix-config", "name": "nix-config", "owner": map[string]string{"login": "wimpysworld"},
				"description": "NixOS and Home Manager configuration", "homepage": "", "html_url": "https://github.com/wimpysworld/nix-config",
				"default_branch": "main", "language": "Shell", "stargazers_count": 250,
			})
		case "/repos/wimpysworld/nix-config/releases/latest":
			writeTestJSON(t, w, map[string]any{
				"tag_name": "v2026.11", "html_url": "https://github.com/wimpysworld/nix-config/releases/tag/v2026.11", "published_at": fixedNow.Add(-12 * time.Hour),
			})
		case "/repos/wimpysworld/nothing-but-nix":
			writeTestJSON(t, w, map[string]any{
				"full_name": "wimpysworld/nothing-but-nix", "name": "nothing-but-nix", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Pure Nix flake showcase", "homepage": "", "html_url": "https://github.com/wimpysworld/nothing-but-nix",
				"default_branch": "main", "language": "Nix", "stargazers_count": 75,
			})
		case "/repos/wimpysworld/nothing-but-nix/releases/latest":
			http.NotFound(w, r)
		case "/wimpysworld/tailor/main/pages/icon.svg":
			_, _ = w.Write([]byte(`<svg id="tailor-icon"></svg>`))
		case "/wimpysworld/nix-config/main/assets/logo.png":
			_, _ = w.Write(pngBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmplPath := filepath.Join("..", "..", "templates", "README.md.tpl")

	// Standalone mode: haberdasher.enabled is false, but top-level projects still render and feed GitHub stats/releases.
	t.Run("standalone without haberdasher", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "vitrine.yaml")
		cfgYAML := `profile:
  name: "Example Dev"
  handle: "exampleuser"
  hero_title: "EXAMPLE"
  hero_url: "https://example.com"
github:
  username: "exampleuser"
  max_items: 5
haberdasher:
  enabled: false
projects:
  - repo: wimpysworld/nix-config
    language: Nix
    logo: assets/logo.png
  - repo: wimpysworld/nothing-but-nix
output:
  asset_base_url: "./assets"
`
		if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		outPath := filepath.Join(dir, "README.md")
		assetsDir := filepath.Join(dir, "assets")
		ghClient := &gh.Client{
			HTTPClient: srv.Client(),
			APIBaseURL: srv.URL,
			RawBaseURL: srv.URL,
			WebBaseURL: srv.URL,
		}
		var stderr bytes.Buffer
		if err := run(context.Background(), []string{
			"-config", cfgPath,
			"-template", tmplPath,
			"-out", outPath,
			"-assets-dir", assetsDir,
		}, runtimeDeps{
			now:          func() time.Time { return fixedNow },
			httpClient:   srv.Client(),
			githubClient: ghClient,
			stderr:       &stderr,
		}); err != nil {
			t.Fatalf("run standalone failed: %v (stderr: %s)", err, stderr.String())
		}

		readmeBytes, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read README.md: %v", err)
		}
		readme := string(readmeBytes)

		nixIdx := strings.Index(readme, `href="https://github.com/wimpysworld/nix-config"`)
		nbnIdx := strings.Index(readme, `href="https://github.com/wimpysworld/nothing-but-nix"`)
		if nixIdx < 0 || nbnIdx < 0 || nixIdx >= nbnIdx {
			t.Fatalf("expected nix-config card before nothing-but-nix card in README:\n%s", readme)
		}
		if !strings.Contains(readme, `[**wimpysworld/nix-config**](https://github.com/wimpysworld/nix-config)`) || !strings.Contains(readme, "v2026.11") {
			t.Errorf("expected nix-config release from ExtraReleaseRepos in README:\n%s", readme)
		}
		if strings.Contains(readme, "section-packages") {
			t.Errorf("standalone projects without haberdasher must not render Package Repositories section")
		}

		statsSVG, err := os.ReadFile(filepath.Join(assetsDir, "stats-dark.svg"))
		if err != nil {
			t.Fatalf("read stats-dark.svg: %v", err)
		}
		if !strings.Contains(string(statsSVG), "325") {
			t.Errorf("expected stats-dark.svg to include 325 stars (250 + 75) from ExtraReleaseRepos:\n%s", string(statsSVG))
		}

		nixSVG, err := os.ReadFile(filepath.Join(assetsDir, "project-nix-config-dark.svg"))
		if err != nil {
			t.Fatalf("read project-nix-config-dark.svg: %v", err)
		}
		if !strings.Contains(string(nixSVG), "data:image/png;base64,") || !strings.Contains(string(nixSVG), ">Nix<") {
			t.Errorf("project-nix-config-dark.svg missing embedded PNG logo or Nix language:\n%s", string(nixSVG))
		}

		nbnSVG, err := os.ReadFile(filepath.Join(assetsDir, "project-nothing-but-nix-dark.svg"))
		if err != nil {
			t.Fatalf("read project-nothing-but-nix-dark.svg: %v", err)
		}
		if strings.Contains(string(nbnSVG), "data:image/png;base64,") || !strings.Contains(string(nbnSVG), ">Nix<") {
			t.Errorf("project-nothing-but-nix-dark.svg expected placeholder monogram and Nix language:\n%s", string(nbnSVG))
		}
	})

	// Merged mode: haberdasher.enabled is true and projects deduplicates an override on tailor while adding nix-config.
	t.Run("merged with haberdasher", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "vitrine.yaml")
		cfgYAML := `profile:
  name: "Example Dev"
  handle: "exampleuser"
  hero_title: "EXAMPLE"
  hero_url: "https://example.com"
github:
  username: "exampleuser"
  max_items: 5
haberdasher:
  enabled: true
  config_url: "` + srv.URL + `/haberdasher.yaml"
  hosts: []
projects:
  - repo: WIMPYSWORLD/TAILOR
    description: "Overridden Tailor description"
  - repo: wimpysworld/nix-config
    language: Nix
    logo: assets/logo.png
output:
  asset_base_url: "./assets"
`
		if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			t.Fatalf("config.Load failed: %v", err)
		}
		ghClient := &gh.Client{
			HTTPClient: srv.Client(),
			APIBaseURL: srv.URL,
			RawBaseURL: srv.URL,
			WebBaseURL: srv.URL,
		}
		fetcher := haberdasher.NewFetcher(srv.Client(), ghClient)
		opts := haberdasher.Options{
			ConfigURL: cfg.Haberdasher.ConfigURL,
			Hosts:     []string{"apt.wimpys.world"},
			Sort:      cfg.Haberdasher.Sort,
			HostBaseURLOverride: func(host string) string {
				return srv.URL + "/hosts/" + host
			},
		}
		cat, err := fetcher.Fetch(context.Background(), opts)
		if err != nil {
			t.Fatalf("Fetch failed: %v", err)
		}
		merged, err := fetcher.EnrichProjects(context.Background(), cat.Projects, toExtraProjects(cfg.Projects), opts)
		if err != nil {
			t.Fatalf("EnrichProjects failed: %v", err)
		}
		if len(merged) != 2 {
			t.Fatalf("expected 2 merged projects without duplicate tailor, got %d: %+v", len(merged), merged)
		}
		if merged[0].FullName != "wimpysworld/nix-config" || merged[1].FullName != "wimpysworld/tailor" {
			t.Fatalf("unexpected merged order: %q, %q", merged[0].FullName, merged[1].FullName)
		}
		if merged[1].Description != "Overridden Tailor description" {
			t.Errorf("expected overridden description on tailor, got %q", merged[1].Description)
		}

		cat.Projects = merged
		names := projectFullNames(cat, cfg.Projects)
		if len(names) != 2 || names[0] != "wimpysworld/nix-config" || names[1] != "wimpysworld/tailor" {
			t.Fatalf("unexpected deduplicated projectFullNames: %v", names)
		}
	})
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}

func TestRenderSponsorsBannerCases(t *testing.T) {
	t.Parallel()

	sr := svgRenderers{
		light: svg.NewRenderer(svg.CatppuccinLattePalette()),
		dark:  svg.NewRenderer(svg.CatppuccinMochaPalette()),
	}

	t.Run("skipped when no profile and no username", func(t *testing.T) {
		t.Parallel()
		data := &render.RenderData{}
		written := make(map[string]string)
		err := renderSponsorsBanner(sr, config.GitHubConfig{}, nil, func(name string, b []byte) error {
			written[name] = string(b)
			return nil
		}, data)
		if err != nil {
			t.Fatalf("renderSponsorsBanner error: %v", err)
		}
		if data.Sponsors != nil || len(written) != 0 {
			t.Fatalf("expected no sponsors output when inactive, got view=%+v files=%v", data.Sponsors, written)
		}
	})

	t.Run("renders when username set even with nil profile", func(t *testing.T) {
		t.Parallel()
		data := &render.RenderData{}
		written := make(map[string]string)
		err := renderSponsorsBanner(sr, config.GitHubConfig{Username: "flexiondotorg"}, nil, func(name string, b []byte) error {
			written[name] = string(b)
			return nil
		}, data)
		if err != nil {
			t.Fatalf("renderSponsorsBanner error: %v", err)
		}
		if data.Sponsors == nil || !data.Sponsors.Enabled || data.Sponsors.URL != "https://github.com/sponsors/flexiondotorg" || data.Sponsors.BaseName != "sponsors" {
			t.Fatalf("unexpected SponsorsView: %+v", data.Sponsors)
		}
		if _, ok := written["sponsors-dark.svg"]; !ok {
			t.Errorf("missing sponsors-dark.svg")
		}
		if _, ok := written["sponsors-light.svg"]; !ok {
			t.Errorf("missing sponsors-light.svg")
		}
	})
}

func TestRenderFeedLinkPills(t *testing.T) {
	t.Parallel()

	sr := svgRenderers{
		light: svg.NewRenderer(svg.CatppuccinLattePalette()),
		dark:  svg.NewRenderer(svg.CatppuccinMochaPalette()),
	}

	feeds := config.FeedsConfig{
		Podcast: &config.FeedConfig{
			Enabled: true,
			Links: []config.NavLink{
				{Label: "Apple Podcasts", URL: "https://example.com/apple", Icon: "apple-podcasts"},
				{Label: "Apple Podcasts", URL: "https://example.com/apple-alt", Icon: "apple-podcasts"},
			},
		},
		Blog: &config.FeedConfig{
			Enabled: true,
			Links: []config.NavLink{
				{Label: "RSS", Slug: "rss", URL: "https://example.com/blog.xml", Icon: "rss"},
			},
		},
		YouTube: &config.FeedConfig{
			Enabled: false,
			Links: []config.NavLink{
				{Label: "YouTube", Slug: "youtube", URL: "https://youtube.com/@example", Icon: "youtube"},
			},
		},
	}
	data := &render.RenderData{
		Podcast: &render.FeedView{Enabled: true},
		Blog:    &render.FeedView{Enabled: true},
		YouTube: &render.FeedView{Enabled: false},
	}

	written := make(map[string]string)
	err := renderFeedLinkPills(sr, feeds, func(name string, b []byte) error {
		written[name] = string(b)
		return nil
	}, data)
	if err != nil {
		t.Fatalf("renderFeedLinkPills error: %v", err)
	}

	if len(data.Podcast.Links) != 2 {
		t.Fatalf("Podcast.Links len = %d, want 2", len(data.Podcast.Links))
	}
	if data.Podcast.Links[0].BaseName != "podcast-apple-podcasts" || data.Podcast.Links[1].BaseName != "podcast-apple-podcasts-2" {
		t.Errorf("unexpected podcast pill base names: %+v", data.Podcast.Links)
	}
	if len(data.Blog.Links) != 1 || data.Blog.Links[0].BaseName != "blog-rss" {
		t.Errorf("unexpected blog pill views: %+v", data.Blog.Links)
	}
	if len(data.YouTube.Links) != 0 {
		t.Errorf("disabled YouTube feed must not populate Links: %+v", data.YouTube.Links)
	}
	for _, wantFile := range []string{
		"podcast-apple-podcasts-dark.svg", "podcast-apple-podcasts-light.svg",
		"podcast-apple-podcasts-2-dark.svg", "podcast-apple-podcasts-2-light.svg",
		"blog-rss-dark.svg", "blog-rss-light.svg",
	} {
		if _, ok := written[wantFile]; !ok {
			t.Errorf("missing expected feed pill asset %s", wantFile)
		}
	}
	if _, ok := written["youtube-youtube-dark.svg"]; ok {
		t.Errorf("disabled YouTube feed must not write pill assets")
	}
}
