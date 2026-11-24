// Package svg renders self-contained animated SVG tiles for Vitrine GitHub
// profile READMEs using the Catppuccin Mocha (dark) and Catppuccin Latte
// (light) visual system.
package svg

import "strings"

// Palette defines the Catppuccin colour tokens used across all rendered SVG
// tiles.
type Palette struct {
	Base            string
	Mantle          string
	Surface0        string
	Surface1        string
	Text            string
	Subtext         string
	Muted           string
	AccentPrimary   string
	AccentSecondary string
	AccentTertiary  string
	AccentPink      string
	AccentGreen     string
	AccentRed       string
	AccentPeach     string
	AccentYellow    string
	AccentSapphire  string
	IsLight         bool
}

// CatppuccinMochaPalette returns the official Catppuccin Mocha (dark) palette.
func CatppuccinMochaPalette() Palette {
	return Palette{
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
}

// CatppuccinLattePalette returns the official Catppuccin Latte (light) palette.
func CatppuccinLattePalette() Palette {
	return Palette{
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
}

// PaletteByName resolves a built-in palette preset name, falling back to
// CatppuccinMochaPalette for unknown or empty names.
func PaletteByName(name string) Palette {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "catppuccin-latte", "latte":
		return CatppuccinLattePalette()
	default:
		return CatppuccinMochaPalette()
	}
}

// normalized fills any missing colour fields with Catppuccin Mocha (or Latte
// when IsLight is true) defaults.
func (p Palette) normalized() Palette {
	def := CatppuccinMochaPalette()
	if p.IsLight {
		def = CatppuccinLattePalette()
	}
	p.Base = fallbackString(p.Base, def.Base)
	p.Mantle = fallbackString(p.Mantle, def.Mantle)
	p.Surface0 = fallbackString(p.Surface0, def.Surface0)
	p.Surface1 = fallbackString(p.Surface1, def.Surface1)
	p.Text = fallbackString(p.Text, def.Text)
	p.Subtext = fallbackString(p.Subtext, def.Subtext)
	p.Muted = fallbackString(p.Muted, def.Muted)
	p.AccentPrimary = fallbackString(p.AccentPrimary, def.AccentPrimary)
	p.AccentSecondary = fallbackString(p.AccentSecondary, def.AccentSecondary)
	p.AccentTertiary = fallbackString(p.AccentTertiary, def.AccentTertiary)
	p.AccentPink = fallbackString(p.AccentPink, def.AccentPink)
	p.AccentGreen = fallbackString(p.AccentGreen, def.AccentGreen)
	p.AccentRed = fallbackString(p.AccentRed, def.AccentRed)
	p.AccentPeach = fallbackString(p.AccentPeach, def.AccentPeach)
	p.AccentYellow = fallbackString(p.AccentYellow, def.AccentYellow)
	p.AccentSapphire = fallbackString(p.AccentSapphire, def.AccentSapphire)
	return p
}

// Renderer generates modular animated SVG tiles styled with a Palette.
type Renderer struct {
	palette    Palette
	darkCanvas bool
}

// NewRenderer creates a Renderer using the supplied Palette, inferring the
// canvas mode from !palette.IsLight and filling any empty tokens from defaults.
func NewRenderer(palette Palette) *Renderer {
	return NewRendererWithMode(palette, !palette.IsLight)
}

// NewRendererWithMode creates a Renderer using the supplied Palette and target
// GitHub canvas mode (darkCanvas true targets dark GitHub pages).
func NewRendererWithMode(palette Palette, darkCanvas bool) *Renderer {
	return &Renderer{
		palette:    palette.normalized(),
		darkCanvas: darkCanvas,
	}
}

// Palette returns the normalized colour palette active on the Renderer.
func (r *Renderer) Palette() Palette {
	return r.palette
}

// DarkCanvas reports whether the Renderer targets a dark GitHub background.
func (r *Renderer) DarkCanvas() bool {
	return r.darkCanvas
}

// HeroData configures the full-width hero banner tile.
type HeroData struct {
	Eyebrow  string
	Title    string
	Subtitle string
	CTA      string
}

// NavPillData configures a navigation pill tile.
type NavPillData struct {
	Label        string
	Icon         string
	DelaySeconds float64
}

// ProjectCardData configures a Haberdasher project card tile.
type ProjectCardData struct {
	Slug         string
	Name         string
	Tagline      string
	Version      string
	Fresh        bool
	Language     string
	Stars        int
	ReleaseAge   string
	PackageCount int
	PackageTotal int
	LogoSVG      string
	LogoDataURI  string
	DelaySeconds float64
}

// PackageHostData configures a single Haberdasher package repository tile.
type PackageHostData struct {
	Name         string
	Host         string
	PackageCount int
	LogoSVG      string
	LogoDataURI  string
	DelaySeconds float64
}

// StatMetric holds a single metric column inside the stats strip tile.
type StatMetric struct {
	Label string
	Value string
}

// StatItem is a backwards-compatible alias for StatMetric.
type StatItem = StatMetric

// StatsData configures the full-width telemetry stats strip tile.
type StatsData struct {
	Metrics []StatMetric
	Items   []StatItem
}

// SectionData configures a section divider banner.
type SectionData struct {
	Title    string
	Subtitle string
}

// DiscordData configures the full-width dynamic Discord community banner tile.
type DiscordData struct {
	Name    string
	Tagline string
	CTA     string
	Online  int
	Members int
}

// TwitchData configures the live or offline Twitch status badge tile.
type TwitchData struct {
	Live    bool
	Uptime  string
	Channel string
}

// SponsorsData configures the full-width animated GitHub Sponsors banner tile.
type SponsorsData struct {
	Active      []string
	Past        []string
	Headline    string
	Subheadline string
	CTA         string
}
