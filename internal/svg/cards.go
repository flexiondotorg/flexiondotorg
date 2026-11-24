package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// RenderProjectCard renders a 408x182 Catppuccin project card with a single
// 1.5px Surface1 border, clipped slim Catppuccin top accent bar with staggered
// top-edge glint, unframed 64x64 logo, 4-line wrapped description, and clean
// 3-part footer.
func (r *Renderer) RenderProjectCard(data ProjectCardData) ([]byte, error) {
	p := r.palette
	name := fallbackString(data.Name, fallbackString(data.Slug, "Project"))
	version := TruncateText(strings.TrimSpace(data.Version), 18)
	lang := TruncateText(strings.TrimSpace(data.Language), 14)
	age := TruncateText(strings.TrimSpace(data.ReleaseAge), 18)

	versionPillSVG, pillW := r.renderVersionPill(version, data.Fresh)

	// Reserve horizontal space for the actual top-right release pill so title never collides.
	titleMaxRunes := 24
	if pillW > 0 {
		availPx := max(110, 388-pillW-12-100)
		titleMaxRunes = min(24, max(12, availPx/10))
	}
	titleText := TruncateText(name, titleMaxRunes)
	taglineLines := WrapText(data.Tagline, 45, 4)

	logoArt := r.renderLogoArt(data.Slug, name, data.LogoSVG, data.LogoDataURI, 20, 34, 64)

	var taglineSVG strings.Builder
	for i, line := range taglineLines {
		y := 68 + i*18
		fmt.Fprintf(
			&taglineSVG,
			`<text x="100" y="%d" font-family="%s" font-size="12.5" fill="%s">%s</text>`,
			y, fontSans, escapeXML(p.Subtext), escapeXML(line),
		)
	}

	langSVG := ""
	if lang != "" {
		langSVG = fmt.Sprintf(
			`<circle cx="24" cy="159" r="4" fill="%s"/><text x="35" y="163" font-family="%s" font-size="12" font-weight="500" fill="%s">%s</text>`,
			escapeXML(languageColor(lang, p.AccentSecondary)),
			fontSans, escapeXML(p.Text), escapeXML(lang),
		)
	}

	starLabel := FormatCompactCount(max(0, data.Stars))
	starsSVG := fmt.Sprintf(
		`<text x="204" y="163" text-anchor="middle" font-family="%s" font-size="12" font-weight="600" fill="%s"><tspan fill="%s">★</tspan> %s</text>`,
		fontMono, escapeXML(p.Text), escapeXML(p.AccentYellow), escapeXML(starLabel),
	)

	ageSVG := ""
	if age != "" {
		ageSVG = fmt.Sprintf(
			`<text x="388" y="163" text-anchor="end" font-family="%s" font-size="11.5" fill="%s">%s</text>`,
			fontMono, escapeXML(p.Muted), escapeXML(age),
		)
	}

	delay := math.Max(0, data.DelaySeconds)
	extraDefs := `
  <clipPath id="card-clip">
    <rect x="1.75" y="1.75" width="404.5" height="178.5" rx="11.25"/>
  </clipPath>`

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="408" height="182" viewBox="0 0 408 182" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-fresh-pulse {
    0%% { transform: scale(1); opacity: 0.85; }
    75%% { transform: scale(2.2); opacity: 0; }
    100%% { transform: scale(2.2); opacity: 0; }
  }
  @keyframes vitrine-wave {
    0%%, 100%% { transform: scaleY(0.45); opacity: 0.72; }
    50%% { transform: scaleY(1); opacity: 1; }
  }
  @keyframes vitrine-card-glint {
    0%% { transform: translateX(-140px); opacity: 0; }
    10%% { opacity: 1; }
    55%% { transform: translateX(430px); opacity: 1; }
    66%%, 100%% { transform: translateX(430px); opacity: 0; }
  }
  .vitrine-fresh-ring {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-fresh-pulse 2.2s ease-out infinite;
  }
  .vitrine-wave-bar {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-wave 1.8s ease-in-out infinite;
  }
  .vitrine-card-glint {
    animation: vitrine-card-glint 6.5s ease-in-out %ss infinite;
  }
  %s
</style>

<!-- Single 1.5px Surface1 outer border -->
<rect x="1" y="1" width="406" height="180" rx="12" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Clipped slim 2.5px Catppuccin top bar with staggered top-edge glint -->
<g clip-path="url(#card-clip)">
  <rect x="0" y="1" width="408" height="3.25" fill="url(#cat-accent)"/>
  <rect x="0" y="1.75" width="120" height="2.5" rx="1.25" fill="url(#cat-glint)" class="vitrine-card-glint"/>
</g>

<!-- Unframed 64x64 project logo -->
%s

<!-- Top-right release version pill -->
%s

<!-- Project title and wrapped 4-line description -->
<text x="100" y="42" font-family="%s" font-size="18" font-weight="700" letter-spacing="-0.2" fill="%s">%s</text>
%s

<!-- Clean 1px Surface0 footer separator -->
<line x1="20" y1="142" x2="388" y2="142" stroke="%s" stroke-width="1"/>

<!-- Clean 3-part footer: language, stars, release age -->
<g>
  %s
  %s
  %s
</g>
</svg>`,
		escapeXML(name),
		r.baseDefs(extraDefs),
		formatFloat(delay),
		reducedMotionCSS,
		escapeXML(p.Surface1),
		logoArt,
		versionPillSVG,
		fontSans, escapeXML(p.Text), escapeXML(titleText),
		taglineSVG.String(),
		escapeXML(p.Surface0),
		langSVG,
		starsSVG,
		ageSVG,
	)

	return []byte(out), nil
}

func (r *Renderer) renderVersionPill(version string, fresh bool) (string, int) {
	p := r.palette
	if version == "" && !fresh {
		return "", 0
	}
	label := fallbackString(version, "NEW")
	runes := utf8.RuneCountInString(label)
	if fresh {
		pillW := max(58, 32+int(math.Ceil(float64(runes)*6.4)))
		if pillW > 152 {
			pillW = 152
		}
		pillX := 388 - pillW
		return fmt.Sprintf(`<g>
  <rect x="%d" y="22" width="%d" height="22" rx="11" fill="%s" stroke="%s" stroke-width="1"/>
  <circle cx="%d" cy="33" r="3.4" fill="%s" opacity="0.45" class="vitrine-fresh-ring"/>
  <circle cx="%d" cy="33" r="3.1" fill="%s"/>
  <text x="%d" y="36.5" font-family="%s" font-size="10.5" font-weight="600" fill="%s">%s</text>
</g>`,
			pillX, pillW, escapeXML(p.Surface0), escapeXML(p.AccentGreen),
			pillX+12, escapeXML(p.AccentGreen),
			pillX+12, escapeXML(p.AccentGreen),
			pillX+21, fontMono, escapeXML(p.Text), escapeXML(label),
		), pillW
	}

	pillW := max(48, 22+int(math.Ceil(float64(runes)*6.4)))
	if pillW > 144 {
		pillW = 144
	}
	pillX := 388 - pillW
	centerX := float64(pillX) + float64(pillW)/2.0
	return fmt.Sprintf(`<g>
  <rect x="%d" y="22" width="%d" height="22" rx="11" fill="%s" stroke="%s" stroke-width="1"/>
  <text x="%s" y="36.5" text-anchor="middle" font-family="%s" font-size="10.5" font-weight="600" fill="%s">%s</text>
</g>`,
		pillX, pillW, escapeXML(p.Surface0), escapeXML(p.Surface1),
		formatFloat(centerX), fontMono, escapeXML(p.Text), escapeXML(label),
	), pillW
}

// RenderPackageHost renders a 198x78 Catppuccin Haberdasher package repository
// card sized so four tiles fit cleanly per row, with a single 1.5px Surface1
// border, clipped slim Catppuccin top bar with staggered top-edge glint, and
// unframed 40x40 host logo.
func (r *Renderer) RenderPackageHost(data PackageHostData) ([]byte, error) {
	p := r.palette
	name := strings.ToUpper(TruncateText(fallbackString(data.Name, "REPO"), 12))
	host := TruncateText(strings.TrimSpace(data.Host), 21)
	count := max(0, data.PackageCount)

	unit := "pkgs"
	if count == 1 {
		unit = "pkg"
	}
	badgeText := fmt.Sprintf("%d %s", count, unit)
	badgeW := max(54, 18+utf8.RuneCountInString(badgeText)*6)
	if badgeW > 120 {
		badgeW = 120
	}
	badgeCenterX := 64.0 + float64(badgeW)/2.0

	logoArt := r.renderLogoArt(strings.ToLower(name), name, data.LogoSVG, data.LogoDataURI, 14, 20, 40)
	delay := math.Max(0, data.DelaySeconds)
	extraDefs := `
  <clipPath id="host-clip">
    <rect x="1.75" y="1.75" width="194.5" height="74.5" rx="9.25"/>
  </clipPath>`

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="198" height="78" viewBox="0 0 198 78" role="img" aria-label="%s package repository">
%s
<style>
  @keyframes vitrine-host-glint {
    0%% { transform: translateX(-90px); opacity: 0; }
    10%% { opacity: 1; }
    55%% { transform: translateX(220px); opacity: 1; }
    66%%, 100%% { transform: translateX(220px); opacity: 0; }
  }
  .vitrine-host-glint {
    animation: vitrine-host-glint 6.5s ease-in-out %ss infinite;
  }
  %s
</style>

<!-- Single 1.5px Surface1 outer border -->
<rect x="1" y="1" width="196" height="76" rx="10" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Clipped slim 2.5px Catppuccin top bar with staggered top-edge glint -->
<g clip-path="url(#host-clip)">
  <rect x="0" y="1" width="198" height="3.25" fill="url(#cat-accent)"/>
  <rect x="0" y="1.75" width="76" height="2.5" rx="1.25" fill="url(#cat-glint)" class="vitrine-host-glint"/>
</g>

<!-- Unframed 40x40 host logo -->
%s

<!-- Host name, domain subtitle, and Surface0 package count pill -->
<text x="64" y="27" font-family="%s" font-size="14.5" font-weight="700" letter-spacing="0.6" fill="%s">%s</text>
<text x="64" y="42" font-family="%s" font-size="9.5" fill="%s">%s</text>

<g>
  <rect x="64" y="49" width="%d" height="19" rx="9.5" fill="%s"/>
  <text x="%s" y="62" text-anchor="middle" font-family="%s" font-size="10" font-weight="600" fill="%s">%s</text>
</g>
</svg>`,
		escapeXML(name),
		r.baseDefs(extraDefs),
		formatFloat(delay),
		reducedMotionCSS,
		escapeXML(p.Surface1),
		logoArt,
		fontSans, escapeXML(p.Text), escapeXML(name),
		fontMono, escapeXML(p.Muted), escapeXML(host),
		badgeW, escapeXML(p.Surface0),
		formatFloat(badgeCenterX), fontMono, escapeXML(p.AccentPrimary), escapeXML(badgeText),
	)

	return []byte(out), nil
}
