package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

func statAccentColor(label string, idx int, p Palette) string {
	lower := strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(lower, "star"):
		return p.AccentYellow
	case strings.Contains(lower, "commit"):
		return p.AccentGreen
	case strings.Contains(lower, "pull") || lower == "prs":
		return p.AccentPrimary
	case strings.Contains(lower, "issue"):
		return p.AccentPeach
	case strings.Contains(lower, "follow"):
		return p.AccentSapphire
	}
	cycle := []string{p.AccentYellow, p.AccentGreen, p.AccentPrimary, p.AccentPeach, p.AccentSapphire}
	return cycle[idx%len(cycle)]
}

// RenderStatsStrip renders an 840x98 full-width Catppuccin 5-metric stats strip
// with a single 1.5px Catppuccin gradient border, bright animated top-edge
// light glint, subtle Surface1 column dividers, and staggered pulsing accent
// dots.
func (r *Renderer) RenderStatsStrip(data StatsData) ([]byte, error) {
	p := r.palette
	metrics := data.Metrics
	if len(metrics) == 0 {
		metrics = data.Items
	}
	if len(metrics) == 0 {
		metrics = []StatMetric{
			{Label: "STARS", Value: "0"},
			{Label: "COMMITS", Value: "0"},
			{Label: "PULL REQUESTS", Value: "0"},
			{Label: "ISSUES", Value: "0"},
			{Label: "FOLLOWERS", Value: "0"},
		}
	}

	const totalWidth = 840.0
	slotW := totalWidth / float64(len(metrics))

	var cols strings.Builder
	for i, item := range metrics {
		cx := slotW * (float64(i) + 0.5)
		val := TruncateText(fallbackString(item.Value, "0"), 12)
		lbl := strings.ToUpper(TruncateText(fallbackString(item.Label, "METRIC"), 18))
		dotColor := statAccentColor(lbl, i, p)
		dotDelay := float64(i) * 0.45

		fmt.Fprintf(&cols, `<g>
  <circle cx="%s" cy="22" r="3.4" fill="%s" class="vitrine-stat-dot" style="animation-delay:%ss"/>
  <text x="%s" y="54" text-anchor="middle" font-family="%s" font-size="23" font-weight="700" letter-spacing="-0.2" fill="%s">%s</text>
  <text x="%s" y="76" text-anchor="middle" font-family="%s" font-size="10.5" font-weight="600" letter-spacing="1.4" fill="%s">%s</text>
</g>`,
			formatFloat(cx), escapeXML(dotColor), formatFloat(dotDelay),
			formatFloat(cx), fontSans, escapeXML(p.Text), escapeXML(val),
			formatFloat(cx), fontSans, escapeXML(p.Subtext), escapeXML(lbl),
		)

		if i < len(metrics)-1 {
			divX := slotW * float64(i+1)
			fmt.Fprintf(&cols, `<line x1="%s" y1="22" x2="%s" y2="76" stroke="%s" stroke-width="1"/>`,
				formatFloat(divX), formatFloat(divX), escapeXML(p.Surface1),
			)
		}
	}

	extraDefs := `
  <clipPath id="stats-top-clip">
    <rect x="12" y="0" width="816" height="3.5"/>
  </clipPath>`

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="840" height="98" viewBox="0 0 840 98" role="img" aria-label="GitHub statistics">
%s
<style>
  @keyframes vitrine-stats-glint {
    0%% { transform: translateX(-180px); opacity: 0; }
    12%% { opacity: 1; }
    68%% { transform: translateX(860px); opacity: 1; }
    80%%, 100%% { transform: translateX(860px); opacity: 0; }
  }
  @keyframes vitrine-stat-pulse {
    0%%, 100%% { transform: scale(0.88); opacity: 0.55; }
    50%% { transform: scale(1.22); opacity: 1; }
  }
  .vitrine-stats-glint {
    animation: vitrine-stats-glint 5s ease-in-out infinite;
  }
  .vitrine-stat-dot {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-stat-pulse 3.6s ease-in-out infinite;
  }
  %s
</style>

<!-- Single 1.5px Catppuccin gradient outer border -->
<rect x="1" y="1" width="838" height="96" rx="12" fill="url(#card-bg)" stroke="url(#cat-accent)" stroke-width="1.5"/>

<!-- Bright animated Catppuccin light glint along top edge -->
<g clip-path="url(#stats-top-clip)">
  <rect x="0" y="0.25" width="170" height="2.5" rx="1.25" fill="url(#cat-glint)" class="vitrine-stats-glint"/>
</g>

%s
</svg>`,
		r.baseDefs(extraDefs),
		reducedMotionCSS,
		cols.String(),
	)

	return []byte(out), nil
}

// RenderDiscordBanner renders an 840x68 full-width Catppuccin Discord
// community banner with a single 1.5px Catppuccin gradient border, unframed
// 28x28 Discord Clyde icon, animated AccentGreen sonar ring, Fira Code
// telemetry, and a Surface0 JOIN pill.
func (r *Renderer) RenderDiscordBanner(data DiscordData) ([]byte, error) {
	p := r.palette
	name := fallbackString(data.Name, "Wimpy's World Discord")
	tagline := strings.TrimSpace(data.Tagline)
	cta := strings.ToUpper(TruncateText(fallbackString(data.CTA, "JOIN ↗"), 16))

	titleText := TruncateText(name, 34)
	taglineText := TruncateText(tagline, 54)

	onlineCount := max(0, data.Online)
	onlineText := fmt.Sprintf("%s online", FormatNumber(onlineCount))

	hasMembers := data.Members > 0
	membersText := ""
	if hasMembers {
		membersText = fmt.Sprintf("%s members", FormatNumber(data.Members))
	}

	badgeW := max(74, int(math.Round(float64(utf8.RuneCountInString(cta))*7.6+24)))
	badgeX := 818 - badgeW
	badgeCenterX := float64(badgeX) + float64(badgeW)/2.0
	sepX := float64(badgeX) - 16.0

	onlineW := float64(utf8.RuneCountInString(onlineText))*7.4 + 20.0
	membersW := float64(utf8.RuneCountInString(membersText))*6.8 + 16.0
	blockW := math.Max(onlineW, membersW)
	statsLeftX := math.Max(480.0, sepX-18.0-blockW)

	titleY := "30"
	taglineSVG := ""
	if taglineText != "" {
		taglineSVG = fmt.Sprintf(
			`<text x="66" y="49" font-family="%s" font-size="12.5" fill="%s">%s</text>`,
			fontSans, escapeXML(p.Subtext), escapeXML(taglineText),
		)
	} else {
		titleY = "39"
	}

	onlineY := 30.0
	beaconCY := 26.0
	membersSVG := ""
	if hasMembers {
		membersSVG = fmt.Sprintf(
			`<text x="%s" y="49" font-family="%s" font-size="11" font-weight="500" fill="%s">%s</text>`,
			formatFloat(statsLeftX+16), fontMono, escapeXML(p.Subtext), escapeXML(membersText),
		)
	} else {
		onlineY = 38.5
		beaconCY = 34.0
	}

	beaconCX := statsLeftX + 5.0
	aria := fmt.Sprintf("%s - %s", titleText, onlineText)
	if hasMembers {
		aria = fmt.Sprintf("%s - %s, %s", titleText, onlineText, membersText)
	}

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="840" height="68" viewBox="0 0 840 68" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-discord-pulse {
    0%% { transform: scale(1); opacity: 0.85; }
    72%% { transform: scale(2.25); opacity: 0; }
    100%% { transform: scale(2.25); opacity: 0; }
  }
  .vitrine-discord-ring {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-discord-pulse 2.2s ease-out infinite;
  }
  %s
</style>

<!-- Single 1.5px Catppuccin gradient outer border -->
<rect x="1" y="1" width="838" height="66" rx="12" fill="url(#card-bg)" stroke="url(#cat-accent)" stroke-width="1.5"/>

<!-- Unframed 28x28 Discord Clyde icon in AccentPrimary -->
%s

<!-- Community title and tagline -->
<g>
  <text x="66" y="%s" font-family="%s" font-size="16" font-weight="700" letter-spacing="-0.1" fill="%s">%s</text>
  %s
</g>

<!-- Live AccentGreen presence sonar and Fira Code member telemetry -->
<g>
  <circle cx="%s" cy="%s" r="4" fill="%s" opacity="0.45" class="vitrine-discord-ring"/>
  <circle cx="%s" cy="%s" r="3.8" fill="%s"/>
  <text x="%s" y="%s" font-family="%s" font-size="12" font-weight="700" fill="%s">%s</text>
  %s
</g>

<!-- Subtle Surface1 divider and Surface0 JOIN badge -->
<line x1="%s" y1="19" x2="%s" y2="49" stroke="%s" stroke-width="1"/>
<g>
  <rect x="%d" y="19" width="%d" height="30" rx="8" fill="%s" stroke="%s" stroke-width="1"/>
  <text x="%s" y="38" text-anchor="middle" font-family="%s" font-size="11" font-weight="700" letter-spacing="0.8" fill="%s">%s</text>
</g>
</svg>`,
		escapeXML(aria),
		r.baseDefs(""),
		reducedMotionCSS,
		renderNavIcon("discord", p.AccentPrimary, 24, 20, 28),
		titleY, fontSans, escapeXML(p.Text), escapeXML(titleText),
		taglineSVG,
		formatFloat(beaconCX), formatFloat(beaconCY), escapeXML(p.AccentGreen),
		formatFloat(beaconCX), formatFloat(beaconCY), escapeXML(p.AccentGreen),
		formatFloat(statsLeftX+16), formatFloat(onlineY), fontMono, escapeXML(p.Text), escapeXML(onlineText),
		membersSVG,
		formatFloat(sepX), formatFloat(sepX), escapeXML(p.Surface1),
		badgeX, badgeW, escapeXML(p.Surface0), escapeXML(p.AccentPrimary),
		formatFloat(badgeCenterX), fontMono, escapeXML(p.AccentPrimary), escapeXML(cta),
	)

	return []byte(out), nil
}

// RenderSectionDivider renders an 840x42 Catppuccin section divider banner
// with a single 1.5px border, centred bold uppercase Work Sans title, widely
// spaced AccentPrimary and AccentTertiary circle pips, and outward-flowing
// Catppuccin gradient flanking rules.
func (r *Renderer) RenderSectionDivider(data SectionData) ([]byte, error) {
	p := r.palette
	upper := strings.ToUpper(TruncateText(fallbackString(data.Title, "SECTION"), 36))
	subtitle := strings.ToUpper(TruncateText(strings.TrimSpace(data.Subtitle), 28))

	halfLabelWidth := float64(len([]rune(upper)))*11.2/2.0 + 34.0
	leftDotX := 420.0 - halfLabelWidth
	rightDotX := 420.0 + halfLabelWidth
	leftRuleEnd := leftDotX - 14.0
	rightRuleStart := rightDotX + 14.0

	rightRuleEnd := 812.0
	subtitleSVG := ""
	if subtitle != "" {
		subWidth := float64(utf8.RuneCountInString(subtitle))*6.5 + 24.0
		rightRuleEnd = 816.0 - subWidth
		subtitleSVG = fmt.Sprintf(
			`<text x="820" y="24.5" text-anchor="end" font-family="%s" font-size="9.5" letter-spacing="1.1" fill="%s">%s</text>`,
			fontMono, escapeXML(p.Muted), escapeXML(subtitle),
		)
	}

	var leftRuleSVG strings.Builder
	if leftRuleEnd > 36.0 {
		leftLen := leftRuleEnd - 28.0
		fmt.Fprintf(&leftRuleSVG,
			`<line x1="28" y1="21" x2="%s" y2="21" stroke="%s" stroke-width="1" opacity="0.65"/>
<line x1="28" y1="21" x2="%s" y2="21" stroke="url(#cat-accent)" stroke-width="1.5" class="vitrine-rule-trace vitrine-rule-left" pathLength="%s"/>
<circle cx="22" cy="21" r="3" fill="%s"/>`,
			formatFloat(leftRuleEnd), escapeXML(p.Surface1),
			formatFloat(leftRuleEnd), formatFloat(leftLen),
			escapeXML(p.AccentPrimary),
		)
	}

	var rightRuleSVG strings.Builder
	if rightRuleEnd >= rightRuleStart+16.0 {
		rightLen := rightRuleEnd - rightRuleStart
		fmt.Fprintf(&rightRuleSVG,
			`<line x1="%s" y1="21" x2="%s" y2="21" stroke="%s" stroke-width="1" opacity="0.65"/>
<line x1="%s" y1="21" x2="%s" y2="21" stroke="url(#cat-accent)" stroke-width="1.5" class="vitrine-rule-trace vitrine-rule-right" pathLength="%s"/>`,
			formatFloat(rightRuleStart), formatFloat(rightRuleEnd), escapeXML(p.Surface1),
			formatFloat(rightRuleStart), formatFloat(rightRuleEnd), formatFloat(rightLen),
		)
		if subtitle == "" {
			fmt.Fprintf(&rightRuleSVG, "\n"+`<circle cx="%s" cy="21" r="3" fill="%s"/>`, formatFloat(rightRuleEnd+6), escapeXML(p.AccentTertiary))
		}
	}

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="840" height="42" viewBox="0 0 840 42" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-divider-out-left {
    0%% { stroke-dashoffset: -64; opacity: 0.7; }
    50%% { stroke-dashoffset: 0; opacity: 1; }
    100%% { stroke-dashoffset: 64; opacity: 0.7; }
  }
  @keyframes vitrine-divider-out-right {
    0%% { stroke-dashoffset: 64; opacity: 0.7; }
    50%% { stroke-dashoffset: 0; opacity: 1; }
    100%% { stroke-dashoffset: -64; opacity: 0.7; }
  }
  @keyframes vitrine-divider-dot-pulse {
    0%%, 100%% { transform: scale(0.9); opacity: 0.65; }
    50%% { transform: scale(1.18); opacity: 1; }
  }
  .vitrine-rule-left {
    stroke-dasharray: 48 16;
    animation: vitrine-divider-out-left 6.5s linear infinite;
  }
  .vitrine-rule-right {
    stroke-dasharray: 48 16;
    animation: vitrine-divider-out-right 6.5s linear infinite;
  }
  .vitrine-divider-dot-left,
  .vitrine-divider-dot-right {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-divider-dot-pulse 3.6s ease-in-out infinite;
  }
  .vitrine-divider-dot-right {
    animation-delay: 0.6s;
  }
  %s
</style>

<!-- Single 1.5px Surface1 section divider banner -->
<rect x="1" y="4" width="838" height="34" rx="10" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Left outward-flowing gradient rule and AccentPrimary circle pip -->
%s
<circle class="vitrine-divider-dot-left" cx="%s" cy="21" r="3.5" fill="%s"/>

<!-- Centred bold uppercase Work Sans title -->
<text x="420" y="25.5" text-anchor="middle" font-family="%s" font-size="13" font-weight="700" letter-spacing="2.8" fill="%s">%s</text>

<!-- Right AccentTertiary circle pip and outward-flowing gradient rule -->
<circle class="vitrine-divider-dot-right" cx="%s" cy="21" r="3.5" fill="%s"/>
%s
%s
</svg>`,
		escapeXML(upper),
		r.baseDefs(""),
		reducedMotionCSS,
		escapeXML(p.Surface1),
		leftRuleSVG.String(),
		formatFloat(leftDotX), escapeXML(p.AccentPrimary),
		fontSans, escapeXML(p.Text), escapeXML(upper),
		formatFloat(rightDotX), escapeXML(p.AccentTertiary),
		rightRuleSVG.String(),
		subtitleSVG,
	)

	return []byte(out), nil
}
