package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
)

func TestHumanizeTime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{name: "zero", at: time.Time{}, want: ""},
		{name: "future", at: now.Add(2 * time.Hour), want: "today"},
		{name: "hours ago", at: now.Add(-6 * time.Hour), want: "today"},
		{name: "one day", at: now.Add(-24 * time.Hour), want: "1 day ago"},
		{name: "five days", at: now.Add(-5 * 24 * time.Hour), want: "5 days ago"},
		{name: "one week", at: now.Add(-7 * 24 * time.Hour), want: "1 week ago"},
		{name: "two weeks", at: now.Add(-15 * 24 * time.Hour), want: "2 weeks ago"},
		{name: "one month", at: now.Add(-35 * 24 * time.Hour), want: "1 month ago"},
		{name: "three months", at: now.Add(-95 * 24 * time.Hour), want: "3 months ago"},
		{name: "one year", at: now.Add(-400 * 24 * time.Hour), want: "1 year ago"},
		{name: "two years", at: now.Add(-800 * 24 * time.Hour), want: "2 years ago"},
	}

	for _, tc := range cases {
		if got := HumanizeTime(tc.at, now); got != tc.want {
			t.Errorf("%s: HumanizeTime() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFormatCompactAge(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		at   time.Time
		want string
	}{
		{at: time.Time{}, want: ""},
		{at: now.Add(-3 * time.Hour), want: "today"},
		{at: now.Add(-3 * 24 * time.Hour), want: "3d ago"},
		{at: now.Add(-14 * 24 * time.Hour), want: "2w ago"},
		{at: now.Add(-90 * 24 * time.Hour), want: "3mo ago"},
		{at: now.Add(-400 * 24 * time.Hour), want: "1y ago"},
	}

	for _, tc := range cases {
		if got := FormatCompactAge(tc.at, now); got != tc.want {
			t.Errorf("FormatCompactAge(%v) = %q, want %q", tc.at, got, tc.want)
		}
	}
}

func TestJoinAssetURLAndFormatInt(t *testing.T) {
	t.Parallel()

	urlCases := []struct {
		base string
		file string
		want string
	}{
		{base: "./assets", file: "hero.svg", want: "./assets/hero.svg"},
		{base: "https://raw.githubusercontent.com/o/r/abc123/", file: "/hero.svg", want: "https://raw.githubusercontent.com/o/r/abc123/hero.svg"},
		{base: "", file: "hero.svg", want: "hero.svg"},
		{base: "./assets", file: "", want: ""},
		{base: "./assets", file: "https://example.com/a.svg", want: "https://example.com/a.svg"},
	}
	for _, tc := range urlCases {
		if got := JoinAssetURL(tc.base, tc.file); got != tc.want {
			t.Errorf("JoinAssetURL(%q, %q) = %q, want %q", tc.base, tc.file, got, tc.want)
		}
	}

	intCases := []struct {
		n    int
		want string
	}{
		{n: 0, want: "0"},
		{n: 42, want: "42"},
		{n: 999, want: "999"},
		{n: 1000, want: "1,000"},
		{n: 4698, want: "4,698"},
		{n: 1234567, want: "1,234,567"},
		{n: -12345, want: "-12,345"},
	}
	for _, tc := range intCases {
		if got := FormatInt(tc.n); got != tc.want {
			t.Errorf("FormatInt(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestPictureTag(t *testing.T) {
	t.Parallel()

	gotWithWidth := PictureTag("https://example.com/assets/", "hero", `Wimpy's "World" & Friends`, 840)
	wantWithWidth := `<picture><source media="(prefers-color-scheme: dark)" srcset="https://example.com/assets/hero-dark.svg"><img src="https://example.com/assets/hero-light.svg" alt="Wimpy&#39;s &#34;World&#34; &amp; Friends" width="840"></picture>`
	if gotWithWidth != wantWithWidth {
		t.Errorf("PictureTag(width=840) =\n%s\nwant:\n%s", gotWithWidth, wantWithWidth)
	}

	gotNoWidth := PictureTag("./assets", "nav-mastodon", "Mastodon", 0)
	wantNoWidth := `<picture><source media="(prefers-color-scheme: dark)" srcset="./assets/nav-mastodon-dark.svg"><img src="./assets/nav-mastodon-light.svg" alt="Mastodon"></picture>`
	if gotNoWidth != wantNoWidth {
		t.Errorf("PictureTag(width=0) =\n%s\nwant:\n%s", gotNoWidth, wantNoWidth)
	}

	if gotEmpty := PictureTag("./assets", "", "Empty", 100); gotEmpty != "" {
		t.Errorf("PictureTag(empty) = %q, want empty", gotEmpty)
	}
}

func TestRenderTemplateHelpersAndConditionals(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	tmpl := `Hero: {{assetURL .HeroFile}}
Picture: {{pictureTag .AssetBaseURL .HeroBase "Hero <Banner>" 840}}
Explicit: {{assetURL .AssetBaseURL .StatsFile}}
Stars: {{formatInt .GitHub.Stats.TotalStars}}
Tags: {{upper (join (index .Projects 0).Hosts ", ")}} / {{lower "HELLO"}}
Age: {{humanize (index .Projects 0).PublishedAt}}
{{- if and .YouTube .YouTube.Enabled .YouTube.Items}}
YOUTUBE_SECTION
{{- end}}
{{- if and .Twitch .Twitch.Enabled .Twitch.File}}
TWITCH_SECTION:{{assetURL .Twitch.File}}
{{- end}}`

	baseData := RenderData{
		Config:       &config.Config{},
		GeneratedAt:  now,
		AssetBaseURL: "https://example.com/assets",
		HeroBase:     "hero",
		HeroFile:     "hero-light.svg",
		StatsBase:    "stats",
		StatsFile:    "stats-light.svg",
		Projects: []ProjectView{
			{
				Name:        "Sample",
				Hosts:       []string{"apt", "nix"},
				PublishedAt: now.Add(-14 * 24 * time.Hour),
			},
		},
		GitHub: &gh.ProfileData{
			Stats: gh.Stats{TotalStars: 4698},
		},
		YouTube: &FeedView{
			Enabled: false,
			Items:   []rss.Item{{Title: "Video 1", URL: "https://youtube.example/1", PublishedAt: now}},
		},
		Twitch: &TwitchView{
			Enabled:  false,
			Live:     true,
			BaseName: "twitch-status",
			File:     "twitch-status-light.svg",
		},
	}

	outDisabled, err := Template(tmpl, baseData)
	if err != nil {
		t.Fatalf("Template (disabled) error: %v", err)
	}
	textDisabled := string(outDisabled)
	for _, want := range []string{
		"Hero: https://example.com/assets/hero-light.svg",
		`Picture: <picture><source media="(prefers-color-scheme: dark)" srcset="https://example.com/assets/hero-dark.svg"><img src="https://example.com/assets/hero-light.svg" alt="Hero &lt;Banner&gt;" width="840"></picture>`,
		"Explicit: https://example.com/assets/stats-light.svg",
		"Stars: 4,698",
		"Tags: APT, NIX / hello",
		"Age: 2 weeks ago",
	} {
		if !strings.Contains(textDisabled, want) {
			t.Errorf("missing %q in output:\n%s", want, textDisabled)
		}
	}
	if strings.Contains(textDisabled, "YOUTUBE_SECTION") || strings.Contains(textDisabled, "TWITCH_SECTION") {
		t.Errorf("expected disabled YouTube and Twitch sections to be omitted:\n%s", textDisabled)
	}

	baseData.YouTube.Enabled = true
	baseData.Twitch.Enabled = true
	outEnabled, err := Template(tmpl, baseData)
	if err != nil {
		t.Fatalf("Template (enabled) error: %v", err)
	}
	textEnabled := string(outEnabled)
	if !strings.Contains(textEnabled, "YOUTUBE_SECTION") {
		t.Errorf("expected enabled YouTube section in output:\n%s", textEnabled)
	}
	if !strings.Contains(textEnabled, "TWITCH_SECTION:https://example.com/assets/twitch-status-light.svg") {
		t.Errorf("expected enabled Twitch section in output:\n%s", textEnabled)
	}
}

func TestRenderFileAndErrors(t *testing.T) {
	t.Parallel()

	if _, err := File("", RenderData{}); err == nil {
		t.Fatal("expected error for empty template path")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "README.md.tpl")
	if err := os.WriteFile(path, []byte("Hello {{.Config.Profile.Name}}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	out, err := File(path, RenderData{
		Config: &config.Config{
			Profile: config.ProfileConfig{Name: "Example"},
		},
	})
	if err != nil {
		t.Fatalf("File error: %v", err)
	}
	if string(out) != "Hello Example\n" {
		t.Fatalf("File = %q, want %q", string(out), "Hello Example\n")
	}
}

func TestRenderREADMETemplateLayoutAndVideosSection(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	tmplPath := filepath.Join("..", "..", "templates", "README.md.tpl")

	makeData := func() RenderData {
		return RenderData{
			Config: &config.Config{
				Profile: config.ProfileConfig{
					HeroTitle: "WIMPY'S WORLD",
					HeroURL:   "https://wimpys.world",
					Bio:       "Short bio line.",
				},
				GitHub: config.GitHubConfig{
					Username:    "flexiondotorg",
					SponsorsURL: "https://github.com/sponsors/flexiondotorg",
				},
			},
			GeneratedAt:  now,
			AssetBaseURL: "./assets",
			HeroBase:     "hero",
			HeroFile:     "hero-light.svg",
			NavPills: []NavPillView{
				{Label: "Mastodon", URL: "https://example.social/@dev", BaseName: "nav-mastodon"},
			},
			Discord: &DiscordView{
				Enabled:  true,
				Name:     "Wimpy's World Discord",
				URL:      "https://wimpysworld.io/discord",
				BaseName: "discord",
			},
			StatsBase: "stats",
			StatsFile: "stats-light.svg",
			Projects: []ProjectView{
				{Name: "Quickemu", URL: "https://github.com/quickemu-project/quickemu", BaseName: "project-quickemu"},
			},
			PackageHosts: []PackageHostView{
				{Name: "APT", Host: "apt.wimpys.world", URL: "https://apt.wimpys.world", BaseName: "host-apt"},
			},
			SectionBases: map[string]string{
				"projects":      "section-projects",
				"packages":      "section-packages",
				"contributions": "section-contributions",
				"releases":      "section-releases",
				"starred":       "section-starred",
				"podcast":       "section-podcast",
				"blog":          "section-blog",
				"videos":        "section-videos",
			},
			VideosHeading: "VIDEOS & STREAMS",
			GitHub: &gh.ProfileData{
				Contributions: []gh.Contribution{
					{RepoFullName: "upstream/project", RepoURL: "https://github.com/upstream/project", Description: "Upstream fix", OccurredAt: now.Add(-24 * time.Hour)},
				},
				Releases: []gh.Release{
					{RepoFullName: "wimpysworld/tool", RepoURL: "https://github.com/wimpysworld/tool", TagName: "v1.2.3", ReleaseURL: "https://github.com/wimpysworld/tool/releases/tag/v1.2.3", PublishedAt: now.Add(-48 * time.Hour)},
				},
				Starred: []gh.StarredRepo{
					{RepoFullName: "other/repo", RepoURL: "https://github.com/other/repo", Stars: 1234, Description: "Nice repo"},
				},
			},
			Podcast: &FeedView{
				Enabled:  true,
				Title:    "Linux Matters Podcast",
				SiteURL:  "https://linuxmatters.sh",
				CoverURL: "https://example.com/cover.png",
				Items: []rss.Item{
					{
						Title:       "Episode 1",
						URL:         "https://linuxmatters.sh/1",
						PublishedAt: now,
						Episode:     "94",
						Duration:    "35 min",
						Subtitle:    "Short episode subtitle.",
					},
					{
						Title:       "Episode Plain",
						URL:         "https://linuxmatters.sh/2",
						PublishedAt: now.Add(-48 * time.Hour),
					},
				},
			},
			Blog: &FeedView{
				Enabled: true,
				Title:   "Blog",
				Items: []rss.Item{
					{
						Title:           "Post 1",
						URL:             "https://wimpysworld.com/1",
						PublishedAt:     now,
						ReadTimeMinutes: 4,
						Subtitle:        "Short blog summary.",
					},
					{
						Title:       "Post Plain",
						URL:         "https://wimpysworld.com/2",
						PublishedAt: now.Add(-72 * time.Hour),
					},
				},
			},
			Twitch: &TwitchView{
				Enabled:  true,
				Channel:  "WimpysWorld",
				URL:      "https://twitch.tv/WimpysWorld",
				Alt:      "Twitch offline",
				BaseName: "twitch-status",
			},
			YouTube: &FeedView{
				Enabled: true,
				Title:   "YouTube",
				Items:   []rss.Item{{Title: "Video 1", URL: "https://youtube.com/watch?v=1", PublishedAt: now}},
			},
		}
	}

	out, err := File(tmplPath, makeData())
	if err != nil {
		t.Fatalf("File(README.md.tpl) error: %v", err)
	}
	readme := string(out)

	orderMarkers := []string{
		"./assets/hero-dark.svg",
		"./assets/nav-mastodon-dark.svg",
		"Short bio line.",
		"./assets/discord-dark.svg",
		"./assets/stats-dark.svg",
		"Sponsor on GitHub",
		"./assets/section-projects-dark.svg",
		"./assets/section-podcast-dark.svg",
		"- **#94** · **[Episode 1](https://linuxmatters.sh/1)** · `35 min` (today)",
		"  - Short episode subtitle.",
		"- **[Episode Plain](https://linuxmatters.sh/2)** (2 days ago)",
		"./assets/section-packages-dark.svg",
		"./assets/section-contributions-dark.svg",
		"./assets/section-releases-dark.svg",
		"./assets/section-starred-dark.svg",
		"./assets/section-blog-dark.svg",
		"- **[Post 1](https://wimpysworld.com/1)** · `4 min read` (today)",
		"  - Short blog summary.",
		"- **[Post Plain](https://wimpysworld.com/2)** (3 days ago)",
		`./assets/section-videos-dark.svg`,
		`alt="VIDEOS &amp; STREAMS"`,
		"[Video 1](https://youtube.com/watch?v=1)",
		"./assets/twitch-status-dark.svg",
	}
	prevIdx := -1
	for _, marker := range orderMarkers {
		idx := strings.Index(readme, marker)
		if idx < 0 {
			t.Fatalf("rendered README missing marker %q:\n%s", marker, readme)
		}
		if idx <= prevIdx {
			t.Errorf("marker %q (pos %d) must appear after previous marker (pos %d):\n%s", marker, idx, prevIdx, readme)
		}
		prevIdx = idx
	}

	t.Run("youtube only", func(t *testing.T) {
		t.Parallel()
		d := makeData()
		d.Twitch = nil
		d.VideosHeading = "VIDEOS"
		b, err := File(tmplPath, d)
		if err != nil {
			t.Fatalf("File error: %v", err)
		}
		got := string(b)
		if !strings.Contains(got, `alt="VIDEOS"`) || !strings.Contains(got, "[Video 1](https://youtube.com/watch?v=1)") {
			t.Errorf("youtube-only output missing VIDEOS divider or item:\n%s", got)
		}
		if strings.Contains(got, "twitch-status") {
			t.Errorf("youtube-only output must not contain twitch-status:\n%s", got)
		}
	})

	t.Run("twitch only", func(t *testing.T) {
		t.Parallel()
		d := makeData()
		d.YouTube.Enabled = false
		d.VideosHeading = "STREAMS"
		b, err := File(tmplPath, d)
		if err != nil {
			t.Fatalf("File error: %v", err)
		}
		got := string(b)
		if !strings.Contains(got, `alt="STREAMS"`) || !strings.Contains(got, "twitch-status-dark.svg") {
			t.Errorf("twitch-only output missing STREAMS divider or tile:\n%s", got)
		}
		if strings.Contains(got, "[Video 1]") {
			t.Errorf("twitch-only output must not contain YouTube videos:\n%s", got)
		}
	})

	t.Run("neither videos nor streams", func(t *testing.T) {
		t.Parallel()
		d := makeData()
		d.Twitch.Enabled = false
		d.YouTube.Enabled = false
		d.VideosHeading = ""
		delete(d.SectionBases, "videos")
		b, err := File(tmplPath, d)
		if err != nil {
			t.Fatalf("File error: %v", err)
		}
		got := string(b)
		if strings.Contains(got, "section-videos") || strings.Contains(got, "twitch-status") || strings.Contains(got, "[Video 1]") {
			t.Errorf("disabled videos/streams must be omitted:\n%s", got)
		}
	})
}
