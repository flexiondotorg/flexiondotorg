package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// RenderHero renders the 840x210 Catppuccin hero banner with a single 1.5px
// Catppuccin gradient border, top-edge animated light glint, bold Work Sans
// typography on the left, Catppuccin Harmonic Aurora Waves on the right, and
// an optional modern CTA pill in the bottom-right corner.
func (r *Renderer) RenderHero(data HeroData) ([]byte, error) {
	p := r.palette
	eyebrow := strings.TrimSpace(data.Eyebrow)
	title := fallbackString(data.Title, "WIMPY'S WORLD")
	subtitle := strings.TrimSpace(data.Subtitle)
	cta := strings.TrimSpace(data.CTA)

	titleRunes := utf8.RuneCountInString(title)
	var (
		titleSize    int
		titleY       int
		subFontSize  string
		subWrapChars int
		eyebrowSVG   string
	)
	if eyebrow == "" {
		titleSize = 42
		switch {
		case titleRunes > 26:
			titleSize = 28
		case titleRunes > 18:
			titleSize = 34
		}
		titleY = 96
		subFontSize = "16"
		subWrapChars = 44
	} else {
		titleSize = 36
		switch {
		case titleRunes > 28:
			titleSize = 24
		case titleRunes > 20:
			titleSize = 30
		}
		titleY = 104
		subFontSize = "15"
		subWrapChars = 46
		eyebrowSVG = fmt.Sprintf(
			`  <circle cx="42" cy="48" r="3.2" fill="%s"/>
  <text x="53" y="52" font-family="%s" font-size="11" font-weight="600" letter-spacing="2.2" fill="%s">%s</text>
`,
			escapeXML(p.AccentPrimary),
			fontSans, escapeXML(p.AccentPrimary), escapeXML(strings.ToUpper(TruncateText(eyebrow, 42))),
		)
	}

	titleText := TruncateText(title, 34)
	subLines := WrapText(subtitle, subWrapChars, 2)

	var subSVG strings.Builder
	for i, line := range subLines {
		var y int
		if eyebrow == "" {
			if len(subLines) == 1 {
				y = 132
			} else {
				y = 128 + i*24
			}
		} else {
			y = 134 + i*22
		}
		fmt.Fprintf(
			&subSVG,
			`<text x="38" y="%d" font-family="%s" font-size="%s" fill="%s">%s</text>`,
			y, fontSans, subFontSize, escapeXML(p.Subtext), escapeXML(line),
		)
	}

	fillOpacityTop := "0.20"
	if p.IsLight {
		fillOpacityTop = "0.14"
	}

	var extraDefs strings.Builder
	fmt.Fprintf(&extraDefs, `
  <clipPath id="hero-clip">
    <rect x="1.75" y="1.75" width="836.5" height="206.5" rx="13.25"/>
  </clipPath>
  <clipPath id="hero-top-glint-clip">
    <rect x="14" y="0" width="812" height="3.5"/>
  </clipPath>
  <linearGradient id="aurora-wave-1" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="aurora-wave-2" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="aurora-wave-3" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="aurora-fill-1" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="100%%" stop-color="%s" stop-opacity="0"/>
  </linearGradient>
  <linearGradient id="aurora-fill-2" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="100%%" stop-color="%s" stop-opacity="0"/>
  </linearGradient>
  <linearGradient id="aurora-fill-3" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="100%%" stop-color="%s" stop-opacity="0"/>
  </linearGradient>`,
		escapeXML(p.AccentPink), escapeXML(p.AccentPrimary),
		escapeXML(p.AccentSecondary), escapeXML(p.AccentSapphire),
		escapeXML(p.AccentTertiary), escapeXML(p.AccentGreen),
		escapeXML(p.AccentPrimary), fillOpacityTop, escapeXML(p.Base),
		escapeXML(p.AccentSecondary), fillOpacityTop, escapeXML(p.Base),
		escapeXML(p.AccentTertiary), fillOpacityTop, escapeXML(p.Base),
	)

	ctaCSS := ""
	ctaSVG := ""
	if cta != "" {
		ctaText := TruncateText(cta, 26)
		pillW := max(126, min(220, int(math.Round(float64(utf8.RuneCountInString(ctaText))*7.8+28))))
		pillX := 816 - pillW
		pillCenterX := float64(pillX) + float64(pillW)/2.0
		fmt.Fprintf(&extraDefs, `
  <clipPath id="hero-cta-clip">
    <rect x="%d" y="158" width="%d" height="32" rx="16"/>
  </clipPath>`, pillX, pillW)

		ctaCSS = fmt.Sprintf(`
  @keyframes vitrine-cta-sweep {
    0%% { transform: translateX(-70px); opacity: 0; }
    12%% { opacity: 0.95; }
    38%% { transform: translateX(%dpx); opacity: 0.95; }
    46%%, 100%% { transform: translateX(%dpx); opacity: 0; }
  }
  .vitrine-cta-sheen {
    animation: vitrine-cta-sweep 6s ease-in-out 1.4s infinite;
  }`, pillW+60, pillW+60)

		ctaSVG = fmt.Sprintf(`
<!-- Modern bottom-right Catppuccin CTA pill -->
<g class="vitrine-hero-cta">
  <rect x="%d" y="158" width="%d" height="32" rx="16" fill="%s" stroke="url(#cat-accent)" stroke-width="1"/>
  <g clip-path="url(#hero-cta-clip)">
    <g class="vitrine-cta-sheen">
      <rect x="%d" y="154" width="54" height="40" fill="url(#cat-sheen)" transform="skewX(-18)"/>
    </g>
  </g>
  <text x="%s" y="178" text-anchor="middle" font-family="%s" font-size="12" font-weight="600" fill="%s">%s</text>
</g>`,
			pillX, pillW, escapeXML(p.Surface0),
			pillX,
			formatFloat(pillCenterX), fontMono, escapeXML(p.Text), escapeXML(ctaText),
		)
	}

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="840" height="210" viewBox="0 0 840 210" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-hero-glint {
    0%% { transform: translateX(-180px); opacity: 0; }
    10%% { opacity: 1; }
    65%% { transform: translateX(860px); opacity: 1; }
    75%%, 100%% { transform: translateX(860px); opacity: 0; }
  }
  @keyframes vitrine-wave-flow {
    0%% { stroke-dashoffset: 0; }
    100%% { stroke-dashoffset: -380; }
  }
  @keyframes vitrine-wave-drift-a {
    0%%, 100%% { transform: translateY(-4px); }
    50%% { transform: translateY(5px); }
  }
  @keyframes vitrine-wave-drift-b {
    0%%, 100%% { transform: translateY(5px); }
    50%% { transform: translateY(-5px); }
  }
  @keyframes vitrine-wave-drift-c {
    0%%, 100%% { transform: translateY(-3px); }
    50%% { transform: translateY(6px); }
  }
  @keyframes vitrine-aurora-dot {
    0%%, 100%% { transform: scale(0.85); opacity: 0.35; }
    50%% { transform: scale(1.25); opacity: 0.95; }
  }
  .vitrine-hero-glint {
    animation: vitrine-hero-glint 5.5s ease-in-out infinite;
  }
  .vitrine-wave-drift-1 {
    animation: vitrine-wave-drift-a 6.8s ease-in-out infinite;
  }
  .vitrine-wave-drift-2 {
    animation: vitrine-wave-drift-b 7.8s ease-in-out 0.8s infinite;
  }
  .vitrine-wave-drift-3 {
    animation: vitrine-wave-drift-c 8.8s ease-in-out 1.6s infinite;
  }
  .vitrine-wave-flow-1 {
    stroke-dasharray: 150 230;
    animation: vitrine-wave-flow 7.5s linear infinite;
  }
  .vitrine-wave-flow-2 {
    stroke-dasharray: 130 250;
    animation: vitrine-wave-flow 9s linear 1.2s infinite;
  }
  .vitrine-wave-flow-3 {
    stroke-dasharray: 115 265;
    animation: vitrine-wave-flow 10.5s linear 2.4s infinite;
  }
  .vitrine-aurora-dot {
    transform-box: fill-box;
    transform-origin: center;
  }
  .vitrine-dot-1 { animation: vitrine-aurora-dot 3.6s ease-in-out infinite; }
  .vitrine-dot-2 { animation: vitrine-aurora-dot 4.2s ease-in-out 0.6s infinite; }
  .vitrine-dot-3 { animation: vitrine-aurora-dot 3.9s ease-in-out 1.3s infinite; }
  .vitrine-dot-4 { animation: vitrine-aurora-dot 4.5s ease-in-out 2.0s infinite; }
  .vitrine-dot-5 { animation: vitrine-aurora-dot 3.4s ease-in-out 2.7s infinite; }%s
  %s
</style>

<!-- Single 1.5px Catppuccin gradient outer frame -->
<rect x="1" y="1" width="838" height="208" rx="14" fill="url(#card-bg)" stroke="url(#cat-accent)" stroke-width="1.5"/>

<!-- Animated top-edge light glint -->
<g clip-path="url(#hero-top-glint-clip)">
  <rect x="0" y="0.25" width="170" height="2.5" rx="1.25" fill="url(#cat-glint)" class="vitrine-hero-glint"/>
</g>

<!-- Right-side Catppuccin Harmonic Aurora Waves -->
<g clip-path="url(#hero-clip)" class="vitrine-aurora-waves">
  <g class="vitrine-wave-drift-1">
    <path d="M460 108 C525 42, 598 158, 672 88 C730 34, 784 126, 838 72 L838 210 L460 210 Z" fill="url(#aurora-fill-1)"/>
    <path d="M460 108 C525 42, 598 158, 672 88 C730 34, 784 126, 838 72" fill="none" stroke="url(#aurora-wave-1)" stroke-width="3.5" stroke-linecap="round" opacity="0.45"/>
    <path d="M460 108 C525 42, 598 158, 672 88 C730 34, 784 126, 838 72" fill="none" stroke="url(#aurora-wave-1)" stroke-width="4" stroke-linecap="round" class="vitrine-wave-flow-1"/>
  </g>
  <g class="vitrine-wave-drift-2">
    <path d="M472 134 C544 76, 616 174, 694 106 C750 56, 794 142, 838 98 L838 210 L472 210 Z" fill="url(#aurora-fill-2)"/>
    <path d="M472 134 C544 76, 616 174, 694 106 C750 56, 794 142, 838 98" fill="none" stroke="url(#aurora-wave-2)" stroke-width="3" stroke-linecap="round" opacity="0.48"/>
    <path d="M472 134 C544 76, 616 174, 694 106 C750 56, 794 142, 838 98" fill="none" stroke="url(#aurora-wave-2)" stroke-width="3.4" stroke-linecap="round" class="vitrine-wave-flow-2"/>
  </g>
  <g class="vitrine-wave-drift-3">
    <path d="M488 156 C560 108, 636 188, 718 130 C768 96, 804 154, 838 124 L838 210 L488 210 Z" fill="url(#aurora-fill-3)"/>
    <path d="M488 156 C560 108, 636 188, 718 130 C768 96, 804 154, 838 124" fill="none" stroke="url(#aurora-wave-3)" stroke-width="2.5" stroke-linecap="round" opacity="0.5"/>
    <path d="M488 156 C560 108, 636 188, 718 130 C768 96, 804 154, 838 124" fill="none" stroke="url(#aurora-wave-3)" stroke-width="2.8" stroke-linecap="round" class="vitrine-wave-flow-3"/>
  </g>

  <!-- Five softly pulsing pastel accent dots -->
  <circle cx="510" cy="70" r="3" fill="%s" class="vitrine-aurora-dot vitrine-dot-1"/>
  <circle cx="604" cy="48" r="2.7" fill="%s" class="vitrine-aurora-dot vitrine-dot-2"/>
  <circle cx="692" cy="64" r="3.2" fill="%s" class="vitrine-aurora-dot vitrine-dot-3"/>
  <circle cx="766" cy="44" r="2.6" fill="%s" class="vitrine-aurora-dot vitrine-dot-4"/>
  <circle cx="804" cy="96" r="2.9" fill="%s" class="vitrine-aurora-dot vitrine-dot-5"/>
</g>

<!-- Left-aligned Work Sans typography -->
<g>
%s  <text x="38" y="%d" font-family="%s" font-size="%d" font-weight="700" letter-spacing="-0.4" fill="%s">%s</text>
  %s
</g>%s
</svg>`,
		escapeXML(titleText),
		r.baseDefs(extraDefs.String()),
		ctaCSS,
		reducedMotionCSS,
		escapeXML(p.AccentPink),
		escapeXML(p.AccentPrimary),
		escapeXML(p.AccentSecondary),
		escapeXML(p.AccentTertiary),
		escapeXML(p.AccentGreen),
		eyebrowSVG,
		titleY, fontSans, titleSize, escapeXML(p.Text), escapeXML(titleText),
		subSVG.String(),
		ctaSVG,
	)

	return []byte(out), nil
}
