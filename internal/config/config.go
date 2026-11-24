// Package config loads and validates Vitrine configuration and colour palettes.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// PresetObsidianGilt is the built-in Obsidian and Gilt palette preset.
	PresetObsidianGilt = "obsidian-gilt"
	// PresetCatppuccinMocha is the built-in Catppuccin Mocha dark palette preset.
	PresetCatppuccinMocha = "catppuccin-mocha"
	// PresetCatppuccinLatte is the built-in Catppuccin Latte light palette preset.
	PresetCatppuccinLatte = "catppuccin-latte"

	// DefaultPreset is the theme preset selected when none is configured.
	DefaultPreset = PresetCatppuccinMocha
	// DefaultDarkPreset is the default dark theme preset.
	DefaultDarkPreset = PresetCatppuccinMocha
	// DefaultLightPreset is the default light theme preset.
	DefaultLightPreset = PresetCatppuccinLatte

	// HaberdasherSortStars sorts curated Haberdasher projects descending by GitHub star count.
	HaberdasherSortStars = "stars"
	// HaberdasherSortRelease sorts curated Haberdasher projects descending by latest release timestamp.
	HaberdasherSortRelease = "release"

	defaultMaxItems                = 5
	defaultContributionsWindowDays = 180
	defaultTwitchPrimary           = "https://decapi.me/twitch/uptime"
	defaultTwitchFallback          = "https://img.shields.io/twitch/status"
	defaultDiscordInvite           = "https://discord.com/api/v9/invites"
	defaultDiscordWidget           = "https://discord.com/api/guilds"
	defaultDiscordShields          = "https://img.shields.io/discord"
)

var defaultHaberdasherHosts = []string{
	"apt.wimpys.world",
	"rpm.wimpys.world",
	"pkg.wimpys.world",
	"apk.wimpys.world",
	"flatpak.wimpys.world",
	"brew.wimpys.world",
	"scoop.wimpys.world",
	"nix.wimpys.world",
}

var builtinPresets = map[string]Palette{
	PresetCatppuccinMocha: {
		Base:            "#1e1e2e",
		Mantle:          "#181825",
		Surface0:        "#313244",
		Surface1:        "#45475a",
		Text:            "#cdd6f4",
		Subtext:         "#a6adc8",
		Muted:           "#6c7086",
		AccentPrimary:   "#cba6f7",
		AccentSecondary: "#89b4fa",
		AccentTertiary:  "#94e2d5",
		AccentPink:      "#f5c2e7",
		AccentGreen:     "#a6e3a1",
		AccentRed:       "#f38ba8",
		AccentPeach:     "#fab387",
		AccentYellow:    "#f9e2af",
		AccentSapphire:  "#74c7ec",
		IsLight:         false,
	},
	PresetCatppuccinLatte: {
		Base:            "#eff1f5",
		Mantle:          "#e6e9ef",
		Surface0:        "#ccd0da",
		Surface1:        "#bcc0cc",
		Text:            "#4c4f69",
		Subtext:         "#6c6f85",
		Muted:           "#9ca0b0",
		AccentPrimary:   "#8839ef",
		AccentSecondary: "#1e66f5",
		AccentTertiary:  "#179299",
		AccentPink:      "#ea76cb",
		AccentGreen:     "#40a02b",
		AccentRed:       "#d20f39",
		AccentPeach:     "#fe640b",
		AccentYellow:    "#df8e1d",
		AccentSapphire:  "#209fb5",
		IsLight:         true,
	},
	PresetObsidianGilt: {
		Base:            "#121216",
		Mantle:          "#0d0d11",
		Surface0:        "#19191f",
		Surface1:        "#7a5e25",
		Text:            "#f4efe4",
		Subtext:         "#b3aa99",
		Muted:           "#7d7668",
		AccentPrimary:   "#cfa84f",
		AccentSecondary: "#f0d890",
		AccentTertiary:  "#7a5e25",
		AccentPink:      "#f0d890",
		AccentGreen:     "#4fb885",
		AccentRed:       "#c9545c",
		AccentPeach:     "#cfa84f",
		AccentYellow:    "#f0d890",
		AccentSapphire:  "#6c9bc9",
		IsLight:         false,
	},
}

// Palette holds the colour tokens used to render Vitrine tiles and views.
type Palette struct {
	Base            string `yaml:"base"`
	Mantle          string `yaml:"mantle"`
	Surface0        string `yaml:"surface0"`
	Surface1        string `yaml:"surface1"`
	Text            string `yaml:"text"`
	Subtext         string `yaml:"subtext"`
	Muted           string `yaml:"muted"`
	AccentPrimary   string `yaml:"accent_primary"`
	AccentSecondary string `yaml:"accent_secondary"`
	AccentTertiary  string `yaml:"accent_tertiary"`
	AccentPink      string `yaml:"accent_pink"`
	AccentGreen     string `yaml:"accent_green"`
	AccentRed       string `yaml:"accent_red"`
	AccentPeach     string `yaml:"accent_peach"`
	AccentYellow    string `yaml:"accent_yellow"`
	AccentSapphire  string `yaml:"accent_sapphire"`
	IsLight         bool   `yaml:"is_light"`
}

// ThemeConfig selects built-in dark and light palette presets and optional per-token overrides.
type ThemeConfig struct {
	Preset      string  `yaml:"preset"`
	DarkPreset  string  `yaml:"dark_preset"`
	LightPreset string  `yaml:"light_preset"`
	Overrides   Palette `yaml:"overrides"`
}

// Theme is an alias for ThemeConfig kept for compatibility with existing callers.
type Theme = ThemeConfig

// NavLink describes a navigation pill shown under the hero banner.
type NavLink struct {
	Label  string `yaml:"label"`
	Slug   string `yaml:"slug"`
	URL    string `yaml:"url"`
	Icon   string `yaml:"icon"`
	Accent string `yaml:"accent"`
}

// ProfileConfig defines the profile identity, hero banner, bio, and navigation links.
type ProfileConfig struct {
	Name         string    `yaml:"name"`
	Handle       string    `yaml:"handle"`
	Eyebrow      string    `yaml:"eyebrow"`
	HeroTitle    string    `yaml:"hero_title"`
	HeroSubtitle string    `yaml:"hero_subtitle"`
	HeroURL      string    `yaml:"hero_url"`
	HeroCTA      string    `yaml:"hero_cta"`
	Bio          string    `yaml:"bio"`
	Links        []NavLink `yaml:"links"`
}

// SponsorConfig defines a fallback GitHub sponsor entry.
type SponsorConfig struct {
	Login string `yaml:"login"`
	URL   string `yaml:"url"`
}

// GitHubConfig configures GitHub REST data fetching and sponsor fallbacks.
type GitHubConfig struct {
	Username                string          `yaml:"username"`
	Orgs                    []string        `yaml:"orgs"`
	ExcludeOwners           []string        `yaml:"exclude_owners"`
	MaxItems                int             `yaml:"max_items"`
	ContributionsWindowDays int             `yaml:"contributions_window_days"`
	SponsorsURL             string          `yaml:"sponsors_url"`
	SponsorsFallback        []SponsorConfig `yaml:"sponsors_fallback"`
}

// HaberdasherConfig configures Haberdasher project and repository manifest fetching.
type HaberdasherConfig struct {
	Enabled       bool              `yaml:"enabled"`
	ConfigURL     string            `yaml:"config_url"`
	Sort          string            `yaml:"sort"`
	Hosts         []string          `yaml:"hosts"`
	LogoOverrides map[string]string `yaml:"logo_overrides"`
	PNGFallbacks  map[string]string `yaml:"png_fallbacks"`
}

// ProjectConfig defines a standalone or override project entry for the Projects grid.
type ProjectConfig struct {
	Repo        string `yaml:"repo"`
	Logo        string `yaml:"logo"`
	Language    string `yaml:"language"`
	Homepage    string `yaml:"homepage"`
	Description string `yaml:"description"`
}

// FeedConfig configures a single RSS 2.0 or Atom feed section.
type FeedConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Title    string `yaml:"title"`
	URL      string `yaml:"url"`
	SiteURL  string `yaml:"site_url"`
	CoverURL string `yaml:"cover_url"`
	MaxItems int    `yaml:"max_items"`
}

// FeedsConfig groups optional podcast, blog, and YouTube feeds.
type FeedsConfig struct {
	Podcast *FeedConfig `yaml:"podcast"`
	Blog    *FeedConfig `yaml:"blog"`
	YouTube *FeedConfig `yaml:"youtube"`
}

// TwitchConfig configures optional Twitch live status checks.
type TwitchConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Channel         string `yaml:"channel"`
	URL             string `yaml:"url"`
	PrimaryBaseURL  string `yaml:"primary_base_url"`
	FallbackBaseURL string `yaml:"fallback_base_url"`
}

// DiscordConfig configures optional live Discord online and member count checks.
type DiscordConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Name           string `yaml:"name"`
	Tagline        string `yaml:"tagline"`
	URL            string `yaml:"url"`
	InviteCode     string `yaml:"invite_code"`
	GuildID        string `yaml:"guild_id"`
	InviteBaseURL  string `yaml:"invite_base_url"`
	WidgetBaseURL  string `yaml:"widget_base_url"`
	ShieldsBaseURL string `yaml:"shields_base_url"`
}

// OutputConfig configures asset URL generation for rendered output.
type OutputConfig struct {
	AssetBaseURL string `yaml:"asset_base_url"`
}

// Config is the top-level structure parsed from vitrine.yaml.
type Config struct {
	Profile     ProfileConfig     `yaml:"profile"`
	Theme       ThemeConfig       `yaml:"theme"`
	GitHub      GitHubConfig      `yaml:"github"`
	Haberdasher HaberdasherConfig `yaml:"haberdasher"`
	Projects    []ProjectConfig   `yaml:"projects"`
	Feeds       FeedsConfig       `yaml:"feeds"`
	Twitch      TwitchConfig      `yaml:"twitch"`
	Discord     DiscordConfig     `yaml:"discord"`
	Output      OutputConfig      `yaml:"output"`
}

// DefaultHaberdasherHosts returns a copy of the eight canonical .wimpys.world hosts.
func DefaultHaberdasherHosts() []string {
	hosts := make([]string, len(defaultHaberdasherHosts))
	copy(hosts, defaultHaberdasherHosts)
	return hosts
}

// Presets returns a copy of all built-in palette presets keyed by preset name.
func Presets() map[string]Palette {
	out := make(map[string]Palette, len(builtinPresets))
	for name, palette := range builtinPresets {
		out[name] = palette
	}
	return out
}

// ResolvePalette resolves a ThemeConfig's primary (dark) preset and validates any custom hex overrides.
func ResolvePalette(theme ThemeConfig) (Palette, error) {
	return theme.ResolvePalette()
}

// ResolveDarkPalette resolves a ThemeConfig's dark preset and validates any custom hex overrides.
func ResolveDarkPalette(theme ThemeConfig) (Palette, error) {
	return theme.ResolveDarkPalette()
}

// ResolveLightPalette resolves a ThemeConfig's light preset and validates any custom hex overrides.
func ResolveLightPalette(theme ThemeConfig) (Palette, error) {
	return theme.ResolveLightPalette()
}

// ResolvePalette resolves the configured primary (dark) theme preset and applies validated overrides.
func (c *Config) ResolvePalette() (Palette, error) {
	if c == nil {
		return Palette{}, errors.New("config is nil")
	}
	return c.Theme.ResolvePalette()
}

// ResolveDarkPalette resolves the configured dark theme preset and applies validated overrides.
func (c *Config) ResolveDarkPalette() (Palette, error) {
	if c == nil {
		return Palette{}, errors.New("config is nil")
	}
	return c.Theme.ResolveDarkPalette()
}

// ResolveLightPalette resolves the configured light theme preset and applies validated overrides.
func (c *Config) ResolveLightPalette() (Palette, error) {
	if c == nil {
		return Palette{}, errors.New("config is nil")
	}
	return c.Theme.ResolveLightPalette()
}

// ResolvePalette resolves the primary (dark) preset and merges validated field-level overrides.
func (t ThemeConfig) ResolvePalette() (Palette, error) {
	return t.ResolveDarkPalette()
}

// ResolveDarkPalette resolves the dark preset and merges validated field-level overrides.
func (t ThemeConfig) ResolveDarkPalette() (Palette, error) {
	return t.resolvePreset(t.darkPresetName())
}

// ResolveLightPalette resolves the light preset and merges validated field-level overrides.
func (t ThemeConfig) ResolveLightPalette() (Palette, error) {
	return t.resolvePreset(t.lightPresetName())
}

func (t ThemeConfig) darkPresetName() string {
	if name := strings.ToLower(strings.TrimSpace(t.DarkPreset)); name != "" {
		return name
	}
	if name := strings.ToLower(strings.TrimSpace(t.Preset)); name != "" {
		return name
	}
	return DefaultDarkPreset
}

func (t ThemeConfig) lightPresetName() string {
	if name := strings.ToLower(strings.TrimSpace(t.LightPreset)); name != "" {
		return name
	}
	preset := strings.ToLower(strings.TrimSpace(t.Preset))
	dark := t.darkPresetName()
	if preset == "" && dark == PresetCatppuccinMocha {
		return DefaultLightPreset
	}
	if preset != "" {
		return preset
	}
	return dark
}

func (t ThemeConfig) resolvePreset(name string) (Palette, error) {
	base, ok := builtinPresets[name]
	if !ok {
		return Palette{}, fmt.Errorf("unknown theme preset %q", name)
	}

	fields := []struct {
		name  string
		value string
		dest  *string
	}{
		{name: "base", value: t.Overrides.Base, dest: &base.Base},
		{name: "mantle", value: t.Overrides.Mantle, dest: &base.Mantle},
		{name: "surface0", value: t.Overrides.Surface0, dest: &base.Surface0},
		{name: "surface1", value: t.Overrides.Surface1, dest: &base.Surface1},
		{name: "text", value: t.Overrides.Text, dest: &base.Text},
		{name: "subtext", value: t.Overrides.Subtext, dest: &base.Subtext},
		{name: "muted", value: t.Overrides.Muted, dest: &base.Muted},
		{name: "accent_primary", value: t.Overrides.AccentPrimary, dest: &base.AccentPrimary},
		{name: "accent_secondary", value: t.Overrides.AccentSecondary, dest: &base.AccentSecondary},
		{name: "accent_tertiary", value: t.Overrides.AccentTertiary, dest: &base.AccentTertiary},
		{name: "accent_pink", value: t.Overrides.AccentPink, dest: &base.AccentPink},
		{name: "accent_green", value: t.Overrides.AccentGreen, dest: &base.AccentGreen},
		{name: "accent_red", value: t.Overrides.AccentRed, dest: &base.AccentRed},
		{name: "accent_peach", value: t.Overrides.AccentPeach, dest: &base.AccentPeach},
		{name: "accent_yellow", value: t.Overrides.AccentYellow, dest: &base.AccentYellow},
		{name: "accent_sapphire", value: t.Overrides.AccentSapphire, dest: &base.AccentSapphire},
	}

	for _, field := range fields {
		trimmed := strings.TrimSpace(field.value)
		if trimmed == "" {
			continue
		}
		if !isValidHexColor(trimmed) {
			return Palette{}, fmt.Errorf("invalid hex colour override for %s: %q (expected #RRGGBB)", field.name, field.value)
		}
		*field.dest = trimmed
	}

	return base, nil
}

// Load reads, parses, defaults, and validates a Vitrine YAML configuration file.
func Load(path string) (*Config, error) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return nil, errors.New("config path must not be empty")
	}

	// #nosec G304 -- configuration file path is explicitly provided by the caller.
	data, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}

// Parse unmarshals YAML configuration bytes, applies defaults, and validates the result.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	if err := cfg.applyDefaultsAndValidate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaultsAndValidate() error {
	c.Theme.Preset = strings.ToLower(strings.TrimSpace(c.Theme.Preset))
	c.Theme.DarkPreset = strings.ToLower(strings.TrimSpace(c.Theme.DarkPreset))
	c.Theme.LightPreset = strings.ToLower(strings.TrimSpace(c.Theme.LightPreset))

	if c.Theme.Preset != "" {
		if _, ok := builtinPresets[c.Theme.Preset]; !ok {
			return fmt.Errorf("unknown theme preset %q", c.Theme.Preset)
		}
	}

	c.Theme.DarkPreset = c.Theme.darkPresetName()
	c.Theme.LightPreset = c.Theme.lightPresetName()
	if c.Theme.Preset == "" {
		c.Theme.Preset = c.Theme.DarkPreset
	}

	if _, err := c.Theme.ResolveDarkPalette(); err != nil {
		return err
	}
	if _, err := c.Theme.ResolveLightPalette(); err != nil {
		return err
	}

	for i, link := range c.Profile.Links {
		if strings.TrimSpace(link.Label) == "" {
			return fmt.Errorf("profile.links[%d]: label must not be empty", i)
		}
		if strings.TrimSpace(link.URL) == "" {
			return fmt.Errorf("profile.links[%d]: url must not be empty", i)
		}
	}

	if err := c.applyGitHubDefaultsAndValidate(); err != nil {
		return err
	}

	if err := c.applyHaberdasherDefaultsAndValidate(); err != nil {
		return err
	}

	if err := c.applyProjectsDefaultsAndValidate(); err != nil {
		return err
	}

	if err := validateFeed("feeds.podcast", c.Feeds.Podcast); err != nil {
		return err
	}
	if err := validateFeed("feeds.blog", c.Feeds.Blog); err != nil {
		return err
	}
	if err := validateFeed("feeds.youtube", c.Feeds.YouTube); err != nil {
		return err
	}

	if err := c.applyTwitchDefaultsAndValidate(); err != nil {
		return err
	}
	if err := c.applyDiscordDefaultsAndValidate(); err != nil {
		return err
	}

	return nil
}

func (c *Config) applyGitHubDefaultsAndValidate() error {
	if c.GitHub.MaxItems < 0 {
		return fmt.Errorf("github.max_items must not be negative: %d", c.GitHub.MaxItems)
	}
	if c.GitHub.MaxItems == 0 {
		c.GitHub.MaxItems = defaultMaxItems
	}
	if c.GitHub.ContributionsWindowDays <= 0 {
		c.GitHub.ContributionsWindowDays = defaultContributionsWindowDays
	}
	c.GitHub.Username = strings.TrimSpace(c.GitHub.Username)
	if len(c.GitHub.ExcludeOwners) > 0 {
		cleaned := make([]string, 0, len(c.GitHub.ExcludeOwners))
		for _, owner := range c.GitHub.ExcludeOwners {
			if trimmed := strings.TrimSpace(owner); trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		c.GitHub.ExcludeOwners = cleaned
	}
	c.GitHub.SponsorsURL = strings.TrimSpace(c.GitHub.SponsorsURL)
	if c.GitHub.SponsorsURL == "" && c.GitHub.Username != "" {
		c.GitHub.SponsorsURL = "https://github.com/sponsors/" + c.GitHub.Username
	}
	for i := range c.GitHub.SponsorsFallback {
		login := strings.TrimSpace(c.GitHub.SponsorsFallback[i].Login)
		if login == "" {
			return fmt.Errorf("github.sponsors_fallback[%d]: login must not be empty", i)
		}
		c.GitHub.SponsorsFallback[i].Login = login
		if strings.TrimSpace(c.GitHub.SponsorsFallback[i].URL) == "" {
			c.GitHub.SponsorsFallback[i].URL = "https://github.com/" + login
		}
	}
	return nil
}

func (c *Config) applyHaberdasherDefaultsAndValidate() error {
	if len(c.Haberdasher.Hosts) == 0 {
		c.Haberdasher.Hosts = DefaultHaberdasherHosts()
	}
	sortMode := strings.ToLower(strings.TrimSpace(c.Haberdasher.Sort))
	if sortMode == "" {
		sortMode = HaberdasherSortStars
	}
	switch sortMode {
	case HaberdasherSortStars, HaberdasherSortRelease:
		c.Haberdasher.Sort = sortMode
	default:
		return fmt.Errorf("haberdasher.sort must be %q or %q, got %q", HaberdasherSortStars, HaberdasherSortRelease, c.Haberdasher.Sort)
	}
	return nil
}

func (c *Config) applyProjectsDefaultsAndValidate() error {
	for i := range c.Projects {
		c.Projects[i].Repo = strings.TrimSpace(c.Projects[i].Repo)
		c.Projects[i].Logo = strings.TrimSpace(c.Projects[i].Logo)
		c.Projects[i].Language = strings.TrimSpace(c.Projects[i].Language)
		c.Projects[i].Homepage = strings.TrimSpace(c.Projects[i].Homepage)
		c.Projects[i].Description = strings.TrimSpace(c.Projects[i].Description)
		if !isValidRepoFullName(c.Projects[i].Repo) {
			return fmt.Errorf("projects[%d].repo must be in owner/repo format, got %q", i, c.Projects[i].Repo)
		}
	}
	return nil
}

func isValidRepoFullName(repo string) bool {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return false
	}
	if strings.ContainsAny(owner, " \t\r\n") || strings.ContainsAny(name, " \t\r\n") {
		return false
	}
	return true
}

func (c *Config) applyTwitchDefaultsAndValidate() error {
	c.Twitch.Channel = strings.TrimSpace(c.Twitch.Channel)
	c.Twitch.URL = strings.TrimSpace(c.Twitch.URL)
	c.Twitch.PrimaryBaseURL = strings.TrimSpace(c.Twitch.PrimaryBaseURL)
	c.Twitch.FallbackBaseURL = strings.TrimSpace(c.Twitch.FallbackBaseURL)
	if c.Twitch.Enabled && c.Twitch.Channel == "" {
		return errors.New("twitch.channel must not be empty when twitch is enabled")
	}
	if c.Twitch.URL == "" && c.Twitch.Channel != "" {
		c.Twitch.URL = "https://www.twitch.tv/" + c.Twitch.Channel
	}
	if c.Twitch.PrimaryBaseURL == "" {
		c.Twitch.PrimaryBaseURL = defaultTwitchPrimary
	}
	if c.Twitch.FallbackBaseURL == "" {
		c.Twitch.FallbackBaseURL = defaultTwitchFallback
	}
	return nil
}

func (c *Config) applyDiscordDefaultsAndValidate() error {
	c.Discord.Name = strings.TrimSpace(c.Discord.Name)
	c.Discord.Tagline = strings.TrimSpace(c.Discord.Tagline)
	c.Discord.URL = strings.TrimSpace(c.Discord.URL)
	c.Discord.InviteCode = strings.TrimSpace(c.Discord.InviteCode)
	c.Discord.GuildID = strings.TrimSpace(c.Discord.GuildID)
	c.Discord.InviteBaseURL = strings.TrimSpace(c.Discord.InviteBaseURL)
	c.Discord.WidgetBaseURL = strings.TrimSpace(c.Discord.WidgetBaseURL)
	c.Discord.ShieldsBaseURL = strings.TrimSpace(c.Discord.ShieldsBaseURL)
	if c.Discord.Enabled {
		if c.Discord.InviteCode == "" && c.Discord.GuildID == "" {
			return errors.New("discord.invite_code or discord.guild_id must be set when discord is enabled")
		}
		if c.Discord.URL == "" {
			return errors.New("discord.url must not be empty when discord is enabled")
		}
	}
	if c.Discord.InviteBaseURL == "" {
		c.Discord.InviteBaseURL = defaultDiscordInvite
	}
	if c.Discord.WidgetBaseURL == "" {
		c.Discord.WidgetBaseURL = defaultDiscordWidget
	}
	if c.Discord.ShieldsBaseURL == "" {
		c.Discord.ShieldsBaseURL = defaultDiscordShields
	}
	return nil
}

func validateFeed(name string, feed *FeedConfig) error {
	if feed == nil {
		return nil
	}
	if feed.MaxItems < 0 {
		return fmt.Errorf("%s.max_items must not be negative: %d", name, feed.MaxItems)
	}
	if feed.MaxItems == 0 {
		feed.MaxItems = defaultMaxItems
	}
	feed.URL = strings.TrimSpace(feed.URL)
	if feed.Enabled && feed.URL == "" {
		return fmt.Errorf("%s.url must not be empty when enabled", name)
	}
	return nil
}

func isValidHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}
