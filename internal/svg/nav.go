package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// RenderNavPill renders a 44px-tall Catppuccin social navigation pill with a
// single 1.5px platform-tinted border, an authentic 20x20 brand icon tightly
// coupled beside a 14px Work Sans label, and a staggered diagonal shimmer sweep.
func (r *Renderer) RenderNavPill(data NavPillData) ([]byte, error) {
	p := r.palette
	label := TruncateText(fallbackString(data.Label, "Link"), 22)
	runeCount := utf8.RuneCountInString(label)

	approxTextWidth := float64(runeCount)*8.4 + float64(max(runeCount-1, 0))*0.4
	groupWidth := 20.0 + 10.0 + approxTextWidth
	width := max(168, int(math.Round(groupWidth+56)))
	iconX := (float64(width) - groupWidth) / 2.0
	textX := iconX + 30.0
	delay := math.Max(0, data.DelaySeconds)
	accent := navPlatformAccent(data.Icon, p)

	extraDefs := fmt.Sprintf(`
  <clipPath id="pill-clip">
    <rect x="2" y="2" width="%d" height="40" rx="9"/>
  </clipPath>`, width-4)

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="44" viewBox="0 0 %d 44" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-pill-sheen {
    0%% { transform: translateX(-80px); opacity: 0; }
    12%% { opacity: 1; }
    42%% { transform: translateX(%dpx); opacity: 1; }
    52%%, 100%% { transform: translateX(%dpx); opacity: 0; }
  }
  @keyframes vitrine-icon-breathe {
    0%%, 100%% { opacity: 0.82; }
    50%% { opacity: 1; }
  }
  .vitrine-pill-glow {
    animation: vitrine-pill-sheen 5.5s ease-in-out %ss infinite;
  }
  .vitrine-pill-icon {
    animation: vitrine-icon-breathe 4.4s ease-in-out %ss infinite;
  }
  %s
</style>

<!-- Single 1.5px platform-accent outer border -->
<rect x="1" y="1" width="%d" height="42" rx="10" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Staggered diagonal shimmer sweep -->
<g clip-path="url(#pill-clip)">
  <g class="vitrine-pill-glow">
    <rect x="0" y="-4" width="64" height="52" fill="url(#cat-sheen)" transform="skewX(-18)"/>
  </g>
</g>

<!-- Coupled 20x20 authentic brand icon and 14px Work Sans label -->
<g class="vitrine-pill-icon">
  %s
</g>
<text x="%s" y="27" text-anchor="start" font-family="%s" font-size="14" font-weight="600" letter-spacing="0.2" fill="%s">%s</text>
</svg>`,
		width, width, escapeXML(label),
		r.baseDefs(extraDefs),
		width+56, width+56,
		formatFloat(delay),
		formatFloat(delay),
		reducedMotionCSS,
		width-2, escapeXML(accent),
		renderNavIcon(data.Icon, accent, iconX, 12, 20),
		formatFloat(textX), fontSans, escapeXML(p.Text), escapeXML(label),
	)

	return []byte(out), nil
}

// RenderTwitchStatus renders a 220x40 Catppuccin Twitch live or offline status
// tile with a single 1.5px border and authentic Twitch glitch icon.
func (r *Renderer) RenderTwitchStatus(data TwitchData) ([]byte, error) {
	p := r.palette

	if data.Live {
		uptime := TruncateText(strings.TrimSpace(data.Uptime), 10)
		aria := "Live on Twitch"
		if uptime != "" {
			aria = "Live on Twitch for " + uptime
		}

		uptimeSVG := ""
		if uptime != "" {
			uptimeSVG = fmt.Sprintf(
				`<text x="208" y="24" text-anchor="end" font-family="%s" font-size="10.5" font-weight="600" fill="%s">%s</text>`,
				fontMono, escapeXML(p.Text), escapeXML(uptime),
			)
		}

		out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="220" height="40" viewBox="0 0 220 40" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-live-pulse {
    0%% { transform: scale(1); opacity: 0.85; }
    72%% { transform: scale(2.2); opacity: 0; }
    100%% { transform: scale(2.2); opacity: 0; }
  }
  .vitrine-live-ring {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-live-pulse 2s ease-out infinite;
  }
  %s
</style>

<!-- Single 1.5px AccentRed outer border -->
<rect x="1" y="1" width="218" height="38" rx="10" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Authentic 18x18 Twitch glitch icon -->
%s

<!-- Pulsing AccentRed sonar ring -->
<circle cx="41" cy="20" r="4" fill="%s" opacity="0.45" class="vitrine-live-ring"/>
<circle cx="41" cy="20" r="3.6" fill="%s"/>

<text x="52" y="24" font-family="%s" font-size="10.5" font-weight="700" letter-spacing="0.8" fill="%s">LIVE ON TWITCH</text>
%s
</svg>`,
			escapeXML(aria),
			r.baseDefs(""),
			reducedMotionCSS,
			escapeXML(p.AccentRed),
			renderNavIcon("twitch", p.AccentRed, 12, 11, 18),
			escapeXML(p.AccentRed),
			escapeXML(p.AccentRed),
			fontSans, escapeXML(p.Text),
			uptimeSVG,
		)
		return []byte(out), nil
	}

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="220" height="40" viewBox="0 0 220 40" role="img" aria-label="Twitch offline">
%s
<style>
  @keyframes vitrine-calm-pulse {
    0%%, 100%% { opacity: 0.42; }
    50%% { opacity: 0.95; }
  }
  .vitrine-offline-pip {
    animation: vitrine-calm-pulse 4.5s ease-in-out infinite;
  }
  %s
</style>

<!-- Single 1.5px Surface1 outer border -->
<rect x="1" y="1" width="218" height="38" rx="10" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Authentic 18x18 Twitch glitch icon -->
%s

<!-- Calm AccentPrimary offline pulse -->
<circle cx="41" cy="20" r="3.5" fill="%s" class="vitrine-offline-pip"/>
<text x="54" y="24" font-family="%s" font-size="11" font-weight="600" letter-spacing="1.2" fill="%s">TWITCH · OFFLINE</text>
</svg>`,
		r.baseDefs(""),
		reducedMotionCSS,
		escapeXML(p.Surface1),
		renderNavIcon("twitch", p.AccentPrimary, 12, 11, 18),
		escapeXML(p.AccentPrimary),
		fontSans, escapeXML(p.Subtext),
	)

	return []byte(out), nil
}
