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
	"testing"
	"time"

	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/site"
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

	fixedNow := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/haberdasher.yaml":
			_, _ = w.Write([]byte("sources:\n  - wimpysworld/deb-get\n"))
		case "/hosts/apt.wimpys.world/haberdasher.json":
			writeTestJSON(t, w, map[string]any{
				"assets": []map[string]any{
					{
						"identity": map[string]string{"name": "deb-get"},
						"source":   map[string]string{"repository": "wimpysworld/deb-get"},
					},
				},
				"homepages": map[string]string{
					"wimpysworld/deb-get": "https://wimpysworld.github.io/deb-get",
				},
			})
		case "/hosts/apt.wimpys.world/logo.svg":
			_, _ = w.Write([]byte(`<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/></svg>`))
		case "/repos/wimpysworld/deb-get":
			writeTestJSON(t, w, map[string]any{
				"full_name":        "wimpysworld/deb-get",
				"name":             "deb-get",
				"owner":            map[string]string{"login": "wimpysworld"},
				"description":      "apt-get functionality for third-party .debs",
				"homepage":         "https://wimpysworld.github.io/deb-get",
				"html_url":         "https://github.com/wimpysworld/deb-get",
				"default_branch":   "main",
				"language":         "Shell",
				"stargazers_count": 4200,
			})
		case "/repos/wimpysworld/deb-get/releases/latest":
			writeTestJSON(t, w, map[string]any{
				"tag_name":     "v0.4.5",
				"html_url":     "https://github.com/wimpysworld/deb-get/releases/tag/v0.4.5",
				"published_at": fixedNow.Add(-48 * time.Hour),
			})
		case "/wimpysworld/deb-get/main/pages/icon.svg":
			_, _ = w.Write([]byte(`<svg viewBox="0 0 32 32"><rect width="32" height="32"/></svg>`))
		case "/repos/wimpysworld/extra-tool":
			writeTestJSON(t, w, map[string]any{
				"full_name":        "wimpysworld/extra-tool",
				"name":             "extra-tool",
				"owner":            map[string]string{"login": "wimpysworld"},
				"description":      "Standalone Nix utility",
				"html_url":         "https://github.com/wimpysworld/extra-tool",
				"default_branch":   "main",
				"language":         "Nix",
				"stargazers_count": 210,
			})
		case "/users/exampleuser":
			writeTestJSON(t, w, map[string]any{"login": "exampleuser", "followers": 77, "public_repos": 15})
		case "/users/exampleuser/repos":
			writeTestJSON(t, w, []map[string]any{
				{
					"full_name":        "exampleuser/site",
					"name":             "site",
					"owner":            map[string]string{"login": "exampleuser"},
					"html_url":         "https://github.com/exampleuser/site",
					"stargazers_count": 300,
					"pushed_at":        fixedNow.Add(-24 * time.Hour),
				},
			})
		case "/search/commits":
			writeTestJSON(t, w, map[string]any{"total_count": 1500})
		case "/search/issues":
			writeTestJSON(t, w, map[string]any{"total_count": 12, "items": []any{}})
		case "/users/exampleuser/events/public", "/users/exampleuser/starred":
			writeTestJSON(t, w, []any{})
		case "/sponsors/exampleuser/sponsors_partial":
			if r.URL.Query().Get("filter") == "inactive" {
				_, _ = w.Write([]byte(`<html><img alt="@bob"></html>`))
				return
			}
			_, _ = w.Write([]byte(`<html><img alt="@alice"></html>`))
		case "/discord/invites/testcode":
			writeTestJSON(t, w, map[string]any{
				"guild":                      map[string]string{"name": "Wimpy's World Discord"},
				"approximate_presence_count": 195,
				"approximate_member_count":   1600,
			})
		case "/podcast.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel><item><title>Episode 94</title><link>https://linuxmatters.sh/episode/94</link><pubDate>Mon, 23 Nov 2026 08:00:00 +0000</pubDate><itunes:episode>94</itunes:episode><itunes:duration>38:00</itunes:duration><itunes:subtitle>Linux Matters episode summary.</itunes:subtitle></item></channel></rss>`))
		case "/blog.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><item><title>Portfolio Launch</title><link>https://wimpysworld.com/posts/launch</link><pubDate>Mon, 23 Nov 2026 10:00:00 +0000</pubDate><description>Launching wimpys.world.</description></item></channel></rss>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "vitrine.yaml")
	cfgYAML := `profile:
  name: "Martin Wimpress"
  handle: "exampleuser"
  hero_title: "WIMPY'S WORLD"
  hero_subtitle: "Linux enthusiast. Nix wrangler."
  bio: "Open-source developer at [Oval Tutu](https://oval-tutu.com)."
discord:
  enabled: true
  name: "Wimpy's World Discord"
  url: "https://wimpysworld.io/discord"
  invite_code: "testcode"
  guild_id: "123456"
  invite_base_url: "` + srv.URL + `/discord/invites"
  widget_base_url: "` + srv.URL + `/discord/guilds"
  shields_base_url: "` + srv.URL + `/discord/shields"
github:
  username: "exampleuser"
  sponsors_url: "https://github.com/sponsors/exampleuser"
haberdasher:
  enabled: true
  config_url: "` + srv.URL + `/haberdasher.yaml"
  sort: stars
  hosts:
    - "apt.wimpys.world"
projects:
  - repo: "wimpysworld/extra-tool"
feeds:
  podcast:
    enabled: true
    title: "Linux Matters Podcast"
    url: "` + srv.URL + `/podcast.xml"
    site_url: "https://linuxmatters.sh"
    max_items: 5
    links:
      - label: "RSS"
        url: "https://linuxmatters.sh/episode/index.xml"
        icon: "rss"
  blog:
    enabled: true
    title: "Blog"
    url: "` + srv.URL + `/blog.xml"
    site_url: "https://wimpysworld.com"
    max_items: 5
site:
  title: "Wimpy's World"
  description: "Linux enthusiast. Nix wrangler."
  domain: "wimpys.world"
  url: "https://wimpys.world"
  default_deck: "projects"
  links:
    - title: "Wimpy's World Discord"
      slug: "discord"
      category: "Community"
      description: "Friendly Linux hub."
      url: "https://wimpysworld.io/discord"
      icon: "discord"
    - title: "GitHub Sponsors"
      slug: "github-sponsors"
      category: "Support"
      description: "Sponsor ongoing work."
      url: "https://github.com/sponsors/exampleuser"
      icon: "heart"
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "dist")
	ghClient := &gh.Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		RawBaseURL: srv.URL,
		WebBaseURL: srv.URL,
	}

	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{
		"-verbose",
		"-config", cfgPath,
		"-out", outDir,
	}, runtimeDeps{
		now:          func() time.Time { return fixedNow },
		httpClient:   srv.Client(),
		githubClient: ghClient,
		socialFetcher: &site.SocialFetcher{
			HTTPClient:        srv.Client(),
			BlueskyAPIBaseURL: srv.URL,
			DecapiBaseURL:     srv.URL,
		},
		haberdasherHostBaseFn: func(host string) string {
			return srv.URL + "/hosts/" + host
		},
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("run failed: %v\nstderr:\n%s", err, stderr.String())
	}

	dataBytes, err := os.ReadFile(filepath.Join(outDir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}

	var got site.SiteData
	if err := json.Unmarshal(dataBytes, &got); err != nil {
		t.Fatalf("unmarshal data.json: %v", err)
	}

	if len(got.Projects) != 2 {
		t.Fatalf("len(Projects) = %d, want 2", len(got.Projects))
	}
	if got.Projects[0].Name != "Deb Get" || got.Projects[0].Slug != "deb-get" || got.Projects[1].Name != "Extra Tool" || got.Projects[1].Slug != "extra-tool" {
		t.Errorf("unexpected project order or display names: %+v", got.Projects)
	}
	if len(got.Packages) != 1 || got.Packages[0].Host != "apt.wimpys.world" {
		t.Errorf("unexpected packages: %+v", got.Packages)
	}
	if len(got.Links) != 2 || got.Links[0].Badge != "195 online" || got.Links[1].Badge != "1 active" {
		t.Errorf("unexpected links enrichment: %+v", got.Links)
	}
	if len(got.Media) != 2 {
		t.Errorf("len(Media) = %d, want 2", len(got.Media))
	}
	for _, rel := range []string{"vendor/pixi.min.mjs", "vendor/pixi.LICENSE"} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("expected vendored asset %q: %v", rel, err)
		}
	}
	if !strings.Contains(stderr.String(), "site: done in") {
		t.Errorf("expected verbose completion log in stderr, got: %s", stderr.String())
	}
}

func TestRunErrorsOnMissingConfig(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := run(context.Background(), []string{"-config", filepath.Join(t.TempDir(), "missing.yaml")}, runtimeDeps{
		stderr: &stderr,
	})
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestRunGitHubOutageFallsBackToCachedData(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)

	cached := site.SiteData{
		DefaultSort: "stars",
		HostLogos:   map[string]string{"apt.wimpys.world": "<svg>apt</svg>"},
		Stats: site.StatsData{
			TotalStars:     5000,
			StarsFormatted: "5,000",
			Followers:      800,
			ActiveSponsors: 5,
			DiscordOnline:  150,
			DiscordMembers: 1400,
		},
		Projects: []site.ProjectCard{
			{
				ID:             "deb-get",
				FullName:       "wimpysworld/deb-get",
				Name:           "Deb Get",
				Slug:           "deb-get",
				Description:    "Cached deb-get description",
				Language:       "Shell",
				Stars:          4200,
				StarsFormatted: "4,200",
				StarsCompact:   "4.2k",
				LatestTag:      "v0.4.5",
			},
			{
				ID:             "extra-tool",
				FullName:       "wimpysworld/extra-tool",
				Name:           "Extra Tool",
				Slug:           "extra-tool",
				Description:    "Cached extra-tool description",
				Language:       "Nix",
				Stars:          210,
				StarsFormatted: "210",
				StarsCompact:   "210",
			},
		},
		Packages: []site.PackageHostCard{
			{ID: "apt", Name: "apt", Host: "apt.wimpys.world", PackageCount: 4},
		},
		Links: []site.LinkCard{
			{
				ID:       "github",
				Badge:    "5 active",
				Stats:    []site.StatRow{{Label: "STARS", Value: "5,000"}},
				Sponsors: []site.SponsorRef{{Login: "alice", URL: "https://github.com/alice"}},
			},
		},
	}

	// Simulate total GitHub outage (503) while blog RSS and fallback data.json succeed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fallback/data.json":
			writeTestJSON(t, w, cached)
		case "/blog.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0"?><rss version="2.0"><channel><item><title>Fresh Blog Post</title><link>https://blog.wimpys.world/posts/fresh/</link><pubDate>Wed, 25 Nov 2026 09:00:00 +0000</pubDate><description>Live post during GitHub outage.</description></item></channel></rss>`))
		default:
			http.Error(w, "GitHub unavailable", http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "vitrine.yaml")
	cfgYAML := `profile:
  name: "Martin Wimpress"
  handle: "exampleuser"
github:
  username: "exampleuser"
  sponsors_url: "https://github.com/sponsors/exampleuser"
haberdasher:
  enabled: true
  config_url: "` + srv.URL + `/haberdasher.yaml"
  sort: stars
  hosts:
    - "apt.wimpys.world"
projects:
  - repo: "wimpysworld/extra-tool"
feeds:
  blog:
    enabled: true
    title: "Blog"
    url: "` + srv.URL + `/blog.xml"
    site_url: "https://blog.wimpys.world"
    max_items: 5
site:
  title: "Wimpy's World"
  description: "Updated offline tagline"
  domain: "wimpys.world"
  url: "https://wimpys.world"
  default_deck: "projects"
  links:
    - title: "GitHub"
      slug: "github"
      category: "Code"
      description: "Open-source repositories."
      url: "https://github.com/exampleuser"
      icon: "github"
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	outDir := filepath.Join(dir, "dist")
	cacheFile := filepath.Join(dir, "cache", "site-data.json")
	ghClient := &gh.Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		RawBaseURL: srv.URL,
		WebBaseURL: srv.URL,
	}

	var stderr bytes.Buffer
	err := run(context.Background(), []string{
		"-config", cfgPath,
		"-out", outDir,
		"-fallback-data", srv.URL + "/fallback/data.json",
		"-cache", cacheFile,
	}, runtimeDeps{
		now:          func() time.Time { return fixedNow },
		httpClient:   srv.Client(),
		githubClient: ghClient,
		haberdasherHostBaseFn: func(host string) string {
			return srv.URL + "/hosts/" + host
		},
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("run during GitHub outage failed: %v\nstderr:\n%s", err, stderr.String())
	}

	dataBytes, err := os.ReadFile(filepath.Join(outDir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}
	var got site.SiteData
	if err := json.Unmarshal(dataBytes, &got); err != nil {
		t.Fatalf("unmarshal data.json: %v", err)
	}

	if got.Profile.Subtitle != "Updated offline tagline" {
		t.Errorf("Subtitle = %q, want Updated offline tagline", got.Profile.Subtitle)
	}
	if len(got.Projects) != 2 || got.Projects[0].Stars != 4200 || got.Projects[1].Stars != 210 {
		t.Errorf("expected cached projects restored and sorted, got: %+v", got.Projects)
	}
	if len(got.Packages) != 1 || got.Stats.TotalStars != 5000 {
		t.Errorf("expected cached packages and stats restored: packages=%d stats=%+v", len(got.Packages), got.Stats)
	}
	if len(got.Links) != 1 || len(got.Links[0].Sponsors) != 1 || got.Links[0].Badge != "5 active" {
		t.Errorf("expected cached GitHub link sponsors/badge restored: %+v", got.Links)
	}
	if len(got.Media) != 1 || got.Media[0].Title != "Fresh Blog Post" {
		t.Errorf("expected fresh live blog post alongside cached GitHub data: %+v", got.Media)
	}
	if _, err := os.Stat(cacheFile); err != nil {
		t.Errorf("expected cache file written at %q: %v", cacheFile, err)
	}
	if !strings.Contains(stderr.String(), "warning: applied fallback data") {
		t.Errorf("expected fallback warning in stderr, got: %s", stderr.String())
	}
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode test JSON: %v", err)
	}
}
