// Command site builds the static portfolio web application for https://wimpys.world
// from vitrine.yaml and live public data sources.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	"github.com/flexiondotorg/flexiondotorg/internal/discord"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/haberdasher"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
	"github.com/flexiondotorg/flexiondotorg/internal/site"
)

var version = "dev"

const (
	defaultRunTimeout  = 2 * time.Minute
	defaultHTTPTimeout = 30 * time.Second
)

type cliOptions struct {
	showVersion bool
	verbose     bool
	configPath  string
	outDir      string
}

type runtimeDeps struct {
	now                   func() time.Time
	httpClient            *http.Client
	githubClient          *gh.Client
	socialFetcher         *site.SocialFetcher
	haberdasherHostBaseFn func(host string) string
	stdout                io.Writer
	stderr                io.Writer
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
		fmt.Fprintf(os.Stderr, "site: %v\n", err)
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

	ghClient := deps.githubClient
	if ghClient == nil {
		ghClient = gh.NewClient(os.Getenv("GITHUB_TOKEN"))
		ghClient.HTTPClient = deps.httpClient
	}

	logProgress := func(format string, a ...any) {
		if opts.verbose {
			fmt.Fprintf(deps.stderr, "site: "+format+"\n", a...)
		}
	}
	startedAt := time.Now()

	logProgress("fetching Haberdasher projects and package hosts...")
	catalogue := fetchHaberdasher(ctx, cfg, deps.httpClient, ghClient, deps.haberdasherHostBaseFn, deps.stderr)
	extraRepos := projectFullNames(catalogue, cfg.Projects)

	logProgress("fetching GitHub profile, stats, and sponsors...")
	ghProfile := fetchGitHubProfile(ctx, cfg, ghClient, extraRepos, deps.stderr)

	logProgress("fetching Discord, social profiles, and media feeds...")
	discordStatus := fetchDiscordStatus(ctx, deps.httpClient, cfg.Discord, deps.stderr)
	socialProfiles := fetchSocialProfiles(ctx, deps.httpClient, cfg.Site.Links, deps.socialFetcher, deps.stderr)
	podcastItems := fetchFeed(ctx, deps.httpClient, cfg.Feeds.Podcast, "podcast", deps.stderr)
	blogItems := fetchFeed(ctx, deps.httpClient, cfg.Feeds.Blog, "blog", deps.stderr)
	youtubeItems := fetchFeed(ctx, deps.httpClient, cfg.Feeds.YouTube, "youtube", deps.stderr)

	logProgress("building portfolio site data...")
	siteData, err := site.Build(site.Input{
		Config:      cfg,
		Catalogue:   catalogue,
		GitHub:      ghProfile,
		Discord:     discordStatus,
		Social:      socialProfiles,
		Podcast:     podcastItems,
		Blog:        blogItems,
		YouTube:     youtubeItems,
		GeneratedAt: deps.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("build site data: %w", err)
	}

	logProgress("writing static site assets to %s...", opts.outDir)
	if err := site.Write(opts.outDir, siteData); err != nil {
		return fmt.Errorf("write site: %w", err)
	}

	logProgress("done in %s", time.Since(startedAt).Round(100*time.Millisecond))
	return nil
}

func parseFlags(args []string, stderr io.Writer) (cliOptions, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("site", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.BoolVar(&opts.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&opts.verbose, "verbose", false, "log fetch and build progress to stderr")
	fs.StringVar(&opts.configPath, "config", "vitrine.yaml", "path to vitrine YAML configuration")
	fs.StringVar(&opts.outDir, "out", "dist", "directory for generated static site output")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}
	return opts, nil
}

func fetchHaberdasher(
	ctx context.Context,
	cfg *config.Config,
	httpClient *http.Client,
	ghClient *gh.Client,
	hostBaseFn func(host string) string,
	stderr io.Writer,
) *haberdasher.Catalogue {
	if !cfg.Haberdasher.Enabled && len(cfg.Projects) == 0 {
		return nil
	}

	fetcher := haberdasher.NewFetcher(httpClient, ghClient)
	opts := haberdasher.Options{
		ConfigURL:           cfg.Haberdasher.ConfigURL,
		Hosts:               cfg.Haberdasher.Hosts,
		Sort:                cfg.Haberdasher.Sort,
		LogoOverrides:       cfg.Haberdasher.LogoOverrides,
		PNGFallbacks:        cfg.Haberdasher.PNGFallbacks,
		HostBaseURLOverride: hostBaseFn,
	}

	var cat *haberdasher.Catalogue
	if cfg.Haberdasher.Enabled {
		fetched, err := fetcher.Fetch(ctx, opts)
		if err != nil {
			fmt.Fprintf(stderr, "warning: haberdasher fetch failed: %v\n", err)
		} else {
			cat = fetched
		}
	}

	if len(cfg.Projects) == 0 {
		return cat
	}

	var existing []haberdasher.Project
	if cat != nil {
		existing = cat.Projects
	}
	merged, err := fetcher.EnrichProjects(ctx, existing, toExtraProjects(cfg.Projects), opts)
	if err != nil {
		fmt.Fprintf(stderr, "warning: project enrichment failed: %v\n", err)
		return cat
	}
	if cat == nil {
		cat = &haberdasher.Catalogue{}
	}
	cat.Projects = merged
	return cat
}

func toExtraProjects(configured []config.ProjectConfig) []haberdasher.ExtraProject {
	extras := make([]haberdasher.ExtraProject, 0, len(configured))
	for _, p := range configured {
		extras = append(extras, haberdasher.ExtraProject{
			Repo:        p.Repo,
			Logo:        p.Logo,
			Language:    p.Language,
			Homepage:    p.Homepage,
			Description: p.Description,
		})
	}
	return extras
}

func projectFullNames(cat *haberdasher.Catalogue, configured []config.ProjectConfig) []string {
	seen := make(map[string]bool)
	var repos []string
	add := func(raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			return
		}
		seen[key] = true
		repos = append(repos, trimmed)
	}
	if cat != nil {
		for _, p := range cat.Projects {
			add(p.FullName)
		}
	}
	for _, p := range configured {
		add(p.Repo)
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

func fetchSocialProfiles(
	ctx context.Context,
	httpClient *http.Client,
	links []config.SiteLink,
	fetcher *site.SocialFetcher,
	stderr io.Writer,
) map[string]site.SocialProfile {
	if len(links) == 0 {
		return nil
	}
	if fetcher == nil {
		fetcher = &site.SocialFetcher{HTTPClient: httpClient}
	}
	profiles, errs := fetcher.FetchAll(ctx, links)
	for _, err := range errs {
		fmt.Fprintf(stderr, "warning: social profile fetch failed: %v\n", err)
	}
	return profiles
}

func fetchFeed(
	ctx context.Context,
	httpClient *http.Client,
	feedCfg *config.FeedConfig,
	label string,
	stderr io.Writer,
) []rss.Item {
	if feedCfg == nil || !feedCfg.Enabled {
		return nil
	}
	items, err := rss.Fetch(ctx, httpClient, feedCfg.URL, feedCfg.MaxItems)
	if err != nil {
		fmt.Fprintf(stderr, "warning: %s feed fetch failed: %v\n", label, err)
		return nil
	}
	return items
}
