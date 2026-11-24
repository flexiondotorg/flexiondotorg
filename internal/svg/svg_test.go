package svg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	reAnyHTTPURL = regexp.MustCompile(`(?i)https?://[^\s"'<>]+`)
	reRootDims   = regexp.MustCompile(`<svg[^>]+width="([0-9]+)"[^>]+height="([0-9]+)"`)
)

// assertValidGitHubSVG verifies XML well-formedness, standalone SVG root
// attributes, reduced-motion media query, and strict GitHub <img> SVG rules.
func assertValidGitHubSVG(t *testing.T, name string, raw []byte) {
	t.Helper()

	s := string(raw)
	if len(raw) == 0 {
		t.Fatalf("%s: rendered empty SVG", name)
	}

	// 1. Parse entire SVG strictly with encoding/xml to guarantee well-formed XML.
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var rootFound bool
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("%s: invalid XML: %v\n%s", name, err, s)
		}
		if start, ok := tok.(xml.StartElement); ok && !rootFound {
			rootFound = true
			if start.Name.Local != "svg" {
				t.Fatalf("%s: root element is <%s>, want <svg>", name, start.Name.Local)
			}
			if start.Name.Space != "http://www.w3.org/2000/svg" {
				t.Fatalf("%s: root xmlns is %q, want http://www.w3.org/2000/svg", name, start.Name.Space)
			}
			attrs := make(map[string]string, len(start.Attr))
			for _, a := range start.Attr {
				attrs[a.Name.Local] = a.Value
			}
			for _, required := range []string{"width", "height", "viewBox"} {
				if strings.TrimSpace(attrs[required]) == "" {
					t.Errorf("%s: missing required root attribute %q", name, required)
				}
			}
		}
	}
	if !rootFound {
		t.Fatalf("%s: no XML elements found", name)
	}

	// 2. Enforce strict GitHub <img> SVG prohibitions.
	lower := strings.ToLower(s)
	for _, forbidden := range []string{"<script", "<a ", ":hover"} {
		if strings.Contains(lower, forbidden) {
			t.Errorf("%s: contains forbidden pattern %q", name, forbidden)
		}
	}

	// 3. No external http:// or https:// URLs outside standard SVG XML namespaces.
	for _, match := range reAnyHTTPURL.FindAllString(s, -1) {
		if match != "http://www.w3.org/2000/svg" && match != "http://www.w3.org/1999/xlink" {
			t.Errorf("%s: contains disallowed external URL %q", name, match)
		}
	}

	// 4. Every animated tile must honour prefers-reduced-motion.
	if !strings.Contains(s, "@media (prefers-reduced-motion: reduce)") {
		t.Errorf("%s: missing prefers-reduced-motion media query", name)
	}
}

func TestRenderAllTilesStrictCompliance(t *testing.T) {
	t.Parallel()

	sampleLogo := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
  <defs>
    <linearGradient id="accent"><stop offset="0%" stop-color="#cba6f7"/><stop offset="100%" stop-color="#89b4fa"/></linearGradient>
  </defs>
  <script>alert('xss')</script>
  <a href="https://evil.example"><circle cx="32" cy="32" r="24" fill="url(#accent)"/></a>
</svg>`

	pngDataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

	palettes := []struct {
		name string
		p    Palette
	}{
		{"catppuccin-mocha", CatppuccinMochaPalette()},
		{"catppuccin-latte", CatppuccinLattePalette()},
		{"zero-defaults", Palette{}},
	}

	for _, tc := range palettes {
		r := NewRenderer(tc.p)

		hero, err := r.RenderHero(HeroData{
			Eyebrow:  "THE VITRINE · MARTIN WIMPRESS",
			Title:    "WIMPY'S WORLD",
			Subtitle: "Open-source tools, Linux desktop craft & audio utilities crafted in Go & Rust.",
			CTA:      "wimpys.world ↗",
		})
		if err != nil {
			t.Fatalf("%s RenderHero error: %v", tc.name, err)
		}
		assertValidGitHubSVG(t, tc.name+"/hero", hero)

		icons := []string{"globe", "blog", "mastodon", "bluesky", "linkedin", "discord", "github", "heart", "podcast", "twitch", "youtube", "unknown"}
		for i, icon := range icons {
			pill, err := r.RenderNavPill(NavPillData{
				Label:        icon,
				Icon:         icon,
				DelaySeconds: float64(i) * 0.35,
			})
			if err != nil {
				t.Fatalf("%s RenderNavPill(%s) error: %v", tc.name, icon, err)
			}
			assertValidGitHubSVG(t, tc.name+"/nav-"+icon, pill)
		}

		// Project cards: inline SVG logo, data URI logo, Jive placeholder, monogram placeholder.
		cards := []struct {
			label string
			data  ProjectCardData
		}{
			{
				label: "inline-svg-fresh",
				data: ProjectCardData{
					Slug:         "quickemu",
					Name:         "Quickemu & Friends",
					Tagline:      "Quickly create and run optimised Windows, macOS & Linux virtual machines.",
					Version:      "v4.9.7",
					Fresh:        true,
					Language:     "Shell",
					Stars:        12840,
					ReleaseAge:   "today",
					LogoSVG:      sampleLogo,
					DelaySeconds: 0.4,
				},
			},
			{
				label: "data-uri",
				data: ProjectCardData{
					Slug:         "ia-get",
					Name:         "ia-get",
					Tagline:      "Fast resilient downloader for Internet Archive collections.",
					Version:      "v0.2.1",
					Fresh:        false,
					Language:     "Go",
					Stars:        810,
					ReleaseAge:   "1 week ago",
					LogoDataURI:  pngDataURI,
					DelaySeconds: 0.8,
				},
			},
			{
				label: "jive-waveform",
				data: ProjectCardData{
					Slug:         "jive-tuner",
					Name:         "jive-tuner",
					Tagline:      "Broadcast audio chain & voice dynamics toolkit for Linux Matters.",
					Version:      "v0.1.0",
					Fresh:        true,
					Language:     "Go",
					Stars:        42,
					ReleaseAge:   "1d ago",
					DelaySeconds: 1.2,
				},
			},
			{
				label: "monogram-fallback",
				data: ProjectCardData{
					Slug:         "tailor",
					Name:         "Tailor",
					Tagline:      "Fit, baste, alter & measure community health files across repositories.",
					Version:      "",
					Fresh:        false,
					Language:     "Go",
					Stars:        19,
					ReleaseAge:   "3 weeks ago",
					DelaySeconds: 1.6,
				},
			},
		}

		for _, c := range cards {
			out, err := r.RenderProjectCard(c.data)
			if err != nil {
				t.Fatalf("%s RenderProjectCard(%s) error: %v", tc.name, c.label, err)
			}
			assertValidGitHubSVG(t, tc.name+"/project-"+c.label, out)
		}

		hosts := []PackageHostData{
			{Name: "APT", Host: "apt.wimpys.world", PackageCount: 9, LogoSVG: sampleLogo, DelaySeconds: 0.25},
			{Name: "FLATPAK", Host: "flatpak.wimpys.world", PackageCount: 1, LogoDataURI: pngDataURI, DelaySeconds: 0.50},
			{Name: "NIX", Host: "nix.wimpys.world", PackageCount: 12, DelaySeconds: 0.75},
		}
		for _, h := range hosts {
			out, err := r.RenderPackageHost(h)
			if err != nil {
				t.Fatalf("%s RenderPackageHost(%s) error: %v", tc.name, h.Name, err)
			}
			assertValidGitHubSVG(t, tc.name+"/host-"+h.Name, out)
		}

		stats, err := r.RenderStatsStrip(StatsData{
			Metrics: []StatMetric{
				{Label: "STARS", Value: "24,810"},
				{Label: "COMMITS", Value: "18,420"},
				{Label: "PULL REQUESTS", Value: "1,402"},
				{Label: "ISSUES", Value: "684"},
				{Label: "FOLLOWERS", Value: "3,920"},
			},
		})
		if err != nil {
			t.Fatalf("%s RenderStatsStrip error: %v", tc.name, err)
		}
		assertValidGitHubSVG(t, tc.name+"/stats", stats)

		discord, err := r.RenderDiscordBanner(DiscordData{
			Name:    "Wimpy's World Discord",
			Tagline: "Linux desktop craft, Quickemu, Ubuntu MATE & audio chat",
			CTA:     "JOIN ↗",
			Online:  169,
			Members: 2857,
		})
		if err != nil {
			t.Fatalf("%s RenderDiscordBanner error: %v", tc.name, err)
		}
		assertValidGitHubSVG(t, tc.name+"/discord", discord)

		divider, err := r.RenderSectionDivider(SectionData{
			Title:    "CURATED PROJECTS",
			Subtitle: "SORTED BY LATEST RELEASE",
		})
		if err != nil {
			t.Fatalf("%s RenderSectionDivider error: %v", tc.name, err)
		}
		assertValidGitHubSVG(t, tc.name+"/divider", divider)

		liveTwitch, err := r.RenderTwitchStatus(TwitchData{Live: true, Uptime: "2h 14m", Channel: "wimpysworld"})
		if err != nil {
			t.Fatalf("%s RenderTwitchStatus(live) error: %v", tc.name, err)
		}
		assertValidGitHubSVG(t, tc.name+"/twitch-live", liveTwitch)

		offTwitch, err := r.RenderTwitchStatus(TwitchData{Live: false, Channel: "wimpysworld"})
		if err != nil {
			t.Fatalf("%s RenderTwitchStatus(offline) error: %v", tc.name, err)
		}
		assertValidGitHubSVG(t, tc.name+"/twitch-offline", offTwitch)
	}
}

func TestCatppuccinMochaAndLattePalettesAndFonts(t *testing.T) {
	t.Parallel()

	mocha := CatppuccinMochaPalette()
	latte := CatppuccinLattePalette()

	if mocha.IsLight {
		t.Error("CatppuccinMochaPalette().IsLight must be false")
	}
	if !latte.IsLight {
		t.Error("CatppuccinLattePalette().IsLight must be true")
	}
	if mocha.Base != "#1e1e2e" || mocha.AccentPrimary != "#cba6f7" || mocha.AccentSecondary != "#89b4fa" || mocha.AccentTertiary != "#94e2d5" {
		t.Errorf("unexpected Mocha palette values: %+v", mocha)
	}
	if latte.Base != "#eff1f5" || latte.AccentPrimary != "#8839ef" || latte.AccentSecondary != "#1e66f5" || latte.AccentTertiary != "#179299" {
		t.Errorf("unexpected Latte palette values: %+v", latte)
	}

	mochaR := NewRenderer(mocha)
	latteR := NewRenderer(latte)
	if !mochaR.DarkCanvas() {
		t.Error("NewRenderer(Mocha) must set DarkCanvas() to true")
	}
	if latteR.DarkCanvas() {
		t.Error("NewRenderer(Latte) must set DarkCanvas() to false")
	}

	renderFamily := func(r *Renderer) map[string]string {
		out := make(map[string]string)
		b, _ := r.RenderHero(HeroData{Title: "WIMPY'S WORLD", Subtitle: "Open-source tools & Linux craft.", CTA: "wimpys.world ↗"})
		out["hero"] = string(b)
		b, _ = r.RenderNavPill(NavPillData{Label: "Mastodon", Icon: "mastodon", DelaySeconds: 0.4})
		out["nav"] = string(b)
		b, _ = r.RenderProjectCard(ProjectCardData{Slug: "ia-get", Name: "ia-get", Tagline: "Fast downloader.", Version: "v0.2.1", Fresh: true, Language: "Go", Stars: 810, ReleaseAge: "today", DelaySeconds: 0.6})
		out["project"] = string(b)
		b, _ = r.RenderPackageHost(PackageHostData{Name: "APT", Host: "apt.wimpys.world", PackageCount: 9, DelaySeconds: 0.3})
		out["host"] = string(b)
		b, _ = r.RenderStatsStrip(StatsData{})
		out["stats"] = string(b)
		b, _ = r.RenderDiscordBanner(DiscordData{Online: 169, Members: 2857})
		out["discord"] = string(b)
		b, _ = r.RenderSectionDivider(SectionData{Title: "PROJECTS"})
		out["divider"] = string(b)
		b, _ = r.RenderTwitchStatus(TwitchData{Live: true, Uptime: "1h 12m"})
		out["twitch-live"] = string(b)
		b, _ = r.RenderTwitchStatus(TwitchData{Live: false})
		out["twitch-offline"] = string(b)
		return out
	}

	mochaTiles := renderFamily(mochaR)
	latteTiles := renderFamily(latteR)

	for name, mochaSVG := range mochaTiles {
		latteSVG := latteTiles[name]
		if mochaSVG == latteSVG {
			t.Errorf("%s: Mocha and Latte outputs must differ", name)
		}
		for mode, svgText := range map[string]string{"mocha": mochaSVG, "latte": latteSVG} {
			if strings.Contains(svgText, "Georgia, serif") || strings.Contains(svgText, "Iowan Old Style") {
				t.Errorf("%s/%s: must not use serif fonts", mode, name)
			}
			if !strings.Contains(svgText, "'Work Sans'") {
				t.Errorf("%s/%s: missing Work Sans font stack", mode, name)
			}
			if !strings.Contains(svgText, `stroke-width="1.5"`) {
				t.Errorf("%s/%s: missing single 1.5px border", mode, name)
			}
			if strings.Contains(svgText, "<polygon") {
				t.Errorf("%s/%s: must not contain polygon logo frames", mode, name)
			}
		}
		if !strings.Contains(mochaSVG, `stop-opacity="0.34"`) {
			t.Errorf("mocha/%s: missing Mocha sheen peak 0.34", name)
		}
		if !strings.Contains(latteSVG, `stop-opacity="0.65"`) {
			t.Errorf("latte/%s: missing Latte sheen peak 0.65", name)
		}
	}

	// Verify Fira Code appears in all monospace tiles.
	for _, monoTile := range []string{"hero", "project", "host", "discord", "twitch-live"} {
		if !strings.Contains(mochaTiles[monoTile], "'Fira Code'") {
			t.Errorf("mocha/%s: missing Fira Code font stack", monoTile)
		}
	}
}

func TestHeroAuroraWavesAndTypography(t *testing.T) {
	t.Parallel()

	r := NewRenderer(CatppuccinMochaPalette())

	heroBytes, err := r.RenderHero(HeroData{
		Title:    "WIMPY'S WORLD",
		Subtitle: "Open-source tools, Linux desktop craft & audio utilities crafted in Go & Rust.",
		CTA:      "wimpys.world ↗",
	})
	if err != nil {
		t.Fatalf("RenderHero error: %v", err)
	}
	assertValidGitHubSVG(t, "hero-no-eyebrow", heroBytes)
	heroStr := string(heroBytes)
	for _, want := range []string{
		`width="840" height="210"`,
		`stroke="url(#cat-accent)" stroke-width="1.5"`,
		`font-size="42"`,
		`font-size="16"`,
		`vitrine-hero-glint`,
		`fill="url(#cat-glint)"`,
		`vitrine-aurora-waves`,
		`vitrine-wave-flow-1`,
		`vitrine-wave-flow-2`,
		`vitrine-wave-flow-3`,
		`vitrine-dot-5`,
		`vitrine-hero-cta`,
		`vitrine-cta-sheen`,
		`fill="url(#cat-sheen)"`,
		`wimpys.world ↗`,
	} {
		if !strings.Contains(heroStr, want) {
			t.Errorf("RenderHero missing %q", want)
		}
	}

	// Empty CTA omits pill cleanly.
	noCTA, err := r.RenderHero(HeroData{
		Title:    "WIMPY'S WORLD",
		Subtitle: "Open-source tools & Linux desktop craft.",
	})
	if err != nil {
		t.Fatalf("RenderHero(no-cta) error: %v", err)
	}
	assertValidGitHubSVG(t, "hero-no-cta", noCTA)
	if strings.Contains(string(noCTA), "vitrine-hero-cta") {
		t.Errorf("RenderHero with empty CTA must omit CTA pill")
	}

	// Explicit eyebrow renders eyebrow above 36px title.
	withEyebrow, err := r.RenderHero(HeroData{
		Eyebrow:  "OPEN SOURCE STUDIO",
		Title:    "VITRINE",
		Subtitle: "Reusable SVG cards.",
		CTA:      "example.org",
	})
	if err != nil {
		t.Fatalf("RenderHero(eyebrow) error: %v", err)
	}
	assertValidGitHubSVG(t, "hero-with-eyebrow", withEyebrow)
	if !strings.Contains(string(withEyebrow), "OPEN SOURCE STUDIO") || !strings.Contains(string(withEyebrow), `font-size="36"`) {
		t.Errorf("RenderHero with explicit eyebrow must render eyebrow and 36px title")
	}
}

func TestNavPillsAndProjectAndHostDetails(t *testing.T) {
	t.Parallel()

	p := CatppuccinMochaPalette()
	r := NewRenderer(p)

	// 1. Nav pills: platform accent border + icon, coupled icon/text (textX - iconX == 30), width >= 168.
	brandChecks := []struct {
		label      string
		icon       string
		accent     string
		pathPrefix string
		delay      float64
	}{
		{"Mastodon", "mastodon", p.AccentPrimary, "M23.268 5.313", 0.35},
		{"Bluesky", "bluesky", p.AccentSecondary, "M5.202 2.857", 0.70},
		{"LinkedIn", "linkedin", p.AccentSapphire, "M20.447 20.452", 1.05},
	}
	for _, bc := range brandChecks {
		pillBytes, err := r.RenderNavPill(NavPillData{Label: bc.label, Icon: bc.icon, DelaySeconds: bc.delay})
		if err != nil {
			t.Fatalf("RenderNavPill(%s) error: %v", bc.label, err)
		}
		assertValidGitHubSVG(t, "nav-brand-"+bc.icon, pillBytes)
		pillStr := string(pillBytes)
		m := reRootDims.FindStringSubmatch(pillStr)
		if len(m) != 3 {
			t.Fatalf("RenderNavPill(%s): missing root width/height", bc.label)
		}
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		if h != 44 {
			t.Errorf("RenderNavPill(%s): height = %d, want 44", bc.label, h)
		}
		if w < 168 {
			t.Errorf("RenderNavPill(%s): width = %d, want >= 168", bc.label, w)
		}
		if !strings.Contains(pillStr, fmt.Sprintf(`stroke="%s" stroke-width="1.5"`, bc.accent)) {
			t.Errorf("RenderNavPill(%s): missing 1.5px platform accent border %s", bc.label, bc.accent)
		}
		if !strings.Contains(pillStr, bc.pathPrefix) {
			t.Errorf("RenderNavPill(%s): missing authentic brand path %q", bc.label, bc.pathPrefix)
		}
		if !strings.Contains(pillStr, fmt.Sprintf("4.4s ease-in-out %ss infinite", formatFloat(bc.delay))) {
			t.Errorf("RenderNavPill(%s): missing staggered icon breathe delay %v", bc.label, bc.delay)
		}
		for _, forbidden := range []string{"url(#cat-sheen)", "skewX", "vitrine-pill-glow", "vitrine-pill-sheen"} {
			if strings.Contains(pillStr, forbidden) {
				t.Errorf("RenderNavPill(%s): must not contain diagonal face sheen %q", bc.label, forbidden)
			}
		}

		// Parse XML to verify iconX, textX coupling (textX - iconX == 30).
		var iconX, textX float64
		var foundIcon, foundText bool
		var textAnchor string
		dec := xml.NewDecoder(bytes.NewReader(pillBytes))
		rootSeen := false
		for {
			tok, err := dec.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("RenderNavPill(%s): XML parse error: %v", bc.label, err)
			}
			start, ok := tok.(xml.StartElement)
			if !ok {
				continue
			}
			attrs := make(map[string]string, len(start.Attr))
			for _, a := range start.Attr {
				attrs[a.Name.Local] = a.Value
			}
			switch start.Name.Local {
			case "svg":
				if !rootSeen {
					rootSeen = true
					continue
				}
				iconX, _ = strconv.ParseFloat(attrs["x"], 64)
				foundIcon = true
			case "text":
				textX, _ = strconv.ParseFloat(attrs["x"], 64)
				textAnchor = attrs["text-anchor"]
				foundText = true
			}
		}
		if !foundIcon || !foundText {
			t.Fatalf("RenderNavPill(%s): missing nested icon <svg> (%v) or <text> (%v)", bc.label, foundIcon, foundText)
		}
		if textAnchor != "start" {
			t.Errorf("RenderNavPill(%s): text-anchor = %q, want \"start\"", bc.label, textAnchor)
		}
		if diff := math.Abs((textX - iconX) - 30.0); diff > 0.01 {
			t.Errorf("RenderNavPill(%s): textX - iconX = %.2f, want 30.00", bc.label, textX-iconX)
		}
	}

	// 2. Project card: 408x182, unframed 64x64 logo at (20,34), 4-line wrapped description, top-edge glint, 3-part footer.
	sampleLogo := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><circle cx="32" cy="32" r="28" fill="#89b4fa"/></svg>`
	longTagline := "Create a Nix store of up to 129GiB in GitHub runners with a multi-disk BTRFS pool for /nix that purges unused tools and dynamically grows the Nix store."
	cardBytes, err := r.RenderProjectCard(ProjectCardData{
		Slug:         "ia-get",
		Name:         "ia-get",
		Tagline:      longTagline,
		Version:      "v0.2.1",
		Fresh:        true,
		Language:     "Go",
		Stars:        810,
		ReleaseAge:   "1 week ago",
		LogoSVG:      sampleLogo,
		DelaySeconds: 1.25,
	})
	if err != nil {
		t.Fatalf("RenderProjectCard error: %v", err)
	}
	assertValidGitHubSVG(t, "project-card-detail", cardBytes)
	cardStr := string(cardBytes)
	for _, want := range []string{
		`width="408" height="182" viewBox="0 0 408 182"`,
		`x="20" y="34" width="64" height="64"`,
		`vitrine-card-glint`,
		`fill="url(#cat-glint)" class="vitrine-card-glint"`,
		`6.5s ease-in-out 1.25s infinite`,
		`<text x="100" y="68"`,
		`<text x="100" y="86"`,
		`<text x="100" y="104"`,
		`<text x="100" y="122"`,
		`y1="142" x2="388" y2="142"`,
		`cy="159"`,
		`y="163"`,
		"v0.2.1",
		"vitrine-fresh-ring",
		">Go<",
		fmt.Sprintf(`<tspan fill="%s">★</tspan> 810`, p.AccentYellow),
		"1 week ago",
		"Nix store.",
	} {
		if !strings.Contains(cardStr, want) {
			t.Errorf("RenderProjectCard missing %q", want)
		}
	}
	for _, forbidden := range []string{"<polygon", "url(#cat-sheen)", "skewX", "vitrine-card-sheen", "…"} {
		if strings.Contains(cardStr, forbidden) {
			t.Errorf("RenderProjectCard must not contain %q", forbidden)
		}
	}

	// Verify CalVer tags up to 18 characters (such as v26.5.20261113) render untruncated.
	calverBytes, err := r.RenderProjectCard(ProjectCardData{
		Slug:       "nix-config",
		Name:       "nix-config",
		Tagline:    "Wimpy's NixOS, nix-darwin & Home Manager Configurations",
		Version:    "v26.5.20261113",
		Fresh:      true,
		Language:   "Nix",
		Stars:      734,
		ReleaseAge: "5 days ago",
	})
	if err != nil {
		t.Fatalf("RenderProjectCard(calver) error: %v", err)
	}
	assertValidGitHubSVG(t, "project-card-calver", calverBytes)
	calverStr := string(calverBytes)
	if !strings.Contains(calverStr, ">v26.5.20261113<") || strings.Contains(calverStr, "…") {
		t.Errorf("RenderProjectCard(calver) truncated CalVer tag: %s", calverStr)
	}

	// 3. Package host: 198x78, unframed 40x40 logo, top-edge glint, Surface0 count pill in AccentPrimary.
	hostBytes, err := r.RenderPackageHost(PackageHostData{
		Name:         "APT",
		Host:         "apt.wimpys.world",
		PackageCount: 9,
		LogoSVG:      sampleLogo,
		DelaySeconds: 0.9,
	})
	if err != nil {
		t.Fatalf("RenderPackageHost error: %v", err)
	}
	assertValidGitHubSVG(t, "package-host-detail", hostBytes)
	hostStr := string(hostBytes)
	for _, want := range []string{
		`width="198" height="78"`,
		`x="14" y="20" width="40" height="40"`,
		`vitrine-host-glint`,
		`fill="url(#cat-glint)" class="vitrine-host-glint"`,
		`6.5s ease-in-out 0.90s infinite`,
		fmt.Sprintf(`fill="%s"`, p.Surface0),
		fmt.Sprintf(`fill="%s">9 pkgs</text>`, p.AccentPrimary),
	} {
		if !strings.Contains(hostStr, want) {
			t.Errorf("RenderPackageHost missing %q", want)
		}
	}
	for _, forbidden := range []string{"<polygon", "url(#cat-sheen)", "skewX", "vitrine-host-sheen"} {
		if strings.Contains(hostStr, forbidden) {
			t.Errorf("RenderPackageHost must not contain %q", forbidden)
		}
	}
}

func TestRenderDiscordBanner(t *testing.T) {
	t.Parallel()

	p := CatppuccinMochaPalette()
	r := NewRenderer(p)

	full, err := r.RenderDiscordBanner(DiscordData{
		Name:    "Wimpy's World Discord",
		Tagline: "Chat Linux, open source & audio",
		CTA:     "JOIN ↗",
		Online:  169,
		Members: 2857,
	})
	if err != nil {
		t.Fatalf("RenderDiscordBanner(full) error: %v", err)
	}
	assertValidGitHubSVG(t, "discord-full", full)
	s := string(full)
	for _, want := range []string{
		`width="840" height="68"`,
		`stroke="url(#cat-accent)" stroke-width="1.5"`,
		`x="24" y="20" width="28" height="28" viewBox="0 0 24 24"`,
		`M20.317 4.3698`,
		fmt.Sprintf(`fill="%s"`, p.AccentGreen),
		"vitrine-discord-ring",
		"169 online",
		"2,857 members",
		"JOIN ↗",
		"Wimpy&#39;s World Discord",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("RenderDiscordBanner(full) missing %q", want)
		}
	}
	for _, forbidden := range []string{"url(#cat-sheen)", "skewX", "vitrine-discord-sheen", "vitrine-discord-sweep"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("RenderDiscordBanner(full) must not contain diagonal face sheen %q", forbidden)
		}
	}

	noMembers, err := r.RenderDiscordBanner(DiscordData{
		Online:  1245,
		Members: 0,
	})
	if err != nil {
		t.Fatalf("RenderDiscordBanner(noMembers) error: %v", err)
	}
	assertValidGitHubSVG(t, "discord-no-members", noMembers)
	ns := string(noMembers)
	if !strings.Contains(ns, "1,245 online") {
		t.Errorf("RenderDiscordBanner(noMembers) missing comma-formatted online count")
	}
	if strings.Contains(ns, "members") {
		t.Errorf("RenderDiscordBanner(noMembers) must omit members when Members == 0")
	}
}

func TestRenderStatsStripFiveMetricsAndCompatibility(t *testing.T) {
	t.Parallel()

	p := CatppuccinMochaPalette()
	r := NewRenderer(p)

	five, err := r.RenderStatsStrip(StatsData{
		Metrics: []StatMetric{
			{Label: "Stars", Value: "24,810"},
			{Label: "Commits", Value: "18,420"},
			{Label: "Pull Requests", Value: "1,402"},
			{Label: "Issues", Value: "684"},
			{Label: "Followers", Value: "3,920"},
		},
	})
	if err != nil {
		t.Fatalf("RenderStatsStrip(5) error: %v", err)
	}
	assertValidGitHubSVG(t, "stats-5", five)
	s := string(five)

	for _, want := range []string{
		"STARS", "COMMITS", "PULL REQUESTS", "ISSUES", "FOLLOWERS",
		"24,810", "18,420", "1,402", "684", "3,920",
		"vitrine-stats-glint",
		`fill="url(#cat-glint)"`,
		"vitrine-stat-dot",
		fmt.Sprintf(`fill="%s"`, p.AccentYellow),
		fmt.Sprintf(`fill="%s"`, p.AccentGreen),
		fmt.Sprintf(`fill="%s"`, p.AccentPrimary),
		fmt.Sprintf(`fill="%s"`, p.AccentPeach),
		fmt.Sprintf(`fill="%s"`, p.AccentSapphire),
	} {
		if !strings.Contains(s, want) {
			t.Errorf("RenderStatsStrip(5) missing %q", want)
		}
	}
	for _, divX := range []string{"168", "336", "504", "672"} {
		wantLine := fmt.Sprintf(`x1="%s" y1="22" x2="%s" y2="76"`, divX, divX)
		if !strings.Contains(s, wantLine) {
			t.Errorf("RenderStatsStrip(5) missing vertical divider at x=%s", divX)
		}
	}

	// Verify backwards compatibility with legacy StatsData.Items field.
	legacy, err := r.RenderStatsStrip(StatsData{
		Items: []StatItem{
			{Label: "REPOSITORIES", Value: "148"},
			{Label: "FOLLOWERS", Value: "3,920"},
		},
	})
	if err != nil {
		t.Fatalf("RenderStatsStrip(legacy) error: %v", err)
	}
	assertValidGitHubSVG(t, "stats-legacy", legacy)
	if !strings.Contains(string(legacy), "REPOSITORIES") || !strings.Contains(string(legacy), "148") {
		t.Errorf("RenderStatsStrip(legacy) failed to render Items slice")
	}
}

func TestRenderSectionDividerZeroOverlap(t *testing.T) {
	t.Parallel()

	r := NewRenderer(CatppuccinMochaPalette())

	cases := []struct {
		title    string
		subtitle string
	}{
		{title: "PODCAST", subtitle: ""},
		{title: "PODCAST", subtitle: "EPISODES"},
		{title: "STARRED", subtitle: ""},
		{title: "RECENT COMMUNITY CONTRIBUTIONS", subtitle: ""},
		{title: "RECENT COMMUNITY CONTRIBUTIONS", subtitle: "GITHUB"},
		{title: "PACKAGE REPOSITORIES", subtitle: "8 FORMATS"},
	}

	for _, tc := range cases {
		raw, err := r.RenderSectionDivider(SectionData{
			Title:    tc.title,
			Subtitle: tc.subtitle,
		})
		if err != nil {
			t.Fatalf("RenderSectionDivider(%q) error: %v", tc.title, err)
		}
		assertValidGitHubSVG(t, "divider-"+tc.title, raw)

		var leftDotX, rightDotX float64
		var foundLeft, foundRight bool
		var leftRuleEnd, rightRuleStart float64
		var foundLeftRule, foundRightRule bool

		if strings.Contains(string(raw), "<path") {
			t.Errorf("%q: RenderSectionDivider must use <circle> pips and zero diamond <path> elements", tc.title)
		}

		dec := xml.NewDecoder(bytes.NewReader(raw))
		for {
			tok, err := dec.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("parse divider XML for %q: %v", tc.title, err)
			}
			start, ok := tok.(xml.StartElement)
			if !ok {
				continue
			}
			attrs := make(map[string]string, len(start.Attr))
			for _, a := range start.Attr {
				attrs[a.Name.Local] = a.Value
			}
			switch start.Name.Local {
			case "circle":
				switch attrs["class"] {
				case "vitrine-divider-dot-left":
					leftDotX, _ = strconv.ParseFloat(attrs["cx"], 64)
					if attrs["cy"] != "21" || attrs["r"] != "3.5" {
						t.Errorf("%q: left divider circle has cy=%q r=%q, want cy=\"21\" r=\"3.5\"", tc.title, attrs["cy"], attrs["r"])
					}
					foundLeft = true
				case "vitrine-divider-dot-right":
					rightDotX, _ = strconv.ParseFloat(attrs["cx"], 64)
					if attrs["cy"] != "21" || attrs["r"] != "3.5" {
						t.Errorf("%q: right divider circle has cy=%q r=%q, want cy=\"21\" r=\"3.5\"", tc.title, attrs["cy"], attrs["r"])
					}
					foundRight = true
				}
			case "line":
				if strings.Contains(attrs["class"], "vitrine-rule-trace") {
					x1, _ := strconv.ParseFloat(attrs["x1"], 64)
					x2, _ := strconv.ParseFloat(attrs["x2"], 64)
					if x1 < 420 {
						leftRuleEnd = x2
						foundLeftRule = true
					} else {
						rightRuleStart = x1
						foundRightRule = true
					}
				}
			}
		}

		if !foundLeft || !foundRight {
			t.Fatalf("%q: missing flanking centre circles (foundLeft=%v, foundRight=%v)", tc.title, foundLeft, foundRight)
		}

		upper := strings.ToUpper(TruncateText(tc.title, 36))
		halfTextMax := float64(len([]rune(upper))) * 11.2 / 2.0
		const dotRadius = 3.5

		leftClearance := (420.0 - halfTextMax) - (leftDotX + dotRadius)
		rightClearance := (rightDotX - dotRadius) - (420.0 + halfTextMax)
		if leftClearance < 24.0 {
			t.Errorf("%q: left circle too close to title: leftDotX=%.2f, halfTextMax=%.2f, clearance=%.2fpx", tc.title, leftDotX, halfTextMax, leftClearance)
		}
		if rightClearance < 24.0 {
			t.Errorf("%q: right circle too close to title: rightDotX=%.2f, halfTextMax=%.2f, clearance=%.2fpx", tc.title, rightDotX, halfTextMax, rightClearance)
		}

		if foundLeftRule && math.Abs((leftDotX-leftRuleEnd)-14.0) > 0.1 {
			t.Errorf("%q: left rule gap = %.2f, want 14.0", tc.title, leftDotX-leftRuleEnd)
		}
		if foundRightRule && math.Abs((rightRuleStart-rightDotX)-14.0) > 0.1 {
			t.Errorf("%q: right rule gap = %.2f, want 14.0", tc.title, rightRuleStart-rightDotX)
		}
	}
}

func TestSanitizeInlineSVG(t *testing.T) {
	t.Parallel()

	raw := `<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="48px" height="48px">
  <style>
    @import url("https://fonts.example/foo.css");
    .icon:hover { fill: red; }
    .grad { fill: url(#grad); }
  </style>
  <defs>
    <linearGradient id="grad">
      <stop offset="0%" stop-color="#cba6f7"/>
      <stop offset="100%" stop-color="#89b4fa"/>
    </linearGradient>
    <clipPath id="clip1"><rect width="48" height="48"/></clipPath>
  </defs>
  <script type="text/javascript">window.location="https://evil.example"</script>
  <a href="https://example.com" onclick="alert(1)">
    <g clip-path="url(#clip1)" onmouseover="evil()">
      <rect x="4" y="4" width="40" height="40" fill="url(#grad)"/>
      <use xlink:href="#grad"/>
      <image href="https://remote.example/pic.png" width="10" height="10"/>
    </g>
  </a>
</svg>`

	sanitized, err := SanitizeInlineSVG(raw, 10, 12, 64)
	if err != nil {
		t.Fatalf("SanitizeInlineSVG unexpected error: %v", err)
	}

	wrapped := `<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 80 80"><style>` + reducedMotionCSS + `</style>` + sanitized + `</svg>`
	assertValidGitHubSVG(t, "sanitized-inline", []byte(wrapped))

	if !strings.Contains(sanitized, `viewBox="0 0 48 48"`) {
		t.Errorf("expected synthesized viewBox=\"0 0 48 48\", got: %s", sanitized)
	}
	if strings.Contains(sanitized, `id="grad"`) {
		t.Errorf("unprefixed id=\"grad\" still present: %s", sanitized)
	}
	if strings.Contains(sanitized, `url(#grad)`) {
		t.Errorf("unprefixed url(#grad) still present: %s", sanitized)
	}
	if !strings.Contains(sanitized, `id="vlogo-`) || !strings.Contains(sanitized, `url(#vlogo-`) {
		t.Errorf("expected prefixed vlogo- IDs and url(#vlogo-...) references, got: %s", sanitized)
	}
	if !strings.Contains(sanitized, `href="#vlogo-`) {
		t.Errorf("expected normalized prefixed href=\"#vlogo-...\", got: %s", sanitized)
	}
	if strings.Contains(strings.ToLower(sanitized), "onclick") || strings.Contains(strings.ToLower(sanitized), "onmouseover") {
		t.Errorf("event handler attributes must be stripped: %s", sanitized)
	}

	if _, err := SanitizeInlineSVG("", 0, 0, 32); err == nil {
		t.Error("expected error for empty SVG")
	}
	if _, err := SanitizeInlineSVG(raw, 0, 0, 0); err == nil {
		t.Error("expected error for non-positive size")
	}
}

func TestRenderDataURIImageValidation(t *testing.T) {
	t.Parallel()

	valid := "data:image/png;base64,iVBORw0KGgo="
	out, err := RenderDataURIImage(valid, 20, 24, 64)
	if err != nil {
		t.Fatalf("unexpected error for valid data URI: %v", err)
	}
	if !strings.Contains(out, `<image href="data:image/png;base64,iVBORw0KGgo="`) {
		t.Errorf("unexpected image markup: %s", out)
	}

	for _, bad := range []string{
		"",
		"https://example.com/logo.png",
		"javascript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
	} {
		if _, err := RenderDataURIImage(bad, 0, 0, 32); err == nil {
			t.Errorf("expected error for invalid URI %q", bad)
		}
	}
}

func TestWrapTextAndXMLEscaping(t *testing.T) {
	t.Parallel()

	lines := WrapText("A crisp modern Catppuccin card renderer for GitHub profile READMEs with wrapping support.", 32, 2)
	if len(lines) != 2 {
		t.Fatalf("WrapText returned %d lines, want 2: %v", len(lines), lines)
	}
	for i, line := range lines {
		if len([]rune(line)) > 32 {
			t.Errorf("line %d exceeds max length 32: %q", i, line)
		}
	}
	if !strings.HasSuffix(lines[1], "…") {
		t.Errorf("expected truncated final line to end with ellipsis, got %q", lines[1])
	}

	r := NewRenderer(CatppuccinMochaPalette())
	card, err := r.RenderProjectCard(ProjectCardData{
		Slug:       "esc-test",
		Name:       `A & B <"C"> 'D'`,
		Tagline:    `Handles <tags> & "quotes" & 'apostrophes' cleanly without breaking XML.`,
		Version:    "v1.0&2",
		Language:   "C++ & Go",
		ReleaseAge: "<1d ago",
	})
	if err != nil {
		t.Fatalf("RenderProjectCard escaping error: %v", err)
	}
	assertValidGitHubSVG(t, "xml-escaping", card)
	if !strings.Contains(string(card), "&amp;") || !strings.Contains(string(card), "&lt;") {
		t.Errorf("expected escaped XML entities in rendered output")
	}
}

func TestPlaceholderDistinctionAndTwitchVariants(t *testing.T) {
	t.Parallel()

	p := CatppuccinMochaPalette()
	r := NewRenderer(p)

	jiveCard, err := r.RenderProjectCard(ProjectCardData{
		Slug: "jive-meter",
		Name: "jive-meter",
	})
	if err != nil {
		t.Fatalf("RenderProjectCard(jive) error: %v", err)
	}
	if !strings.Contains(string(jiveCard), "vitrine-jive-crest") || !strings.Contains(string(jiveCard), "vitrine-wave-bar") {
		t.Errorf("expected animated Jive waveform crest for jive-* slug")
	}

	genericCard, err := r.RenderProjectCard(ProjectCardData{
		Slug: "obs-studio",
		Name: "OBS Studio",
	})
	if err != nil {
		t.Fatalf("RenderProjectCard(generic) error: %v", err)
	}
	if !strings.Contains(string(genericCard), "vitrine-monogram") || !strings.Contains(string(genericCard), ">OS<") {
		t.Errorf("expected monogram crest with initials OS for generic missing logo")
	}

	live, err := r.RenderTwitchStatus(TwitchData{Live: true, Uptime: "1h 42m"})
	if err != nil {
		t.Fatalf("RenderTwitchStatus(live) error: %v", err)
	}
	liveStr := string(live)
	for _, want := range []string{
		`width="220" height="40"`,
		fmt.Sprintf(`stroke="%s" stroke-width="1.5"`, p.AccentRed),
		`M11.571 4.714h1.715`,
		"LIVE ON TWITCH",
		"1h 42m",
		"vitrine-live-ring",
	} {
		if !strings.Contains(liveStr, want) {
			t.Errorf("live Twitch tile missing %q", want)
		}
	}

	offline, err := r.RenderTwitchStatus(TwitchData{Live: false})
	if err != nil {
		t.Fatalf("RenderTwitchStatus(offline) error: %v", err)
	}
	offStr := string(offline)
	for _, want := range []string{
		`width="220" height="40"`,
		fmt.Sprintf(`stroke="%s" stroke-width="1.5"`, p.Surface1),
		`M11.571 4.714h1.715`,
		"TWITCH · OFFLINE",
		"vitrine-offline-pip",
	} {
		if !strings.Contains(offStr, want) {
			t.Errorf("offline Twitch tile missing %q", want)
		}
	}
	if strings.Contains(offStr, "LIVE ON TWITCH") {
		t.Errorf("offline Twitch tile must not contain LIVE ON TWITCH")
	}
}

func TestPaletteByNameAndHelpers(t *testing.T) {
	t.Parallel()

	if got := PaletteByName("catppuccin-mocha"); got != CatppuccinMochaPalette() {
		t.Errorf("PaletteByName(catppuccin-mocha) mismatch")
	}
	if got := PaletteByName("catppuccin-latte"); got != CatppuccinLattePalette() {
		t.Errorf("PaletteByName(catppuccin-latte) mismatch")
	}
	if got := PaletteByName("unknown"); got != CatppuccinMochaPalette() {
		t.Errorf("PaletteByName(unknown) should fall back to Catppuccin Mocha")
	}

	if got := FormatNumber(24810); got != "24,810" {
		t.Errorf("FormatNumber(24810) = %q, want 24,810", got)
	}
	if got := FormatCompactCount(810); got != "810" {
		t.Errorf("FormatCompactCount(810) = %q, want 810", got)
	}
	if got := FormatCompactCount(1420); got != "1.4k" {
		t.Errorf("FormatCompactCount(1420) = %q, want 1.4k", got)
	}
}
