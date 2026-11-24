// Package render executes Vitrine Markdown templates with relative time,
// number, and asset URL helpers.
package render

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
)

// NavPillView pairs a configured navigation link with its rendered SVG filenames.
type NavPillView struct {
	Label     string
	Slug      string
	URL       string
	Icon      string
	BaseName  string
	LightFile string
	DarkFile  string
	File      string
}

// DiscordView pairs live Discord server counts with its rendered SVG filenames.
type DiscordView struct {
	Enabled   bool
	Name      string
	Tagline   string
	URL       string
	Online    int
	Members   int
	BaseName  string
	LightFile string
	DarkFile  string
	File      string
}

// ProjectView pairs a Haberdasher project with its rendered SVG card filenames.
type ProjectView struct {
	FullName    string
	Name        string
	Slug        string
	Description string
	Homepage    string
	RepoURL     string
	URL         string
	Language    string
	Stars       int
	LatestTag   string
	ReleaseURL  string
	PublishedAt time.Time
	Hosts       []string
	BaseName    string
	LightFile   string
	DarkFile    string
	File        string
}

// PackageHostView pairs a Haberdasher package repository host with its tile filenames.
type PackageHostView struct {
	Name         string
	Host         string
	URL          string
	PackageCount int
	BaseName     string
	LightFile    string
	DarkFile     string
	File         string
}

// FeedView groups a configured RSS or Atom feed with its fetched entries.
type FeedView struct {
	Enabled  bool
	Title    string
	URL      string
	SiteURL  string
	CoverURL string
	Items    []rss.Item
}

// TwitchView pairs Twitch live or offline status with its rendered SVG tile filenames.
type TwitchView struct {
	Enabled   bool
	Channel   string
	Live      bool
	Uptime    string
	URL       string
	Alt       string
	BaseName  string
	LightFile string
	DarkFile  string
	File      string
}

// SponsorsView pairs GitHub Sponsors link details with its rendered SVG banner filenames.
type SponsorsView struct {
	Enabled   bool
	URL       string
	Alt       string
	BaseName  string
	LightFile string
	DarkFile  string
	File      string
}

// RenderData carries all profile configuration, fetched data, and rendered SVG
// filenames passed into the Markdown template.
//
//nolint:revive // Exported as RenderData to match the Vitrine template contract.
type RenderData struct {
	Config        *config.Config
	GeneratedAt   time.Time
	AssetBaseURL  string
	HeroBase      string
	HeroFile      string
	NavPills      []NavPillView
	Discord       *DiscordView
	Sponsors      *SponsorsView
	Projects      []ProjectView
	PackageHosts  []PackageHostView
	StatsBase     string
	StatsFile     string
	SectionBases  map[string]string
	SectionFiles  map[string]string
	VideosHeading string
	Twitch        *TwitchView
	GitHub        *gh.ProfileData
	Podcast       *FeedView
	Blog          *FeedView
	YouTube       *FeedView
}

// File reads a template from tmplPath and executes it with data.
func File(tmplPath string, data RenderData) ([]byte, error) {
	cleaned := filepath.Clean(strings.TrimSpace(tmplPath))
	if cleaned == "" || cleaned == "." {
		return nil, errors.New("template path must not be empty")
	}

	// #nosec G304 -- template file path is explicitly provided by the caller.
	raw, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", tmplPath, err)
	}
	return Template(string(raw), data)
}

// Template parses and executes tmplText using Vitrine template helpers.
func Template(tmplText string, data RenderData) ([]byte, error) {
	now := data.GeneratedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	funcs := template.FuncMap{
		"assetURL": func(parts ...string) string {
			switch len(parts) {
			case 0:
				return ""
			case 1:
				return JoinAssetURL(data.AssetBaseURL, parts[0])
			default:
				return JoinAssetURL(parts[0], strings.Join(parts[1:], "/"))
			}
		},
		"humanize": func(t time.Time) string {
			return HumanizeTime(t, now)
		},
		"pictureTag": PictureTag,
		"formatInt":  FormatInt,
		"join":       strings.Join,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
	}

	tmpl, err := template.New("vitrine").Funcs(funcs).Parse(tmplText)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	out := strings.TrimSpace(buf.String()) + "\n"
	return []byte(out), nil
}

// PictureTag renders a GitHub-compatible <picture> element that serves
// <baseName>-dark.svg for dark colour schemes and <baseName>-light.svg for
// light colour schemes. The width attribute is omitted when width <= 0.
func PictureTag(baseURL, baseName, alt string, width int) string {
	base := strings.TrimSuffix(strings.TrimSpace(baseName), ".svg")
	if base == "" {
		return ""
	}
	darkSrc := JoinAssetURL(baseURL, base+"-dark.svg")
	lightSrc := JoinAssetURL(baseURL, base+"-light.svg")

	var b strings.Builder
	b.WriteString(`<picture><source media="(prefers-color-scheme: dark)" srcset="`)
	b.WriteString(html.EscapeString(darkSrc))
	b.WriteString(`"><img src="`)
	b.WriteString(html.EscapeString(lightSrc))
	b.WriteString(`" alt="`)
	b.WriteString(html.EscapeString(alt))
	b.WriteByte('"')
	if width > 0 {
		fmt.Fprintf(&b, ` width="%d"`, width)
	}
	b.WriteString(`></picture>`)
	return b.String()
}

// JoinAssetURL joins a base asset URL or directory path with a relative asset filename.
func JoinAssetURL(base, file string) string {
	f := strings.TrimSpace(file)
	if f == "" {
		return ""
	}
	if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") || strings.HasPrefix(f, "data:") {
		return f
	}
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	f = strings.TrimLeft(f, "/")
	if b == "" {
		return f
	}
	return b + "/" + f
}

// HumanizeTime formats t as a clean relative age compared to now.
func HumanizeTime(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	diff := now.Sub(t)
	if diff < 24*time.Hour {
		return "today"
	}

	days := int(diff / (24 * time.Hour))
	switch {
	case days == 1:
		return "1 day ago"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case days < 30:
		weeks := days / 7
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case days < 365:
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := days / 365
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// FormatCompactAge formats t as a compact relative age suited to tight SVG cards.
func FormatCompactAge(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	diff := now.Sub(t)
	if diff < 24*time.Hour {
		return "today"
	}

	days := int(diff / (24 * time.Hour))
	switch {
	case days < 7:
		return fmt.Sprintf("%dd ago", days)
	case days < 30:
		return fmt.Sprintf("%dw ago", days/7)
	case days < 365:
		return fmt.Sprintf("%dmo ago", days/30)
	default:
		return fmt.Sprintf("%dy ago", days/365)
	}
}

// FormatInt formats an integer with comma thousands separators.
func FormatInt(n int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return sign + s
	}

	var b strings.Builder
	b.WriteString(sign)
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		b.WriteByte(',')
	}
	for i := rem; i < len(s); i += 3 {
		if i > rem {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
