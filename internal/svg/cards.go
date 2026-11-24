package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// RenderProjectCard renders a 408x164 Catppuccin project card with a single
// 1.5px Surface1 border, clipped slim Catppuccin top accent bar, unframed
// 64x64 logo, staggered diagonal light sheen, and clean 3-part footer.
func (r *Renderer) RenderProjectCard(data ProjectCardData) ([]byte, error) {
	p := r.palette
	name := fallbackString(data.Name, fallbackString(data.Slug, "Project"))
	version := TruncateText(strings.TrimSpace(data.Version), 12)
	lang := TruncateText(strings.TrimSpace(data.Language), 14)
	age := TruncateText(strings.TrimSpace(data.ReleaseAge), 18)

	// Reserve horizontal space for the top-right release pill so title never collides.
	titleMaxRunes := 24
	if version != "" || data.Fresh {
		titleMaxRunes = 17
	}
	titleText := TruncateText(name, titleMaxRunes)
	taglineLines := WrapText(data.Tagline, 40, 2)

	logoArt := r.renderLogoArt(data.Slug, name, data.LogoSVG, data.LogoDataURI, 20, 24, 64)

	var taglineSVG strings.Builder
	for i, line := range taglineLines {
		y := 67 + i*19
		fmt.Fprintf(
			&taglineSVG,
			`<text x="100" y="%d" font-family="%s" font-size="13" fill="%s">%s</text>`,
			y, fontSans, escapeXML(p.Subtext), escapeXML(line),
		)
	}

	versionPillSVG := r.renderVersionPill(version, data.Fresh)

	langSVG := ""
	if lang != "" {
		langSVG = fmt.Sprintf(
			`<circle cx="24" cy="144" r="4" fill="%s"/><text x="35" y="148" font-family="%s" font-size="12" font-weight="500" fill="%s">%s</text>`,
			escapeXML(languageColor(lang, p.AccentSecondary)),
			fontSans, escapeXML(p.Text), escapeXML(lang),
		)
	}

	starLabel := FormatCompactCount(max(0, data.Stars))
	starsSVG := fmt.Sprintf(
		`<text x="204" y="148" text-anchor="middle" font-family="%s" font-size="12" font-weight="600" fill="%s"><tspan fill="%s">★</tspan> %s</text>`,
		fontMono, escapeXML(p.Text), escapeXML(p.AccentYellow), escapeXML(starLabel),
	)

	ageSVG := ""
	if age != "" {
		ageSVG = fmt.Sprintf(
			`<text x="388" y="148" text-anchor="end" font-family="%s" font-size="11.5" fill="%s">%s</text>`,
			fontMono, escapeXML(p.Muted), escapeXML(age),
		)
	}

	delay := math.Max(0, data.DelaySeconds)
	extraDefs := `
  <clipPath id="card-clip">
    <rect x="1.75" y="1.75" width="404.5" height="160.5" rx="11.25"/>
  </clipPath>`

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="408" height="164" viewBox="0 0 408 164" role="img" aria-label="%s">
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
  @keyframes vitrine-card-sweep {
    0%% { transform: translateX(-140px); opacity: 0; }
    8%% { opacity: 1; }
    32%% { transform: translateX(520px); opacity: 1; }
    40%%, 100%% { transform: translateX(520px); opacity: 0; }
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
  .vitrine-card-sheen {
    animation: vitrine-card-sweep 8s ease-in-out %ss infinite;
  }
  %s
</style>

<!-- Single 1.5px Surface1 outer border -->
<rect x="1" y="1" width="406" height="162" rx="12" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Clipped slim 2.5px Catppuccin top bar and staggered diagonal sheen -->
<g clip-path="url(#card-clip)">
  <rect x="0" y="1" width="408" height="3.25" fill="url(#cat-accent)"/>
  <g class="vitrine-card-sheen">
    <rect x="0" y="-12" width="96" height="188" fill="url(#cat-sheen)" transform="skewX(-18)"/>
  </g>
</g>

<!-- Unframed 64x64 project logo -->
%s

<!-- Top-right release version pill -->
%s

<!-- Project title and wrapped 2-line description -->
<text x="100" y="42" font-family="%s" font-size="18" font-weight="700" letter-spacing="-0.2" fill="%s">%s</text>
%s

<!-- Clean 1px Surface0 footer separator -->
<line x1="20" y1="128" x2="388" y2="128" stroke="%s" stroke-width="1"/>

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

func (r *Renderer) renderVersionPill(version string, fresh bool) string {
	p := r.palette
	if version == "" && !fresh {
		return ""
	}
	label := fallbackString(version, "NEW")
	runes := utf8.RuneCountInString(label)
	if fresh {
		pillW := max(58, 28+runes*6)
		if pillW > 116 {
			pillW = 116
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
		)
	}

	pillW := max(48, 18+runes*6)
	if pillW > 108 {
		pillW = 108
	}
	pillX := 388 - pillW
	centerX := float64(pillX) + float64(pillW)/2.0
	return fmt.Sprintf(`<g>
  <rect x="%d" y="22" width="%d" height="22" rx="11" fill="%s" stroke="%s" stroke-width="1"/>
  <text x="%s" y="36.5" text-anchor="middle" font-family="%s" font-size="10.5" font-weight="600" fill="%s">%s</text>
</g>`,
		pillX, pillW, escapeXML(p.Surface0), escapeXML(p.Surface1),
		formatFloat(centerX), fontMono, escapeXML(p.Text), escapeXML(label),
	)
}

// RenderPackageHost renders a 198x78 Catppuccin Haberdasher package repository
// card sized so four tiles fit cleanly per row, with a single 1.5px Surface1
// border, clipped slim Catppuccin top bar, unframed 40x40 host logo, and
// staggered diagonal sheen.
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
  @keyframes vitrine-host-sweep {
    0%% { transform: translateX(-90px); opacity: 0; }
    10%% { opacity: 1; }
    36%% { transform: translateX(260px); opacity: 1; }
    46%%, 100%% { transform: translateX(260px); opacity: 0; }
  }
  .vitrine-host-sheen {
    animation: vitrine-host-sweep 7s ease-in-out %ss infinite;
  }
  %s
</style>

<!-- Single 1.5px Surface1 outer border -->
<rect x="1" y="1" width="196" height="76" rx="10" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Clipped slim 2.5px Catppuccin top bar and staggered diagonal sheen -->
<g clip-path="url(#host-clip)">
  <rect x="0" y="1" width="198" height="3.25" fill="url(#cat-accent)"/>
  <g class="vitrine-host-sheen">
    <rect x="0" y="-8" width="60" height="94" fill="url(#cat-sheen)" transform="skewX(-18)"/>
  </g>
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
