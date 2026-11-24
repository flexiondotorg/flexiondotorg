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

const (
	sponsorsBannerWidth  = 840
	sponsorsMinHeight    = 108
	sponsorsMaxActive    = 3
	sponsorsMaxPast      = 4
	sponsorsHeartPathStr = "M0 -4.5 C-3.2 -9.5,-10.5 -8.5,-10.5 -2.2 C-10.5 3.4,-4.4 6.6,0 10 C4.4 6.6,10.5 3.4,10.5 -2.2 C10.5 -8.5,3.2 -9.5,0 -4.5 Z"
)

type sponsorHeartSpec struct {
	x         float64
	band      int
	scale     float64
	opacity   float64
	colorIdx  int
	riseDur   float64
	riseDelay float64
	swayRight bool
	swayDur   float64
	swayDelay float64
}

var sponsorHearts = []sponsorHeartSpec{
	// Brighter left-edge accent hearts (x = 12..26).
	{x: 16, band: 2, scale: 0.62, opacity: 0.72, colorIdx: 0, riseDur: 9.2, riseDelay: -2.1, swayRight: false, swayDur: 4.2, swayDelay: -1.1},
	{x: 24, band: 0, scale: 0.46, opacity: 0.58, colorIdx: 2, riseDur: 11.4, riseDelay: -7.8, swayRight: true, swayDur: 4.8, swayDelay: -2.6},
	{x: 13, band: 1, scale: 0.50, opacity: 0.66, colorIdx: 1, riseDur: 8.4, riseDelay: -5.2, swayRight: false, swayDur: 3.8, swayDelay: -0.7},

	// Softly translucent hearts drifting behind the left text block (0.14..0.24).
	{x: 78, band: 1, scale: 0.78, opacity: 0.18, colorIdx: 0, riseDur: 12.4, riseDelay: -3.4, swayRight: true, swayDur: 5.1, swayDelay: -1.8},
	{x: 152, band: 0, scale: 0.56, opacity: 0.16, colorIdx: 3, riseDur: 10.2, riseDelay: -8.1, swayRight: false, swayDur: 4.4, swayDelay: -3.0},
	{x: 232, band: 2, scale: 0.88, opacity: 0.15, colorIdx: 1, riseDur: 12.8, riseDelay: -1.5, swayRight: true, swayDur: 5.3, swayDelay: -0.9},
	{x: 318, band: 1, scale: 0.60, opacity: 0.20, colorIdx: 2, riseDur: 9.8, riseDelay: -6.2, swayRight: false, swayDur: 4.0, swayDelay: -2.2},
	{x: 404, band: 0, scale: 0.74, opacity: 0.16, colorIdx: 0, riseDur: 11.6, riseDelay: -4.9, swayRight: true, swayDur: 4.7, swayDelay: -1.5},
	{x: 482, band: 2, scale: 0.54, opacity: 0.22, colorIdx: 3, riseDur: 8.8, riseDelay: -7.1, swayRight: false, swayDur: 3.9, swayDelay: -2.8},
	{x: 552, band: 1, scale: 0.68, opacity: 0.24, colorIdx: 1, riseDur: 10.6, riseDelay: -2.8, swayRight: true, swayDur: 4.5, swayDelay: -1.2},

	// Brighter opaque hearts flanking the CTA pill and right edge (0.54..0.85).
	{x: 602, band: 0, scale: 0.72, opacity: 0.58, colorIdx: 0, riseDur: 8.6, riseDelay: -1.2, swayRight: false, swayDur: 3.7, swayDelay: -0.5},
	{x: 618, band: 2, scale: 0.52, opacity: 0.72, colorIdx: 2, riseDur: 7.6, riseDelay: -4.5, swayRight: true, swayDur: 3.4, swayDelay: -1.9},
	{x: 654, band: 0, scale: 0.86, opacity: 0.62, colorIdx: 1, riseDur: 9.4, riseDelay: -6.8, swayRight: false, swayDur: 4.3, swayDelay: -2.4},
	{x: 694, band: 2, scale: 0.64, opacity: 0.76, colorIdx: 3, riseDur: 10.8, riseDelay: -3.1, swayRight: true, swayDur: 4.6, swayDelay: -1.4},
	{x: 736, band: 1, scale: 0.96, opacity: 0.54, colorIdx: 0, riseDur: 11.8, riseDelay: -9.2, swayRight: false, swayDur: 5.0, swayDelay: -3.3},
	{x: 774, band: 0, scale: 0.58, opacity: 0.82, colorIdx: 1, riseDur: 7.4, riseDelay: -2.4, swayRight: true, swayDur: 3.5, swayDelay: -0.8},
	{x: 804, band: 2, scale: 0.82, opacity: 0.78, colorIdx: 2, riseDur: 8.9, riseDelay: -5.6, swayRight: false, swayDur: 4.1, swayDelay: -2.1},
	{x: 822, band: 1, scale: 0.54, opacity: 0.85, colorIdx: 0, riseDur: 7.8, riseDelay: -0.9, swayRight: true, swayDur: 3.6, swayDelay: -1.3},
	{x: 816, band: 0, scale: 0.66, opacity: 0.68, colorIdx: 3, riseDur: 10.0, riseDelay: -7.5, swayRight: false, swayDur: 4.4, swayDelay: -2.9},
}

func cleanSponsorNames(names []string, addAt bool) []string {
	out := make([]string, 0, len(names))
	for _, raw := range names {
		trimmed := strings.TrimSpace(raw)
		trimmed = strings.TrimPrefix(trimmed, "@")
		if trimmed == "" {
			continue
		}
		if addAt {
			out = append(out, "@"+trimmed)
		} else {
			out = append(out, trimmed)
		}
	}
	return out
}

// wrapSponsorLines packs comma-separated sponsor handles across at most
// maxLines lines without breaking individual handles. Line 0 reserves room for
// prefix. When names exceed maxLines, the final line appends ", +N more" (or
// "+N more") with the exact omitted count.
func wrapSponsorLines(prefix string, names []string, maxChars, maxLines int) []string {
	if len(names) == 0 || maxChars <= 0 || maxLines <= 0 {
		return nil
	}

	prefixLen := utf8.RuneCountInString(prefix)
	lines := make([]string, 0, maxLines)
	idx := 0

	for lineIdx := 0; lineIdx < maxLines && idx < len(names); lineIdx++ {
		budget := maxChars
		if lineIdx == 0 {
			budget = max(20, maxChars-prefixLen)
		}

		if lineIdx == maxLines-1 {
			lines = append(lines, packFinalSponsorLine(names[idx:], budget))
			return lines
		}

		line, consumed := packSponsorLine(names[idx:], budget)
		idx += consumed
		if idx < len(names) {
			line += ","
		}
		lines = append(lines, line)
	}

	return lines
}

func packSponsorLine(remaining []string, budget int) (string, int) {
	if len(remaining) == 0 {
		return "", 0
	}

	var b strings.Builder
	curLen := 0
	consumed := 0

	for i, name := range remaining {
		nameLen := utf8.RuneCountInString(name)
		reserveComma := 0
		if i < len(remaining)-1 {
			reserveComma = 1
		}

		if consumed == 0 {
			if nameLen+reserveComma <= budget {
				b.WriteString(name)
				curLen = nameLen
			} else {
				truncated := TruncateText(name, max(4, budget-reserveComma))
				b.WriteString(truncated)
				curLen = utf8.RuneCountInString(truncated)
			}
			consumed = 1
			continue
		}

		if curLen+2+nameLen+reserveComma <= budget {
			b.WriteString(", ")
			b.WriteString(name)
			curLen += 2 + nameLen
			consumed++
		} else {
			break
		}
	}

	return b.String(), consumed
}

func packFinalSponsorLine(remaining []string, budget int) string {
	full := strings.Join(remaining, ", ")
	if utf8.RuneCountInString(full) <= budget {
		return full
	}

	for k := len(remaining) - 1; k >= 1; k-- {
		suffix := fmt.Sprintf(", +%d more", len(remaining)-k)
		candidate := strings.Join(remaining[:k], ", ") + suffix
		if utf8.RuneCountInString(candidate) <= budget {
			return candidate
		}
	}

	return fmt.Sprintf("+%d more", len(remaining))
}

func renderSponsorHearts(p Palette, height int) string {
	colors := [4]string{p.AccentPink, p.AccentRed, p.AccentPeach, p.AccentPrimary}
	bandFrac := [3]float64{0.24, 0.52, 0.78}
	bandClass := [3]string{"vitrine-heart-rise-a", "vitrine-heart-rise-b", "vitrine-heart-rise-c"}

	var b strings.Builder
	for _, h := range sponsorHearts {
		staticY := float64(height) * bandFrac[h.band]
		swayClass := "vitrine-heart-sway-l"
		if h.swayRight {
			swayClass = "vitrine-heart-sway-r"
		}
		fmt.Fprintf(
			&b,
			`  <g transform="translate(%s,%s)" opacity="%s">
    <g class="%s" style="animation-duration:%ss;animation-delay:%ss">
      <g class="%s" style="animation-duration:%ss;animation-delay:%ss">
        <path d="%s" transform="scale(%s)" fill="%s"/>
      </g>
    </g>
  </g>
`,
			formatFloat(h.x), formatFloat(staticY), formatFloat(h.opacity),
			bandClass[h.band], formatFloat(h.riseDur), formatFloat(h.riseDelay),
			swayClass, formatFloat(h.swayDur), formatFloat(h.swayDelay),
			sponsorsHeartPathStr, formatFloat(h.scale), escapeXML(colors[h.colorIdx]),
		)
	}
	return b.String()
}

type sponsorsLayout struct {
	headline    string
	subLines    []string
	activeLines []string
	pastLines   []string
	height      int
	offsetY     float64
	subRelY     float64
	activeRelY  float64
	pastRelY    float64
	pillX       int
	pillW       int
}

func computeSponsorsLayout(data SponsorsData, cta string) sponsorsLayout {
	active := cleanSponsorNames(data.Active, true)
	past := cleanSponsorNames(data.Past, false)

	headline := strings.TrimSpace(data.Headline)
	subheadline := strings.TrimSpace(data.Subheadline)

	if headline == "" {
		if len(active) > 0 {
			headline = "Thank you to my sponsors"
		} else {
			headline = "Enjoy my work?"
		}
	}
	if subheadline == "" && len(active) == 0 {
		subheadline = "Support ongoing open-source tools, packages, and projects on GitHub."
	}
	headline = TruncateText(headline, 46)

	pillW := max(172, min(226, int(math.Round(float64(utf8.RuneCountInString(cta))*7.5+60))))
	pillX := 816 - pillW

	const textX = 34.0
	availTextW := float64(pillX) - 26.0 - textX
	sponsorLineChars := max(48, int(math.Floor(availTextW/7.4)))
	subLineChars := max(44, int(math.Floor(availTextW/7.8)))

	subLines := WrapText(subheadline, subLineChars, 2)
	activeLines := wrapSponsorLines("Sponsored by ", active, sponsorLineChars, sponsorsMaxActive)
	pastLines := wrapSponsorLines("Past sponsors: ", past, sponsorLineChars, sponsorsMaxPast)

	curY := 18.0
	var subRelY, activeRelY, pastRelY float64
	if len(subLines) > 0 {
		subRelY = curY + 18.0
		curY += 6.0 + float64(len(subLines))*18.0 - 4.0
	}
	if len(activeLines) > 0 {
		activeRelY = curY + 19.0
		curY += 8.0 + float64(len(activeLines))*18.0 - 4.0
	}
	if len(pastLines) > 0 {
		pastRelY = curY + 19.0
		curY += 9.0 + float64(len(pastLines))*17.0 - 4.0
	}

	height := max(sponsorsMinHeight, int(math.Ceil(curY))+44)
	offsetY := (float64(height) - curY) / 2.0

	return sponsorsLayout{
		headline:    headline,
		subLines:    subLines,
		activeLines: activeLines,
		pastLines:   pastLines,
		height:      height,
		offsetY:     offsetY,
		subRelY:     subRelY,
		activeRelY:  activeRelY,
		pastRelY:    pastRelY,
		pillX:       pillX,
		pillW:       pillW,
	}
}

func renderSponsorsText(p Palette, lay sponsorsLayout) string {
	const textX = "34"
	var b strings.Builder

	fmt.Fprintf(
		&b,
		`  <text x="%s" y="%s" font-family="%s" font-size="16" font-weight="700" letter-spacing="-0.15" fill="%s">%s</text>`+"\n",
		textX, formatFloat(lay.offsetY+14.0), fontSans, escapeXML(p.Text), escapeXML(lay.headline),
	)

	for i, line := range lay.subLines {
		y := lay.offsetY + lay.subRelY + float64(i)*18.0
		fmt.Fprintf(
			&b,
			`  <text x="%s" y="%s" font-family="%s" font-size="12.5" fill="%s">%s</text>`+"\n",
			textX, formatFloat(y), fontSans, escapeXML(p.Subtext), escapeXML(line),
		)
	}

	activeLabelColor := p.AccentPink
	if p.IsLight {
		activeLabelColor = p.AccentRed
	}
	for i, line := range lay.activeLines {
		y := lay.offsetY + lay.activeRelY + float64(i)*18.0
		if i == 0 {
			fmt.Fprintf(
				&b,
				`  <text x="%s" y="%s" font-family="%s" font-size="12" font-weight="600" fill="%s"><tspan fill="%s">Sponsored by </tspan>%s</text>`+"\n",
				textX, formatFloat(y), fontSans, escapeXML(p.Text), escapeXML(activeLabelColor), escapeXML(line),
			)
		} else {
			fmt.Fprintf(
				&b,
				`  <text x="%s" y="%s" font-family="%s" font-size="12" font-weight="600" fill="%s">%s</text>`+"\n",
				textX, formatFloat(y), fontSans, escapeXML(p.Text), escapeXML(line),
			)
		}
	}

	for i, line := range lay.pastLines {
		y := lay.offsetY + lay.pastRelY + float64(i)*17.0
		if i == 0 {
			fmt.Fprintf(
				&b,
				`  <text x="%s" y="%s" font-family="%s" font-size="11.5" fill="%s"><tspan font-weight="600" fill="%s">Past sponsors: </tspan>%s</text>`+"\n",
				textX, formatFloat(y), fontSans, escapeXML(p.Subtext), escapeXML(p.Muted), escapeXML(line),
			)
		} else {
			fmt.Fprintf(
				&b,
				`  <text x="%s" y="%s" font-family="%s" font-size="11.5" fill="%s">%s</text>`+"\n",
				textX, formatFloat(y), fontSans, escapeXML(p.Subtext), escapeXML(line),
			)
		}
	}

	return b.String()
}

// RenderSponsorsBanner renders an 840px-wide dynamic-height Catppuccin GitHub
// Sponsors banner with a neutral 1.5px Surface1 border, upward-drifting opaque
// and translucent balloon hearts, wrapped active and past sponsor lists, and a
// warm-bordered Sponsor on GitHub call-to-action pill with a pulsing heart icon.
func (r *Renderer) RenderSponsorsBanner(data SponsorsData) ([]byte, error) {
	p := r.palette
	cta := TruncateText(fallbackString(data.CTA, "Sponsor on GitHub"), 24)
	lay := computeSponsorsLayout(data, cta)

	const pillH = 38.0
	pillY := (float64(lay.height) - pillH) / 2.0
	ctaTextApproxW := float64(utf8.RuneCountInString(cta)) * 7.3
	ctaContentW := 16.0 + 8.0 + ctaTextApproxW
	ctaIconX := float64(lay.pillX) + (float64(lay.pillW)-ctaContentW)/2.0
	ctaTextX := ctaIconX + 24.0
	ctaIconY := pillY + (pillH-16.0)/2.0
	ctaTextY := pillY + 23.5

	pillBaseFill := p.Surface0
	ctaTintStart := "0.20"
	ctaTintEnd := "0.14"
	scrimOpacity := "0.52"
	if p.IsLight {
		pillBaseFill = p.Base
		ctaTintStart = "0.14"
		ctaTintEnd = "0.10"
		scrimOpacity = "0.56"
	}

	hf := float64(lay.height)
	extraDefs := fmt.Sprintf(`
  <clipPath id="sponsors-clip">
    <rect x="1.75" y="1.75" width="836.5" height="%s" rx="11.25"/>
  </clipPath>
  <linearGradient id="sponsors-text-scrim" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="78%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="100%%" stop-color="%s" stop-opacity="0"/>
  </linearGradient>
  <linearGradient id="sponsors-cta-border" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="sponsors-cta-fill" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
    <stop offset="0%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="100%%" stop-color="%s" stop-opacity="%s"/>
  </linearGradient>`,
		formatFloat(hf-3.5),
		escapeXML(p.Base), scrimOpacity, escapeXML(p.Base), scrimOpacity, escapeXML(p.Base),
		escapeXML(p.AccentPink), escapeXML(p.AccentRed),
		escapeXML(p.AccentPink), ctaTintStart, escapeXML(p.AccentRed), ctaTintEnd,
	)

	scrimW := float64(lay.pillX) - 18.0
	aria := fmt.Sprintf("GitHub Sponsors - %s", cta)

	out := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">
%s
<style>
  @keyframes vitrine-heart-rise-a {
    0%% { transform: translateY(%spx); opacity: 0; }
    10%% { opacity: 1; }
    88%% { opacity: 1; }
    100%% { transform: translateY(%spx); opacity: 0; }
  }
  @keyframes vitrine-heart-rise-b {
    0%% { transform: translateY(%spx); opacity: 0; }
    10%% { opacity: 1; }
    88%% { opacity: 1; }
    100%% { transform: translateY(%spx); opacity: 0; }
  }
  @keyframes vitrine-heart-rise-c {
    0%% { transform: translateY(%spx); opacity: 0; }
    10%% { opacity: 1; }
    88%% { opacity: 1; }
    100%% { transform: translateY(%spx); opacity: 0; }
  }
  @keyframes vitrine-heart-sway-l {
    0%%, 100%% { transform: translateX(-6px) rotate(-5deg); }
    50%% { transform: translateX(6px) rotate(5deg); }
  }
  @keyframes vitrine-heart-sway-r {
    0%%, 100%% { transform: translateX(6px) rotate(5deg); }
    50%% { transform: translateX(-6px) rotate(-5deg); }
  }
  @keyframes vitrine-heart-beat {
    0%%, 100%% { transform: scale(1); }
    10%% { transform: scale(1.22); }
    20%% { transform: scale(1); }
    30%% { transform: scale(1.15); }
    44%% { transform: scale(1); }
  }
  .vitrine-heart-rise-a { animation: vitrine-heart-rise-a 10s linear infinite; }
  .vitrine-heart-rise-b { animation: vitrine-heart-rise-b 10s linear infinite; }
  .vitrine-heart-rise-c { animation: vitrine-heart-rise-c 10s linear infinite; }
  .vitrine-heart-sway-l { animation: vitrine-heart-sway-l 4.2s ease-in-out infinite; }
  .vitrine-heart-sway-r { animation: vitrine-heart-sway-r 4.2s ease-in-out infinite; }
  .vitrine-heart-beat {
    transform-box: fill-box;
    transform-origin: center;
    animation: vitrine-heart-beat 2.6s ease-in-out infinite;
  }
  %s
</style>

<!-- Single 1.5px neutral Surface1 outer border -->
<rect x="1" y="1" width="838" height="%d" rx="12" fill="url(#card-bg)" stroke="%s" stroke-width="1.5"/>

<!-- Upward-drifting Catppuccin balloon hearts clipped to card bounds -->
<g clip-path="url(#sponsors-clip)" class="vitrine-sponsors-hearts">
%s</g>

<!-- Soft readability scrim behind left text column -->
<rect x="24" y="6" width="%s" height="%d" rx="10" fill="url(#sponsors-text-scrim)"/>

<!-- Left sponsor headline and wrapped sponsor lists -->
<g class="vitrine-sponsors-text">
%s</g>

<!-- Right Sponsor on GitHub call-to-action pill -->
<g class="vitrine-sponsors-cta">
  <rect x="%d" y="%s" width="%d" height="%s" rx="19" fill="%s"/>
  <rect x="%d" y="%s" width="%d" height="%s" rx="19" fill="url(#sponsors-cta-fill)" stroke="url(#sponsors-cta-border)" stroke-width="1.5"/>
  <svg x="%s" y="%s" width="16" height="16" viewBox="0 0 24 24" overflow="visible"><path class="vitrine-heart-beat" d="%s" fill="%s"/></svg>
  <text x="%s" y="%s" text-anchor="start" font-family="%s" font-size="13" font-weight="700" letter-spacing="0.1" fill="%s">%s</text>
</g>
</svg>`,
		sponsorsBannerWidth, lay.height, sponsorsBannerWidth, lay.height, escapeXML(aria),
		r.baseDefs(extraDefs),
		formatFloat(hf*1.00), formatFloat(-hf*0.56),
		formatFloat(hf*0.72), formatFloat(-hf*0.84),
		formatFloat(hf*0.46), formatFloat(-hf*1.10),
		reducedMotionCSS,
		lay.height-2, escapeXML(p.Surface1),
		renderSponsorHearts(p, lay.height),
		formatFloat(scrimW), lay.height-12,
		renderSponsorsText(p, lay),
		lay.pillX, formatFloat(pillY), lay.pillW, formatFloat(pillH), escapeXML(pillBaseFill),
		lay.pillX, formatFloat(pillY), lay.pillW, formatFloat(pillH),
		formatFloat(ctaIconX), formatFloat(ctaIconY), navIconPaths["heart"], escapeXML(p.AccentRed),
		formatFloat(ctaTextX), formatFloat(ctaTextY), fontSans, escapeXML(p.Text), escapeXML(cta),
	)

	return []byte(out), nil
}
