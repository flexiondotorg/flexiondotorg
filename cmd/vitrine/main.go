// Command vitrine generates animated SVG tiles and a dynamic GitHub profile
// README from configuration and live public data sources.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	"github.com/flexiondotorg/flexiondotorg/internal/discord"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/haberdasher"
	"github.com/flexiondotorg/flexiondotorg/internal/render"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
	"github.com/flexiondotorg/flexiondotorg/internal/svg"
	"github.com/flexiondotorg/flexiondotorg/internal/twitch"
)

var version = "dev"

const (
	defaultRunTimeout  = 2 * time.Minute
	defaultHTTPTimeout = 30 * time.Second
	freshReleaseWindow = 14 * 24 * time.Hour
)

type cliOptions struct {
	showVersion  bool
	configPath   string
	templatePath string
	outPath      string
	assetsDir    string
	assetBaseURL string
	previewHTML  string
}

type runtimeDeps struct {
	now          func() time.Time
	httpClient   *http.Client
	githubClient *gh.Client
	stdout       io.Writer
	stderr       io.Writer
}

type svgRenderers struct {
	light *svg.Renderer
	dark  *svg.Renderer
}

func main() {
	os.Exit(runMain())
}

func runMain() int {
	deps := runtimeDeps{
		now:        func() time.Time { return time.Now().UTC() },
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRunTimeout)
	defer cancel()

	if err := run(ctx, os.Args[1:], deps); err != nil {
		fmt.Fprintf(os.Stderr, "vitrine: %v\n", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string, deps runtimeDeps) error {
	if deps.stdout == nil {
		deps.stdout = io.Discard
	}
	if deps.stderr == nil {
		deps.stderr = io.Discard
	}
	if deps.now == nil {
		deps.now = func() time.Time { return time.Now().UTC() }
	}
	if deps.httpClient == nil {
		deps.httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	opts, err := parseFlags(args, deps.stderr)
	if err != nil {
		return err
	}
	if opts.showVersion {
		_, err := fmt.Fprintln(deps.stdout, version)
		return err
	}

	cfg, err := config.Load(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	darkPalette, err := cfg.ResolveDarkPalette()
	if err != nil {
		return fmt.Errorf("resolve dark palette: %w", err)
	}
	lightPalette, err := cfg.ResolveLightPalette()
	if err != nil {
		return fmt.Errorf("resolve light palette: %w", err)
	}
	darkSVG := svg.NewRenderer(toSVGPalette(darkPalette))
	lightSVG := svg.NewRenderer(toSVGPalette(lightPalette))
	renderers := svgRenderers{
		light: lightSVG,
		dark:  darkSVG,
	}

	ghClient := deps.githubClient
	if ghClient == nil {
		ghClient = gh.NewClient(os.Getenv("GITHUB_TOKEN"))
		ghClient.HTTPClient = deps.httpClient
	}

	now := deps.now().UTC()
	data := render.RenderData{
		Config:       cfg,
		GeneratedAt:  now,
		AssetBaseURL: resolveAssetBaseURL(opts.assetBaseURL, cfg.Output.AssetBaseURL),
		SectionBases: make(map[string]string),
		SectionFiles: make(map[string]string),
	}

	catalogue := fetchHaberdasher(ctx, cfg, deps.httpClient, ghClient, deps.stderr)
	extraRepos := projectFullNames(catalogue)
	data.GitHub = fetchGitHubProfile(ctx, cfg, ghClient, extraRepos, deps.stderr)
	discordStatus := fetchDiscordStatus(ctx, deps.httpClient, cfg.Discord, deps.stderr)
	data.Podcast = fetchFeedSection(ctx, deps.httpClient, cfg.Feeds.Podcast, "podcast", deps.stderr)
	data.Blog = fetchFeedSection(ctx, deps.httpClient, cfg.Feeds.Blog, "blog", deps.stderr)
	data.YouTube = fetchFeedSection(ctx, deps.httpClient, cfg.Feeds.YouTube, "youtube", deps.stderr)
	twitchStatus := fetchTwitchStatus(ctx, deps.httpClient, cfg.Twitch, deps.stderr)

	if err := os.MkdirAll(filepath.Clean(opts.assetsDir), 0o750); err != nil {
		return fmt.Errorf("create assets directory %q: %w", opts.assetsDir, err)
	}

	if err := generateSVGs(renderers, opts.assetsDir, cfg, catalogue, discordStatus, twitchStatus, opts.previewHTML != "", &data); err != nil {
		return err
	}

	rendered, err := render.File(opts.templatePath, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	if err := writeOutputFile(opts.outPath, rendered); err != nil {
		return fmt.Errorf("write output %q: %w", opts.outPath, err)
	}

	if strings.TrimSpace(opts.previewHTML) != "" {
		previewBytes := buildPreviewHTML(string(rendered), data, "twitch-live-demo", "twitch-offline-demo")
		if err := writeOutputFile(opts.previewHTML, previewBytes); err != nil {
			return fmt.Errorf("write preview HTML %q: %w", opts.previewHTML, err)
		}
	}

	return nil
}

func parseFlags(args []string, stderr io.Writer) (cliOptions, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("vitrine", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.BoolVar(&opts.showVersion, "version", false, "print version and exit")
	fs.StringVar(&opts.configPath, "config", "vitrine.yaml", "path to vitrine YAML configuration")
	fs.StringVar(&opts.templatePath, "template", "templates/README.md.tpl", "path to README template")
	fs.StringVar(&opts.outPath, "out", "README.md", "path to rendered Markdown output")
	fs.StringVar(&opts.assetsDir, "assets-dir", "assets/generated", "directory for generated SVG tiles")
	fs.StringVar(&opts.assetBaseURL, "asset-base-url", "", "optional base URL or path prefix for SVG assets")
	fs.StringVar(&opts.previewHTML, "preview-html", "", "optional path for local GitHub Dark/Light HTML preview")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}
	return opts, nil
}

func resolveAssetBaseURL(flagVal, cfgVal string) string {
	if trimmed := strings.TrimSpace(flagVal); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(cfgVal)
}

func toSVGPalette(p config.Palette) svg.Palette {
	return svg.Palette{
		Base:            p.Base,
		Mantle:          p.Mantle,
		Surface0:        p.Surface0,
		Surface1:        p.Surface1,
		Text:            p.Text,
		Subtext:         p.Subtext,
		Muted:           p.Muted,
		AccentPrimary:   p.AccentPrimary,
		AccentSecondary: p.AccentSecondary,
		AccentTertiary:  p.AccentTertiary,
		AccentPink:      p.AccentPink,
		AccentGreen:     p.AccentGreen,
		AccentRed:       p.AccentRed,
		AccentPeach:     p.AccentPeach,
		AccentYellow:    p.AccentYellow,
		AccentSapphire:  p.AccentSapphire,
		IsLight:         p.IsLight,
	}
}

func fetchHaberdasher(
	ctx context.Context,
	cfg *config.Config,
	httpClient *http.Client,
	ghClient *gh.Client,
	stderr io.Writer,
) *haberdasher.Catalogue {
	if !cfg.Haberdasher.Enabled {
		return nil
	}
	fetcher := haberdasher.NewFetcher(httpClient, ghClient)
	cat, err := fetcher.Fetch(ctx, haberdasher.Options{
		ConfigURL:     cfg.Haberdasher.ConfigURL,
		Hosts:         cfg.Haberdasher.Hosts,
		Sort:          cfg.Haberdasher.Sort,
		LogoOverrides: cfg.Haberdasher.LogoOverrides,
		PNGFallbacks:  cfg.Haberdasher.PNGFallbacks,
	})
	if err != nil {
		fmt.Fprintf(stderr, "warning: haberdasher fetch failed: %v\n", err)
		return nil
	}
	return cat
}

func projectFullNames(cat *haberdasher.Catalogue) []string {
	if cat == nil || len(cat.Projects) == 0 {
		return nil
	}
	repos := make([]string, 0, len(cat.Projects))
	for _, p := range cat.Projects {
		if strings.TrimSpace(p.FullName) != "" {
			repos = append(repos, p.FullName)
		}
	}
	return repos
}

func fetchGitHubProfile(
	ctx context.Context,
	cfg *config.Config,
	ghClient *gh.Client,
	extraReleaseRepos []string,
	stderr io.Writer,
) *gh.ProfileData {
	if strings.TrimSpace(cfg.GitHub.Username) == "" {
		return nil
	}

	fallback := make([]gh.Sponsor, 0, len(cfg.GitHub.SponsorsFallback))
	for _, s := range cfg.GitHub.SponsorsFallback {
		fallback = append(fallback, gh.Sponsor{Login: s.Login, URL: s.URL})
	}

	profile, err := ghClient.FetchProfile(ctx, cfg.GitHub.Username, gh.ProfileOptions{
		Orgs:                    cfg.GitHub.Orgs,
		ExcludeOwners:           cfg.GitHub.ExcludeOwners,
		MaxItems:                cfg.GitHub.MaxItems,
		ContributionsWindowDays: cfg.GitHub.ContributionsWindowDays,
		FallbackSponsors:        fallback,
		ExtraReleaseRepos:       extraReleaseRepos,
	})
	if err != nil {
		fmt.Fprintf(stderr, "warning: github profile fetch failed: %v\n", err)
		return &gh.ProfileData{
			Stats:        gh.Stats{Username: cfg.GitHub.Username},
			Sponsors:     fallback,
			SponsorCount: len(fallback),
		}
	}
	return profile
}

func fetchDiscordStatus(
	ctx context.Context,
	httpClient *http.Client,
	cfg config.DiscordConfig,
	stderr io.Writer,
) discord.Status {
	fallback := discord.Status{
		Name: cfg.Name,
		URL:  cfg.URL,
	}
	if !cfg.Enabled {
		return fallback
	}

	client := &discord.Client{
		HTTPClient:     httpClient,
		InviteBaseURL:  cfg.InviteBaseURL,
		WidgetBaseURL:  cfg.WidgetBaseURL,
		ShieldsBaseURL: cfg.ShieldsBaseURL,
	}
	status, err := client.Fetch(ctx, cfg.InviteCode, cfg.GuildID, cfg.URL)
	if err != nil {
		fmt.Fprintf(stderr, "warning: discord status fetch failed: %v\n", err)
		return fallback
	}
	if strings.TrimSpace(cfg.Name) != "" {
		status.Name = cfg.Name
	}
	if status.URL == "" {
		status.URL = cfg.URL
	}
	return status
}

func fetchFeedSection(
	ctx context.Context,
	httpClient *http.Client,
	feedCfg *config.FeedConfig,
	label string,
	stderr io.Writer,
) *render.FeedView {
	if feedCfg == nil {
		return nil
	}
	view := &render.FeedView{
		Enabled:  feedCfg.Enabled,
		Title:    feedCfg.Title,
		URL:      feedCfg.URL,
		SiteURL:  feedCfg.SiteURL,
		CoverURL: feedCfg.CoverURL,
	}
	if !feedCfg.Enabled {
		return view
	}

	items, err := rss.Fetch(ctx, httpClient, feedCfg.URL, feedCfg.MaxItems)
	if err != nil {
		fmt.Fprintf(stderr, "warning: %s feed fetch failed: %v\n", label, err)
		return view
	}
	view.Items = items
	return view
}

func fetchTwitchStatus(
	ctx context.Context,
	httpClient *http.Client,
	cfg config.TwitchConfig,
	stderr io.Writer,
) twitch.Status {
	fallback := twitch.Status{
		Channel: cfg.Channel,
		Live:    false,
		URL:     cfg.URL,
	}
	if !cfg.Enabled {
		return fallback
	}

	client := &twitch.Client{
		HTTPClient:      httpClient,
		PrimaryBaseURL:  cfg.PrimaryBaseURL,
		FallbackBaseURL: cfg.FallbackBaseURL,
	}
	status, err := client.Fetch(ctx, cfg.Channel)
	if err != nil {
		fmt.Fprintf(stderr, "warning: twitch status fetch failed: %v\n", err)
		return fallback
	}
	if status.URL == "" {
		status.URL = cfg.URL
	}
	return status
}

func writeDualSVG(
	sr svgRenderers,
	writeAsset func(string, []byte) error,
	baseName string,
	renderFn func(*svg.Renderer) ([]byte, error),
) (lightFile, darkFile string, err error) {
	lightBytes, err := renderFn(sr.light)
	if err != nil {
		return "", "", fmt.Errorf("render %s (light): %w", baseName, err)
	}
	lightFile = baseName + "-light.svg"
	if err := writeAsset(lightFile, lightBytes); err != nil {
		return "", "", err
	}

	darkBytes, err := renderFn(sr.dark)
	if err != nil {
		return "", "", fmt.Errorf("render %s (dark): %w", baseName, err)
	}
	darkFile = baseName + "-dark.svg"
	if err := writeAsset(darkFile, darkBytes); err != nil {
		return "", "", err
	}

	return lightFile, darkFile, nil
}

func generateSVGs(
	sr svgRenderers,
	assetsDir string,
	cfg *config.Config,
	cat *haberdasher.Catalogue,
	discordStatus discord.Status,
	twitchStatus twitch.Status,
	includePreviewTiles bool,
	data *render.RenderData,
) error {
	root, err := os.OpenRoot(filepath.Clean(assetsDir))
	if err != nil {
		return fmt.Errorf("open assets directory %q: %w", assetsDir, err)
	}
	defer func() { _ = root.Close() }()

	writeAsset := func(filename string, content []byte) error {
		return writeRootFile(root, filename, content)
	}

	heroLight, _, err := writeDualSVG(sr, writeAsset, "hero", func(r *svg.Renderer) ([]byte, error) {
		return r.RenderHero(svg.HeroData{
			Eyebrow:  cfg.Profile.Eyebrow,
			Title:    cfg.Profile.HeroTitle,
			Subtitle: cfg.Profile.HeroSubtitle,
			CTA:      cfg.Profile.HeroCTA,
		})
	})
	if err != nil {
		return err
	}
	data.HeroBase = "hero"
	data.HeroFile = heroLight

	if err := renderNavPills(sr, cfg.Profile.Links, writeAsset, data); err != nil {
		return err
	}
	if err := renderDiscordBanner(sr, cfg.Discord, discordStatus, writeAsset, data); err != nil {
		return err
	}
	if err := renderCatalogueTiles(sr, cat, data.GeneratedAt, writeAsset, data); err != nil {
		return err
	}
	if err := renderStatsAndDividers(sr, cfg, writeAsset, data); err != nil {
		return err
	}
	return renderTwitchTiles(sr, cfg.Twitch, twitchStatus, includePreviewTiles, writeAsset, data)
}

func renderNavPills(
	sr svgRenderers,
	links []config.NavLink,
	writeAsset func(string, []byte) error,
	data *render.RenderData,
) error {
	seen := make(map[string]int, len(links))
	for i, link := range links {
		slug := slugify(link.Slug)
		if slug == "" {
			slug = slugify(link.Label)
		}
		if slug == "" {
			slug = fmt.Sprintf("link-%d", i+1)
		}
		seen[slug]++
		if seen[slug] > 1 {
			slug = fmt.Sprintf("%s-%d", slug, seen[slug])
		}

		baseName := "nav-" + slug
		lightFile, darkFile, err := writeDualSVG(sr, writeAsset, baseName, func(r *svg.Renderer) ([]byte, error) {
			return r.RenderNavPill(svg.NavPillData{
				Label:        link.Label,
				Icon:         link.Icon,
				DelaySeconds: float64(i) * 0.45,
			})
		})
		if err != nil {
			return err
		}
		data.NavPills = append(data.NavPills, render.NavPillView{
			Label:     link.Label,
			Slug:      slug,
			URL:       link.URL,
			Icon:      link.Icon,
			BaseName:  baseName,
			LightFile: lightFile,
			DarkFile:  darkFile,
			File:      lightFile,
		})
	}
	return nil
}

func renderDiscordBanner(
	sr svgRenderers,
	cfg config.DiscordConfig,
	status discord.Status,
	writeAsset func(string, []byte) error,
	data *render.RenderData,
) error {
	if !cfg.Enabled {
		return nil
	}

	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = strings.TrimSpace(status.Name)
	}
	if name == "" {
		name = "Discord"
	}
	joinURL := strings.TrimSpace(cfg.URL)
	if joinURL == "" {
		joinURL = status.URL
	}

	lightFile, darkFile, err := writeDualSVG(sr, writeAsset, "discord", func(r *svg.Renderer) ([]byte, error) {
		return r.RenderDiscordBanner(svg.DiscordData{
			Name:    name,
			Tagline: cfg.Tagline,
			Online:  status.Online,
			Members: status.Members,
		})
	})
	if err != nil {
		return err
	}

	data.Discord = &render.DiscordView{
		Enabled:   true,
		Name:      name,
		Tagline:   cfg.Tagline,
		URL:       joinURL,
		Online:    status.Online,
		Members:   status.Members,
		BaseName:  "discord",
		LightFile: lightFile,
		DarkFile:  darkFile,
		File:      lightFile,
	}
	return nil
}

func renderCatalogueTiles(
	sr svgRenderers,
	cat *haberdasher.Catalogue,
	now time.Time,
	writeAsset func(string, []byte) error,
	data *render.RenderData,
) error {
	if cat == nil {
		return nil
	}

	hostTotal := len(cat.Hosts)
	if hostTotal == 0 {
		hostTotal = 8
	}

	for i, proj := range cat.Projects {
		slug := slugify(proj.Slug)
		if slug == "" {
			slug = slugify(proj.Name)
		}
		if slug == "" {
			slug = fmt.Sprintf("project-%d", i+1)
		}
		baseName := "project-" + slug
		fresh := !proj.PublishedAt.IsZero() && now.Sub(proj.PublishedAt) <= freshReleaseWindow && now.Sub(proj.PublishedAt) >= 0

		lightFile, darkFile, err := writeDualSVG(sr, writeAsset, baseName, func(r *svg.Renderer) ([]byte, error) {
			return r.RenderProjectCard(svg.ProjectCardData{
				Slug:         slug,
				Name:         proj.Name,
				Tagline:      proj.Description,
				Version:      proj.LatestTag,
				Fresh:        fresh,
				Language:     proj.Language,
				Stars:        proj.Stars,
				ReleaseAge:   render.FormatCompactAge(proj.PublishedAt, now),
				PackageCount: len(proj.Hosts),
				PackageTotal: hostTotal,
				LogoSVG:      proj.LogoSVG,
				LogoDataURI:  proj.LogoDataURI,
				DelaySeconds: float64(i) * 0.55,
			})
		})
		if err != nil {
			return err
		}

		targetURL := proj.Homepage
		if targetURL == "" {
			targetURL = proj.RepoURL
		}
		data.Projects = append(data.Projects, render.ProjectView{
			FullName:    proj.FullName,
			Name:        proj.Name,
			Slug:        slug,
			Description: proj.Description,
			Homepage:    proj.Homepage,
			RepoURL:     proj.RepoURL,
			URL:         targetURL,
			Language:    proj.Language,
			Stars:       proj.Stars,
			LatestTag:   proj.LatestTag,
			ReleaseURL:  proj.ReleaseURL,
			PublishedAt: proj.PublishedAt,
			Hosts:       proj.Hosts,
			BaseName:    baseName,
			LightFile:   lightFile,
			DarkFile:    darkFile,
			File:        lightFile,
		})
	}

	for i, host := range cat.Hosts {
		slug := slugify(host.Name)
		if slug == "" {
			slug = fmt.Sprintf("host-%d", i+1)
		}
		baseName := "host-" + slug
		lightFile, darkFile, err := writeDualSVG(sr, writeAsset, baseName, func(r *svg.Renderer) ([]byte, error) {
			return r.RenderPackageHost(svg.PackageHostData{
				Name:         host.Name,
				Host:         host.Host,
				PackageCount: host.PackageCount,
				LogoSVG:      host.LogoSVG,
				DelaySeconds: float64(i) * 0.45,
			})
		})
		if err != nil {
			return err
		}
		data.PackageHosts = append(data.PackageHosts, render.PackageHostView{
			Name:         host.Name,
			Host:         host.Host,
			URL:          host.URL,
			PackageCount: host.PackageCount,
			BaseName:     baseName,
			LightFile:    lightFile,
			DarkFile:     darkFile,
			File:         lightFile,
		})
	}

	return nil
}

func renderStatsAndDividers(
	sr svgRenderers,
	cfg *config.Config,
	writeAsset func(string, []byte) error,
	data *render.RenderData,
) error {
	if data.GitHub != nil {
		s := data.GitHub.Stats
		lightFile, _, err := writeDualSVG(sr, writeAsset, "stats", func(r *svg.Renderer) ([]byte, error) {
			return r.RenderStatsStrip(svg.StatsData{
				Metrics: []svg.StatMetric{
					{Label: "Stars", Value: render.FormatInt(s.TotalStars)},
					{Label: "Commits", Value: render.FormatInt(s.TotalCommits)},
					{Label: "Pull Requests", Value: render.FormatInt(s.PublicPRs)},
					{Label: "Issues", Value: render.FormatInt(s.PublicIssues)},
					{Label: "Followers", Value: render.FormatInt(s.Followers)},
				},
			})
		})
		if err != nil {
			return err
		}
		data.StatsBase = "stats"
		data.StatsFile = lightFile
	}

	youtubeEnabled := cfg.Feeds.YouTube != nil && cfg.Feeds.YouTube.Enabled
	videosTitle := videosSectionTitle(youtubeEnabled, cfg.Twitch.Enabled)
	data.VideosHeading = videosTitle

	sections := []struct {
		key      string
		needed   bool
		title    string
		subtitle string
	}{
		{
			key:      "projects",
			needed:   len(data.Projects) > 0,
			title:    "PROJECTS",
			subtitle: fmt.Sprintf("%d FEATURED", len(data.Projects)),
		},
		{
			key:      "packages",
			needed:   len(data.PackageHosts) > 0,
			title:    "PACKAGE REPOSITORIES",
			subtitle: fmt.Sprintf("%d FORMATS", len(data.PackageHosts)),
		},
		{
			key:    "contributions",
			needed: data.GitHub != nil && len(data.GitHub.Contributions) > 0,
			title:  "COMMUNITY CONTRIBUTIONS",
		},
		{
			key:    "releases",
			needed: data.GitHub != nil && len(data.GitHub.Releases) > 0,
			title:  "RECENT RELEASES",
		},
		{
			key:    "starred",
			needed: data.GitHub != nil && len(data.GitHub.Starred) > 0,
			title:  "STARRED",
		},
		{
			key:      "podcast",
			needed:   data.Podcast != nil && data.Podcast.Enabled,
			title:    "PODCAST",
			subtitle: "EPISODES",
		},
		{
			key:      "blog",
			needed:   data.Blog != nil && data.Blog.Enabled,
			title:    "BLOG",
			subtitle: "POSTS",
		},
		{
			key:    "videos",
			needed: videosTitle != "",
			title:  videosTitle,
		},
	}

	for _, sec := range sections {
		if !sec.needed {
			continue
		}
		baseName := "section-" + sec.key
		lightFile, _, err := writeDualSVG(sr, writeAsset, baseName, func(r *svg.Renderer) ([]byte, error) {
			return r.RenderSectionDivider(svg.SectionData{
				Title:    sec.title,
				Subtitle: sec.subtitle,
			})
		})
		if err != nil {
			return err
		}
		data.SectionBases[sec.key] = baseName
		data.SectionFiles[sec.key] = lightFile
	}

	return nil
}

func videosSectionTitle(youtubeEnabled, twitchEnabled bool) string {
	switch {
	case youtubeEnabled && twitchEnabled:
		return "VIDEOS & STREAMS"
	case youtubeEnabled:
		return "VIDEOS"
	case twitchEnabled:
		return "STREAMS"
	default:
		return ""
	}
}

func renderTwitchTiles(
	sr svgRenderers,
	cfg config.TwitchConfig,
	status twitch.Status,
	includePreviewTiles bool,
	writeAsset func(string, []byte) error,
	data *render.RenderData,
) error {
	if cfg.Enabled {
		lightFile, darkFile, err := writeDualSVG(sr, writeAsset, "twitch-status", func(r *svg.Renderer) ([]byte, error) {
			return r.RenderTwitchStatus(svg.TwitchData{
				Live:    status.Live,
				Uptime:  status.Uptime,
				Channel: status.Channel,
			})
		})
		if err != nil {
			return err
		}
		alt := "Twitch offline"
		if status.Live {
			alt = "Live on Twitch"
		}
		data.Twitch = &render.TwitchView{
			Enabled:   true,
			Channel:   status.Channel,
			Live:      status.Live,
			Uptime:    status.Uptime,
			URL:       status.URL,
			Alt:       alt,
			BaseName:  "twitch-status",
			LightFile: lightFile,
			DarkFile:  darkFile,
			File:      lightFile,
		}
	}

	if includePreviewTiles {
		if _, _, err := writeDualSVG(sr, writeAsset, "twitch-live-demo", func(r *svg.Renderer) ([]byte, error) {
			return r.RenderTwitchStatus(svg.TwitchData{
				Live:    true,
				Uptime:  "2h 14m",
				Channel: cfg.Channel,
			})
		}); err != nil {
			return err
		}

		if _, _, err := writeDualSVG(sr, writeAsset, "twitch-offline-demo", func(r *svg.Renderer) ([]byte, error) {
			return r.RenderTwitchStatus(svg.TwitchData{
				Live:    false,
				Channel: cfg.Channel,
			})
		}); err != nil {
			return err
		}
	}

	return nil
}

func writeRootFile(root *os.Root, filename string, data []byte) error {
	f, err := root.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open asset %q: %w", filename, err)
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("write asset %q: %w", filename, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close asset %q: %w", filename, closeErr)
	}
	return nil
}

func writeOutputFile(path string, data []byte) error {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return errors.New("output path must not be empty")
	}
	dir := filepath.Dir(cleaned)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create parent directory %q: %w", dir, err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open parent directory %q: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	return writeRootFile(root, filepath.Base(cleaned), data)
}

func slugify(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
