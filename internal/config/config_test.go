package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPresets(t *testing.T) {
	t.Parallel()

	presets := Presets()
	if len(presets) != 2 {
		t.Fatalf("Presets() returned %d presets, want 2", len(presets))
	}
	for _, name := range []string{PresetCatppuccinMocha, PresetCatppuccinLatte} {
		p, ok := presets[name]
		if !ok {
			t.Fatalf("missing built-in preset %q", name)
		}
		for token, value := range map[string]string{
			"base":             p.Base,
			"mantle":           p.Mantle,
			"surface0":         p.Surface0,
			"surface1":         p.Surface1,
			"text":             p.Text,
			"subtext":          p.Subtext,
			"muted":            p.Muted,
			"accent_primary":   p.AccentPrimary,
			"accent_secondary": p.AccentSecondary,
			"accent_tertiary":  p.AccentTertiary,
			"accent_pink":      p.AccentPink,
			"accent_green":     p.AccentGreen,
			"accent_red":       p.AccentRed,
			"accent_peach":     p.AccentPeach,
			"accent_yellow":    p.AccentYellow,
			"accent_sapphire":  p.AccentSapphire,
		} {
			if !isValidHexColor(value) {
				t.Errorf("preset %q token %s has invalid hex %q", name, token, value)
			}
		}
	}

	wantMocha := Palette{
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
	}
	if got := presets[PresetCatppuccinMocha]; got != wantMocha {
		t.Fatalf("catppuccin-mocha palette mismatch:\ngot:  %+v\nwant: %+v", got, wantMocha)
	}

	wantLatte := Palette{
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
	}
	if got := presets[PresetCatppuccinLatte]; got != wantLatte {
		t.Fatalf("catppuccin-latte palette mismatch:\ngot:  %+v\nwant: %+v", got, wantLatte)
	}

	// Mutating the returned map must not alter built-in presets.
	presets[PresetCatppuccinMocha] = Palette{Base: "#000000"}
	if Presets()[PresetCatppuccinMocha].Base != "#1e1e2e" {
		t.Fatal("Presets returned a mutable reference to internal presets")
	}
}

func TestResolvePaletteDualPresetsAndValidation(t *testing.T) {
	t.Parallel()

	p, err := ResolvePalette(ThemeConfig{
		Preset: PresetCatppuccinMocha,
		Overrides: Palette{
			Base:          "#0a0b0c",
			AccentPrimary: "#ABCDEF",
		},
	})
	if err != nil {
		t.Fatalf("ResolvePalette failed: %v", err)
	}
	if p.Base != "#0a0b0c" {
		t.Errorf("Base override not applied: got %q", p.Base)
	}
	if p.AccentPrimary != "#ABCDEF" {
		t.Errorf("AccentPrimary override not applied: got %q", p.AccentPrimary)
	}
	if p.Surface0 != builtinPresets[PresetCatppuccinMocha].Surface0 {
		t.Errorf("Surface0 changed unexpectedly: got %q", p.Surface0)
	}

	// Empty ThemeConfig defaults dark to catppuccin-mocha and light to catppuccin-latte.
	empty := ThemeConfig{}
	darkDefault, err := ResolveDarkPalette(empty)
	if err != nil {
		t.Fatalf("ResolveDarkPalette default failed: %v", err)
	}
	if darkDefault != builtinPresets[PresetCatppuccinMocha] {
		t.Fatalf("empty theme dark palette = %+v, want catppuccin-mocha", darkDefault)
	}
	primaryDefault, err := ResolvePalette(empty)
	if err != nil {
		t.Fatalf("ResolvePalette default failed: %v", err)
	}
	if primaryDefault != darkDefault {
		t.Fatalf("ResolvePalette != ResolveDarkPalette for empty theme")
	}
	lightDefault, err := ResolveLightPalette(empty)
	if err != nil {
		t.Fatalf("ResolveLightPalette default failed: %v", err)
	}
	if lightDefault != builtinPresets[PresetCatppuccinLatte] {
		t.Fatalf("empty theme light palette = %+v, want catppuccin-latte", lightDefault)
	}
	if !lightDefault.IsLight {
		t.Fatal("catppuccin-latte must have IsLight=true")
	}

	// Dual preset matrix.
	cases := []struct {
		name      string
		theme     ThemeConfig
		wantDark  string
		wantLight string
	}{
		{
			name:      "empty defaults to mocha and latte",
			theme:     ThemeConfig{},
			wantDark:  PresetCatppuccinMocha,
			wantLight: PresetCatppuccinLatte,
		},
		{
			name:      "dark_preset mocha without preset defaults light to latte",
			theme:     ThemeConfig{DarkPreset: PresetCatppuccinMocha},
			wantDark:  PresetCatppuccinMocha,
			wantLight: PresetCatppuccinLatte,
		},
		{
			name:      "dark_preset latte only falls back light to dark_preset",
			theme:     ThemeConfig{DarkPreset: PresetCatppuccinLatte},
			wantDark:  PresetCatppuccinLatte,
			wantLight: PresetCatppuccinLatte,
		},
		{
			name:      "explicit preset mocha applies to both dark and light",
			theme:     ThemeConfig{Preset: PresetCatppuccinMocha},
			wantDark:  PresetCatppuccinMocha,
			wantLight: PresetCatppuccinMocha,
		},
		{
			name:      "explicit preset latte applies to both dark and light",
			theme:     ThemeConfig{Preset: PresetCatppuccinLatte},
			wantDark:  PresetCatppuccinLatte,
			wantLight: PresetCatppuccinLatte,
		},
		{
			name:      "explicit dark_preset mocha and light_preset latte",
			theme:     ThemeConfig{DarkPreset: PresetCatppuccinMocha, LightPreset: PresetCatppuccinLatte},
			wantDark:  PresetCatppuccinMocha,
			wantLight: PresetCatppuccinLatte,
		},
		{
			name:      "explicit dark_preset latte and light_preset mocha",
			theme:     ThemeConfig{DarkPreset: PresetCatppuccinLatte, LightPreset: PresetCatppuccinMocha},
			wantDark:  PresetCatppuccinLatte,
			wantLight: PresetCatppuccinMocha,
		},
		{
			name:      "preset with light_preset override",
			theme:     ThemeConfig{Preset: PresetCatppuccinMocha, LightPreset: PresetCatppuccinLatte},
			wantDark:  PresetCatppuccinMocha,
			wantLight: PresetCatppuccinLatte,
		},
		{
			name:      "preset with dark_preset override",
			theme:     ThemeConfig{Preset: PresetCatppuccinLatte, DarkPreset: PresetCatppuccinMocha},
			wantDark:  PresetCatppuccinMocha,
			wantLight: PresetCatppuccinLatte,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotDark, err := tc.theme.ResolveDarkPalette()
			if err != nil {
				t.Fatalf("ResolveDarkPalette error: %v", err)
			}
			if gotDark != builtinPresets[tc.wantDark] {
				t.Errorf("dark palette = %+v, want %q (%+v)", gotDark, tc.wantDark, builtinPresets[tc.wantDark])
			}
			gotLight, err := tc.theme.ResolveLightPalette()
			if err != nil {
				t.Fatalf("ResolveLightPalette error: %v", err)
			}
			if gotLight != builtinPresets[tc.wantLight] {
				t.Errorf("light palette = %+v, want %q (%+v)", gotLight, tc.wantLight, builtinPresets[tc.wantLight])
			}
		})
	}

	// Overrides preserve IsLight on light preset.
	overriddenLight, err := ResolveLightPalette(ThemeConfig{
		LightPreset: PresetCatppuccinLatte,
		Overrides:   Palette{AccentGreen: "#11aa22"},
	})
	if err != nil {
		t.Fatalf("ResolveLightPalette override failed: %v", err)
	}
	if !overriddenLight.IsLight || overriddenLight.AccentGreen != "#11aa22" {
		t.Fatalf("unexpected overridden light palette: %+v", overriddenLight)
	}

	// Unknown preset fails.
	if _, err := ResolvePalette(Theme{Preset: "unknown-theme"}); err == nil {
		t.Fatal("expected error for unknown preset")
	}
	if _, err := ResolveDarkPalette(ThemeConfig{DarkPreset: "unknown-dark"}); err == nil {
		t.Fatal("expected error for unknown dark preset")
	}
	if _, err := ResolveLightPalette(ThemeConfig{LightPreset: "unknown-light"}); err == nil {
		t.Fatal("expected error for unknown light preset")
	}

	// Invalid hex overrides fail.
	badHexes := []string{"1e1e2e", "#fff", "#12345g", "#1234567", "rgb(0,0,0)"}
	for _, bad := range badHexes {
		_, err := ResolvePalette(Theme{
			Preset:    PresetCatppuccinMocha,
			Overrides: Palette{AccentRed: bad},
		})
		if err == nil {
			t.Errorf("expected error for invalid hex override %q", bad)
		}
	}
}

func TestParseDefaultsAndFullConfig(t *testing.T) {
	t.Parallel()

	minimal, err := Parse([]byte(`
github:
  username: flexiondotorg
`))
	if err != nil {
		t.Fatalf("Parse minimal config failed: %v", err)
	}

	if minimal.Theme.Preset != DefaultPreset {
		t.Errorf("Theme.Preset = %q, want %q", minimal.Theme.Preset, DefaultPreset)
	}
	if minimal.Theme.DarkPreset != DefaultDarkPreset {
		t.Errorf("Theme.DarkPreset = %q, want %q", minimal.Theme.DarkPreset, DefaultDarkPreset)
	}
	if minimal.Theme.LightPreset != DefaultLightPreset {
		t.Errorf("Theme.LightPreset = %q, want %q", minimal.Theme.LightPreset, DefaultLightPreset)
	}
	if darkP, err := minimal.ResolveDarkPalette(); err != nil || darkP != builtinPresets[PresetCatppuccinMocha] {
		t.Errorf("minimal.ResolveDarkPalette() = (%+v, %v), want catppuccin-mocha", darkP, err)
	}
	if lightP, err := minimal.ResolveLightPalette(); err != nil || lightP != builtinPresets[PresetCatppuccinLatte] {
		t.Errorf("minimal.ResolveLightPalette() = (%+v, %v), want catppuccin-latte", lightP, err)
	}

	if minimal.GitHub.MaxItems != 5 {
		t.Errorf("GitHub.MaxItems = %d, want 5", minimal.GitHub.MaxItems)
	}
	if minimal.GitHub.ContributionsWindowDays != 180 {
		t.Errorf("GitHub.ContributionsWindowDays = %d, want 180", minimal.GitHub.ContributionsWindowDays)
	}
	if minimal.GitHub.SponsorsURL != "https://github.com/sponsors/flexiondotorg" {
		t.Errorf("GitHub.SponsorsURL = %q", minimal.GitHub.SponsorsURL)
	}
	wantHosts := []string{
		"apt.wimpys.world",
		"rpm.wimpys.world",
		"pkg.wimpys.world",
		"apk.wimpys.world",
		"flatpak.wimpys.world",
		"brew.wimpys.world",
		"scoop.wimpys.world",
		"nix.wimpys.world",
	}
	if !slices.Equal(minimal.Haberdasher.Hosts, wantHosts) {
		t.Errorf("Haberdasher.Hosts = %v, want %v", minimal.Haberdasher.Hosts, wantHosts)
	}
	if minimal.Haberdasher.Sort != HaberdasherSortStars {
		t.Errorf("Haberdasher.Sort = %q, want %q", minimal.Haberdasher.Sort, HaberdasherSortStars)
	}
	if minimal.Twitch.PrimaryBaseURL != defaultTwitchPrimary {
		t.Errorf("Twitch.PrimaryBaseURL = %q, want %q", minimal.Twitch.PrimaryBaseURL, defaultTwitchPrimary)
	}
	if minimal.Twitch.FallbackBaseURL != defaultTwitchFallback {
		t.Errorf("Twitch.FallbackBaseURL = %q, want %q", minimal.Twitch.FallbackBaseURL, defaultTwitchFallback)
	}
	if minimal.Discord.InviteBaseURL != defaultDiscordInvite {
		t.Errorf("Discord.InviteBaseURL = %q, want %q", minimal.Discord.InviteBaseURL, defaultDiscordInvite)
	}
	if minimal.Discord.WidgetBaseURL != defaultDiscordWidget {
		t.Errorf("Discord.WidgetBaseURL = %q, want %q", minimal.Discord.WidgetBaseURL, defaultDiscordWidget)
	}
	if minimal.Discord.ShieldsBaseURL != defaultDiscordShields {
		t.Errorf("Discord.ShieldsBaseURL = %q, want %q", minimal.Discord.ShieldsBaseURL, defaultDiscordShields)
	}
	if minimal.Site.DefaultDeck != SiteDeckProjects {
		t.Errorf("Site.DefaultDeck = %q, want %q", minimal.Site.DefaultDeck, SiteDeckProjects)
	}

	// Legacy single-theme preset parses identically for dark and light.
	legacy, err := Parse([]byte("theme:\n  preset: catppuccin-latte\n"))
	if err != nil {
		t.Fatalf("Parse legacy preset failed: %v", err)
	}
	if legacy.Theme.DarkPreset != PresetCatppuccinLatte || legacy.Theme.LightPreset != PresetCatppuccinLatte {
		t.Errorf("legacy preset resolved to dark=%q light=%q, want catppuccin-latte for both", legacy.Theme.DarkPreset, legacy.Theme.LightPreset)
	}

	fullYAML := `
profile:
  name: Martin Wimpress
  handle: flexiondotorg
  eyebrow: Open Source
  hero_title: Wimpy's World
  hero_subtitle: Linux and Go
  hero_url: https://wimpys.world
  hero_cta: Visit site
  bio: Builder of open source software.
  links:
    - label: Website
      slug: website
      url: https://wimpys.world
      icon: globe
      accent: mauve
theme:
  dark_preset: catppuccin-mocha
  light_preset: catppuccin-latte
  overrides:
    accent_green: "#11aa22"
github:
  username: flexiondotorg
  orgs:
    - wimpysworld
  exclude_owners:
    - " flexiondotorg "
    - wimpysworld
    - ""
  max_items: 8
  contributions_window_days: 90
  sponsors_fallback:
    - login: alice
haberdasher:
  enabled: true
  config_url: https://example.com/haberdasher.yaml
  sort: release
  hosts:
    - apt.wimpys.world
  logo_overrides:
    wimpysworld/demo: custom.svg
  png_fallbacks:
    wimpysworld/ia-get: assets/ia-get.png
projects:
  - repo: " wimpysworld/nix-config "
    logo: " assets/logo.png "
    language: " Nix "
    homepage: " https://wimpys.world/nix-config "
    description: " NixOS and Home Manager configuration "
  - repo: wimpysworld/nothing-but-nix
feeds:
  podcast:
    enabled: true
    title: Linux Matters
    url: https://linuxmatters.sh/episode/index.xml
    site_url: https://linuxmatters.sh
    cover_url: https://linuxmatters.sh/cover.png
    links:
      - label: " Apple Podcasts "
        slug: " apple-podcasts "
        url: " https://geo.itunes.apple.com/podcast/feed/id1682797246 "
        icon: " apple-podcasts "
      - label: Spotify
        url: https://open.spotify.com/show/388W00MbvHGI69accqEHQK
        icon: spotify
  blog:
    enabled: true
    title: Wimpy's World
    url: https://wimpysworld.com/posts/rss.xml
    max_items: 3
  youtube:
    enabled: false
    title: YouTube
twitch:
  enabled: true
  channel: wimpysworld
discord:
  enabled: true
  name: Wimpy's World
  tagline: Linux & open source community
  url: https://wimpysworld.com/discord
  invite_code: wimpysworld
  guild_id: "1234567890"
  invite_base_url: https://invites.example.test
  widget_base_url: https://widgets.example.test
  shields_base_url: https://shields.example.test
output:
  asset_base_url: https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/abc123
site:
  title: " Wimpy's World "
  description: " Open-source portfolio "
  domain: " wimpys.world "
  default_deck: " PACKAGES "
  links:
    - title: " Matrix "
      slug: " matrix "
      category: " Community "
      description: " Federated chat "
      url: " https://matrix.to/#/@wimpress:matrix.org "
      icon: " matrix "
      accent: " teal "
`
	cfg, err := Parse([]byte(fullYAML))
	if err != nil {
		t.Fatalf("Parse full config failed: %v", err)
	}

	if cfg.Profile.Name != "Martin Wimpress" || len(cfg.Profile.Links) != 1 || cfg.Profile.Links[0].Slug != "website" {
		t.Errorf("unexpected profile config: %+v", cfg.Profile)
	}
	if cfg.GitHub.MaxItems != 8 || len(cfg.GitHub.SponsorsFallback) != 1 || cfg.GitHub.SponsorsFallback[0].URL != "https://github.com/alice" {
		t.Errorf("unexpected github config: %+v", cfg.GitHub)
	}
	if cfg.GitHub.ContributionsWindowDays != 90 {
		t.Errorf("GitHub.ContributionsWindowDays = %d, want 90", cfg.GitHub.ContributionsWindowDays)
	}
	if nonPositiveWindow, err := Parse([]byte("github:\n  contributions_window_days: -30\n")); err != nil {
		t.Fatalf("Parse non-positive contributions_window_days failed: %v", err)
	} else if nonPositiveWindow.GitHub.ContributionsWindowDays != 180 {
		t.Errorf("GitHub.ContributionsWindowDays = %d, want 180", nonPositiveWindow.GitHub.ContributionsWindowDays)
	}
	if !slices.Equal(cfg.GitHub.ExcludeOwners, []string{"flexiondotorg", "wimpysworld"}) {
		t.Errorf("GitHub.ExcludeOwners = %v, want [flexiondotorg wimpysworld]", cfg.GitHub.ExcludeOwners)
	}
	if !cfg.Haberdasher.Enabled || cfg.Haberdasher.Sort != HaberdasherSortRelease || len(cfg.Haberdasher.Hosts) != 1 || cfg.Haberdasher.PNGFallbacks["wimpysworld/ia-get"] != "assets/ia-get.png" {
		t.Errorf("unexpected haberdasher config: %+v", cfg.Haberdasher)
	}
	wantProjects := []ProjectConfig{
		{
			Repo:        "wimpysworld/nix-config",
			Logo:        "assets/logo.png",
			Language:    "Nix",
			Homepage:    "https://wimpys.world/nix-config",
			Description: "NixOS and Home Manager configuration",
		},
		{
			Repo: "wimpysworld/nothing-but-nix",
		},
	}
	if !slices.Equal(cfg.Projects, wantProjects) {
		t.Errorf("Projects = %+v, want %+v", cfg.Projects, wantProjects)
	}
	if starsCfg, err := Parse([]byte("haberdasher:\n  sort: \" STARS \"\n")); err != nil {
		t.Fatalf("Parse haberdasher.sort=STARS failed: %v", err)
	} else if starsCfg.Haberdasher.Sort != HaberdasherSortStars {
		t.Errorf("Haberdasher.Sort = %q, want %q", starsCfg.Haberdasher.Sort, HaberdasherSortStars)
	}
	if cfg.Feeds.Podcast == nil || cfg.Feeds.Podcast.MaxItems != 5 {
		t.Errorf("unexpected podcast feed config: %+v", cfg.Feeds.Podcast)
	}
	wantPodcastLinks := []NavLink{
		{
			Label: "Apple Podcasts",
			Slug:  "apple-podcasts",
			URL:   "https://geo.itunes.apple.com/podcast/feed/id1682797246",
			Icon:  "apple-podcasts",
		},
		{
			Label: "Spotify",
			URL:   "https://open.spotify.com/show/388W00MbvHGI69accqEHQK",
			Icon:  "spotify",
		},
	}
	if cfg.Feeds.Podcast == nil || !slices.Equal(cfg.Feeds.Podcast.Links, wantPodcastLinks) {
		t.Errorf("Feeds.Podcast.Links = %+v, want %+v", cfg.Feeds.Podcast.Links, wantPodcastLinks)
	}
	if cfg.Feeds.Blog == nil || cfg.Feeds.Blog.MaxItems != 3 {
		t.Errorf("unexpected blog feed config: %+v", cfg.Feeds.Blog)
	}
	if cfg.Twitch.URL != "https://www.twitch.tv/wimpysworld" {
		t.Errorf("Twitch.URL = %q", cfg.Twitch.URL)
	}
	wantDiscord := DiscordConfig{
		Enabled:        true,
		Name:           "Wimpy's World",
		Tagline:        "Linux & open source community",
		URL:            "https://wimpysworld.com/discord",
		InviteCode:     "wimpysworld",
		GuildID:        "1234567890",
		InviteBaseURL:  "https://invites.example.test",
		WidgetBaseURL:  "https://widgets.example.test",
		ShieldsBaseURL: "https://shields.example.test",
	}
	if cfg.Discord != wantDiscord {
		t.Errorf("Discord = %+v, want %+v", cfg.Discord, wantDiscord)
	}
	if cfg.Output.AssetBaseURL != "https://raw.githubusercontent.com/flexiondotorg/flexiondotorg/abc123" {
		t.Errorf("Output.AssetBaseURL = %q", cfg.Output.AssetBaseURL)
	}
	wantSiteLinks := []SiteLink{
		{
			Title:       "Matrix",
			Slug:        "matrix",
			Category:    "Community",
			Description: "Federated chat",
			URL:         "https://matrix.to/#/@wimpress:matrix.org",
			Icon:        "matrix",
			Accent:      "teal",
		},
	}
	if cfg.Site.Title != "Wimpy's World" ||
		cfg.Site.Description != "Open-source portfolio" ||
		cfg.Site.Domain != "wimpys.world" ||
		cfg.Site.URL != "https://wimpys.world" ||
		cfg.Site.DefaultDeck != SiteDeckPackages ||
		!slices.Equal(cfg.Site.Links, wantSiteLinks) {
		t.Errorf("unexpected site config: %+v", cfg.Site)
	}

	// Discord enabled with only guild_id or only invite_code is valid when url is set.
	if _, err := Parse([]byte("discord:\n  enabled: true\n  guild_id: \"123\"\n  url: https://discord.gg/abc\n")); err != nil {
		t.Errorf("Parse discord guild_id-only config failed: %v", err)
	}
	if _, err := Parse([]byte("discord:\n  enabled: true\n  invite_code: abc\n  url: https://discord.gg/abc\n")); err != nil {
		t.Errorf("Parse discord invite_code-only config failed: %v", err)
	}

	darkPalette, err := cfg.ResolveDarkPalette()
	if err != nil {
		t.Fatalf("cfg.ResolveDarkPalette failed: %v", err)
	}
	if darkPalette.AccentGreen != "#11aa22" || darkPalette.Base != "#1e1e2e" || darkPalette.IsLight {
		t.Errorf("unexpected darkPalette: %+v", darkPalette)
	}

	lightPalette, err := cfg.ResolveLightPalette()
	if err != nil {
		t.Fatalf("cfg.ResolveLightPalette failed: %v", err)
	}
	if lightPalette.AccentGreen != "#11aa22" || lightPalette.Base != "#eff1f5" || !lightPalette.IsLight {
		t.Errorf("unexpected lightPalette: %+v", lightPalette)
	}

	primaryPalette, err := cfg.ResolvePalette()
	if err != nil {
		t.Fatalf("cfg.ResolvePalette failed: %v", err)
	}
	if primaryPalette != darkPalette {
		t.Errorf("ResolvePalette() != ResolveDarkPalette()")
	}
}

func TestParseValidationErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "invalid yaml",
			yaml:    "profile: [unclosed",
			wantErr: "unmarshal yaml",
		},
		{
			name:    "unknown preset",
			yaml:    "theme:\n  preset: solarized\n",
			wantErr: "unknown theme preset",
		},
		{
			name:    "unknown dark_preset",
			yaml:    "theme:\n  dark_preset: solarized\n",
			wantErr: "unknown theme preset",
		},
		{
			name:    "unknown light_preset",
			yaml:    "theme:\n  light_preset: solarized\n",
			wantErr: "unknown theme preset",
		},
		{
			name:    "invalid hex override",
			yaml:    "theme:\n  overrides:\n    text: not-a-hex\n",
			wantErr: "invalid hex",
		},
		{
			name:    "negative github max_items",
			yaml:    "github:\n  max_items: -1\n",
			wantErr: "max_items",
		},
		{
			name:    "enabled feed missing url",
			yaml:    "feeds:\n  blog:\n    enabled: true\n",
			wantErr: "feeds.blog.url",
		},
		{
			name:    "negative feed max_items",
			yaml:    "feeds:\n  podcast:\n    max_items: -2\n",
			wantErr: "feeds.podcast.max_items",
		},
		{
			name:    "enabled twitch missing channel",
			yaml:    "twitch:\n  enabled: true\n",
			wantErr: "twitch.channel",
		},
		{
			name:    "enabled discord missing invite_code and guild_id",
			yaml:    "discord:\n  enabled: true\n  url: https://discord.gg/abc\n",
			wantErr: "discord.invite_code",
		},
		{
			name:    "enabled discord missing url with invite_code",
			yaml:    "discord:\n  enabled: true\n  invite_code: abc\n",
			wantErr: "discord.url",
		},
		{
			name:    "enabled discord missing url with guild_id",
			yaml:    "discord:\n  enabled: true\n  guild_id: \"123456\"\n",
			wantErr: "discord.url",
		},
		{
			name:    "nav link missing label",
			yaml:    "profile:\n  links:\n    - url: https://example.com\n",
			wantErr: "profile.links[0]: label",
		},
		{
			name:    "nav link missing url",
			yaml:    "profile:\n  links:\n    - label: Site\n",
			wantErr: "profile.links[0]: url",
		},
		{
			name:    "feed link missing label",
			yaml:    "feeds:\n  podcast:\n    enabled: true\n    url: https://example.com/feed.xml\n    links:\n      - url: https://example.com/sub\n",
			wantErr: "feeds.podcast.links[0]: label",
		},
		{
			name:    "feed link missing url",
			yaml:    "feeds:\n  podcast:\n    enabled: true\n    url: https://example.com/feed.xml\n    links:\n      - label: Spotify\n",
			wantErr: "feeds.podcast.links[0]: url",
		},
		{
			name:    "sponsor fallback missing login",
			yaml:    "github:\n  sponsors_fallback:\n    - url: https://github.com/alice\n",
			wantErr: "sponsors_fallback",
		},
		{
			name:    "unknown haberdasher sort",
			yaml:    "haberdasher:\n  sort: name\n",
			wantErr: "haberdasher.sort",
		},
		{
			name:    "project missing repo",
			yaml:    "projects:\n  - language: Nix\n",
			wantErr: "projects[0].repo",
		},
		{
			name:    "project repo missing slash",
			yaml:    "projects:\n  - repo: nix-config\n",
			wantErr: "projects[0].repo",
		},
		{
			name:    "project repo missing owner",
			yaml:    "projects:\n  - repo: /nix-config\n",
			wantErr: "projects[0].repo",
		},
		{
			name:    "project repo missing name",
			yaml:    "projects:\n  - repo: wimpysworld/\n",
			wantErr: "projects[0].repo",
		},
		{
			name:    "project repo extra slash",
			yaml:    "projects:\n  - repo: wimpysworld/nix-config/extra\n",
			wantErr: "projects[0].repo",
		},
		{
			name:    "project repo inner whitespace",
			yaml:    "projects:\n  - repo: wimpysworld /nix-config\n",
			wantErr: "projects[0].repo",
		},
		{
			name:    "unknown site default_deck",
			yaml:    "site:\n  default_deck: archive\n",
			wantErr: "site.default_deck",
		},
		{
			name:    "site link missing title",
			yaml:    "site:\n  links:\n    - url: https://example.com\n",
			wantErr: "site.links[0]: title",
		},
		{
			name:    "site link missing url",
			yaml:    "site:\n  links:\n    - title: Example\n",
			wantErr: "site.links[0]: url",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "vitrine.yaml")
	content := "profile:\n  name: Test User\ngithub:\n  username: testuser\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Profile.Name != "Test User" || cfg.GitHub.Username != "testuser" {
		t.Fatalf("unexpected loaded config: %+v", cfg)
	}

	if _, err := Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
	if _, err := Load(""); err == nil {
		t.Fatal("expected error for empty path")
	}

	var nilCfg *Config
	if _, err := nilCfg.ResolvePalette(); err == nil {
		t.Fatal("expected error for nil Config.ResolvePalette")
	}
	if _, err := nilCfg.ResolveDarkPalette(); err == nil {
		t.Fatal("expected error for nil Config.ResolveDarkPalette")
	}
	if _, err := nilCfg.ResolveLightPalette(); err == nil {
		t.Fatal("expected error for nil Config.ResolveLightPalette")
	}
}

func TestLoadRepositoryVitrineYAML(t *testing.T) {
	t.Parallel()

	cfg, err := Load(filepath.Join("..", "..", "vitrine.yaml"))
	if err != nil {
		t.Fatalf("Load repository vitrine.yaml failed: %v", err)
	}
	if cfg.Site.Domain != "wimpys.world" || cfg.Site.DefaultDeck != SiteDeckProjects {
		t.Fatalf("unexpected repository site config: %+v", cfg.Site)
	}
	if len(cfg.Site.Links) != 13 {
		t.Fatalf("len(Site.Links) = %d, want 13", len(cfg.Site.Links))
	}
}
