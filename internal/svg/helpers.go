package svg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	fontSans = "'Work Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif"
	fontMono = "'Fira Code', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace"

	reducedMotionCSS = `@media (prefers-reduced-motion: reduce) { * { animation: none !important; } }`
)

var xmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#39;",
)

func escapeXML(s string) string {
	// Strip ASCII control characters forbidden in XML 1.0 except tab, LF, CR.
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)
	return xmlReplacer.Replace(cleaned)
}

func fallbackString(val, def string) string {
	if strings.TrimSpace(val) == "" {
		return def
	}
	return val
}

func formatFloat(v float64) string {
	if math.Abs(v-math.Round(v)) < 1e-6 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// TruncateText shortens text to at most maxRunes runes, appending an ellipsis
// when truncation occurs.
func TruncateText(text string, maxRunes int) string {
	clean := normalizeSpaces(text)
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(clean)
	if len(runes) <= maxRunes {
		return clean
	}
	if maxRunes <= 1 {
		return "…"
	}
	return strings.TrimRightFunc(string(runes[:maxRunes-1]), unicode.IsSpace) + "…"
}

// WrapText wraps text cleanly on word boundaries into at most maxLines lines,
// each containing at most maxChars runes. Overflow on the final line is
// truncated with an ellipsis.
func WrapText(text string, maxChars, maxLines int) []string {
	clean := normalizeSpaces(text)
	if clean == "" || maxChars <= 0 || maxLines <= 0 {
		return nil
	}

	words := strings.Fields(clean)
	var lines []string
	var current string

	for i, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if current == "" {
			if wordLen <= maxChars {
				current = word
				continue
			}
			// Single word longer than a full line.
			if len(lines) == maxLines-1 {
				lines = append(lines, TruncateText(word, maxChars))
				return lines
			}
			runes := []rune(word)
			lines = append(lines, string(runes[:maxChars]))
			rest := string(runes[maxChars:])
			if rest != "" {
				current = TruncateText(rest, maxChars)
			}
			continue
		}

		candidate := current + " " + word
		if utf8.RuneCountInString(candidate) <= maxChars {
			current = candidate
			continue
		}

		if len(lines) == maxLines-1 {
			remaining := current + " " + strings.Join(words[i:], " ")
			lines = append(lines, TruncateText(remaining, maxChars))
			return lines
		}

		lines = append(lines, current)
		if wordLen <= maxChars {
			current = word
		} else {
			current = TruncateText(word, maxChars)
		}
	}

	if current != "" && len(lines) < maxLines {
		lines = append(lines, current)
	}
	return lines
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// FormatNumber formats an integer with comma thousands separators.
func FormatNumber(n int) string {
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

// FormatCompactCount formats counts compactly for tight card footers
// (for example, 810 -> "810", 1420 -> "1.4k").
func FormatCompactCount(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 1000000 {
		val := float64(n) / 1000.0
		if val >= 100 {
			return fmt.Sprintf("%.0fk", val)
		}
		return strings.TrimSuffix(fmt.Sprintf("%.1fk", val), ".0k")
	}
	val := float64(n) / 1000000.0
	return strings.TrimSuffix(fmt.Sprintf("%.1fm", val), ".0m")
}

// baseDefs emits the shared Catppuccin gradients used across every tile.
func (r *Renderer) baseDefs(extra string) string {
	p := r.palette
	sheenPeak := "0.34"
	sheenAccent := "0.14"
	if p.IsLight {
		sheenPeak = "0.65"
		sheenAccent = "0.18"
	}

	return fmt.Sprintf(`<defs>
  <linearGradient id="card-bg" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="cat-accent" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="50%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="cat-aurora" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s"/>
    <stop offset="25%%" stop-color="%s"/>
    <stop offset="50%%" stop-color="%s"/>
    <stop offset="75%%" stop-color="%s"/>
    <stop offset="100%%" stop-color="%s"/>
  </linearGradient>
  <linearGradient id="cat-sheen" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="#ffffff" stop-opacity="0"/>
    <stop offset="32%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="50%%" stop-color="#ffffff" stop-opacity="%s"/>
    <stop offset="68%%" stop-color="%s" stop-opacity="%s"/>
    <stop offset="100%%" stop-color="#ffffff" stop-opacity="0"/>
  </linearGradient>
  <linearGradient id="cat-glint" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
    <stop offset="0%%" stop-color="%s" stop-opacity="0"/>
    <stop offset="28%%" stop-color="%s" stop-opacity="0.88"/>
    <stop offset="50%%" stop-color="#ffffff" stop-opacity="0.96"/>
    <stop offset="72%%" stop-color="%s" stop-opacity="0.88"/>
    <stop offset="100%%" stop-color="%s" stop-opacity="0"/>
  </linearGradient>%s
</defs>`,
		escapeXML(p.Base), escapeXML(p.Mantle),
		escapeXML(p.AccentPrimary), escapeXML(p.AccentSecondary), escapeXML(p.AccentTertiary),
		escapeXML(p.AccentPink), escapeXML(p.AccentPrimary), escapeXML(p.AccentSecondary), escapeXML(p.AccentTertiary), escapeXML(p.AccentGreen),
		escapeXML(p.AccentPrimary), sheenAccent, sheenPeak, escapeXML(p.AccentPrimary), sheenAccent,
		escapeXML(p.AccentPrimary), escapeXML(p.AccentPrimary), escapeXML(p.AccentTertiary), escapeXML(p.AccentTertiary),
		extra,
	)
}

// cornerStarPath builds a 4-point star centered at (cx, cy).
func cornerStarPath(cx, cy, outerR, innerR float64) string {
	return fmt.Sprintf(
		"M%s %s L%s %s L%s %s L%s %s L%s %s L%s %s L%s %s L%s %s Z",
		formatFloat(cx), formatFloat(cy-outerR),
		formatFloat(cx+innerR), formatFloat(cy-innerR),
		formatFloat(cx+outerR), formatFloat(cy),
		formatFloat(cx+innerR), formatFloat(cy+innerR),
		formatFloat(cx), formatFloat(cy+outerR),
		formatFloat(cx-innerR), formatFloat(cy+innerR),
		formatFloat(cx-outerR), formatFloat(cy),
		formatFloat(cx-innerR), formatFloat(cy-innerR),
	)
}

// renderLogoArt renders unframed project or package-host artwork in fallback
// order: inline SVG -> base64 data URI -> animated Catppuccin Jive waveform ->
// modern Catppuccin monogram badge.
func (r *Renderer) renderLogoArt(slug, name, logoSVG, logoDataURI string, x, y, size float64) string {
	if strings.TrimSpace(logoSVG) != "" {
		if sanitized, err := SanitizeInlineSVG(logoSVG, x, y, size); err == nil {
			return sanitized
		}
	}
	if strings.TrimSpace(logoDataURI) != "" {
		if img, err := RenderDataURIImage(logoDataURI, x, y, size); err == nil {
			return img
		}
	}
	if isJiveSlug(slug) {
		return r.renderJiveWaveform(x, y, size)
	}
	return r.renderMonogram(slug, name, x, y, size)
}

func isJiveSlug(slug string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(slug)), "jive-")
}

// renderJiveWaveform draws an unframed animated 5-bar Catppuccin acoustic
// waveform emblem for linuxmatters/jive-* projects.
func (r *Renderer) renderJiveWaveform(x, y, size float64) string {
	p := r.palette
	cx := x + size/2
	cy := y + size/2
	barW := size * 0.115
	gap := size * 0.068
	totalW := 5*barW + 4*gap
	startX := cx - totalW/2
	heights := []float64{size * 0.36, size * 0.66, size * 0.88, size * 0.60, size * 0.40}
	colors := []string{p.AccentPink, p.AccentPrimary, p.AccentSecondary, p.AccentSapphire, p.AccentTertiary}
	delays := []string{"0s", "0.2s", "0.4s", "0.6s", "0.8s"}

	var b strings.Builder
	b.WriteString(`<g class="vitrine-jive-crest">`)
	for i := range 5 {
		bx := startX + float64(i)*(barW+gap)
		bh := heights[i]
		by := cy - bh/2
		fmt.Fprintf(
			&b,
			`<rect class="vitrine-wave-bar" x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s" style="animation-delay:%s"/>`,
			formatFloat(bx), formatFloat(by), formatFloat(barW), formatFloat(bh), formatFloat(barW/2), escapeXML(colors[i]), delays[i],
		)
	}
	b.WriteString(`</g>`)
	return b.String()
}

// renderMonogram draws a clean modern Catppuccin rounded-square monogram badge
// when no project or host logo is available.
func (r *Renderer) renderMonogram(slug, name string, x, y, size float64) string {
	p := r.palette
	cx := x + size/2
	cy := y + size/2
	initials := deriveMonogram(slug, name)
	fontSize := size * 0.36
	if utf8.RuneCountInString(initials) == 1 {
		fontSize = size * 0.44
	}
	radius := size * 0.22

	return fmt.Sprintf(
		`<g class="vitrine-monogram">
  <rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s" stroke="url(#cat-accent)" stroke-width="1.2"/>
  <text x="%s" y="%s" text-anchor="middle" dominant-baseline="central" font-family="%s" font-size="%s" font-weight="700" letter-spacing="0.6" fill="%s">%s</text>
</g>`,
		formatFloat(x), formatFloat(y), formatFloat(size), formatFloat(size), formatFloat(radius), escapeXML(p.Surface0),
		formatFloat(cx), formatFloat(cy+size*0.02),
		fontSans, formatFloat(fontSize),
		escapeXML(p.AccentPrimary), escapeXML(initials),
	)
}

func deriveMonogram(slug, name string) string {
	src := strings.TrimSpace(name)
	if src == "" {
		src = strings.TrimSpace(slug)
	}
	if src == "" {
		return "V"
	}

	tokens := strings.FieldsFunc(src, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(tokens) >= 2 {
		r1, _ := utf8.DecodeRuneInString(tokens[0])
		r2, _ := utf8.DecodeRuneInString(tokens[1])
		return strings.ToUpper(string([]rune{r1, r2}))
	}
	if len(tokens) == 1 {
		runes := []rune(tokens[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[:2]))
		}
		return strings.ToUpper(string(runes[0]))
	}
	return "V"
}

// languageColor returns a GitHub-inspired accent colour for common primary
// programming languages, falling back to fallback when unknown.
func languageColor(lang, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go":
		return "#00add8"
	case "rust":
		return "#dea584"
	case "shell", "bash", "sh":
		return "#89e051"
	case "python":
		return "#4b8bbe"
	case "typescript":
		return "#3178c6"
	case "javascript":
		return "#f1e05a"
	case "nix":
		return "#7e7bae"
	case "c":
		return "#8b949e"
	case "c++":
		return "#f34b7d"
	case "lua":
		return "#51a0cf"
	case "zig":
		return "#ec915c"
	default:
		return fallback
	}
}

// navPlatformAccent returns the Catppuccin accent colour mapped to a social or
// feed navigation platform icon key.
func navPlatformAccent(icon string, p Palette) string {
	switch strings.ToLower(strings.TrimSpace(icon)) {
	case "mastodon", "apple-podcasts", "applepodcasts":
		return p.AccentPrimary
	case "bluesky":
		return p.AccentSecondary
	case "linkedin":
		return p.AccentSapphire
	case "spotify":
		return p.AccentGreen
	case "youtube":
		return p.AccentRed
	case "rss":
		return p.AccentPeach
	default:
		return p.AccentPrimary
	}
}

const applePodcastsIconPath = "M5.34 0A5.328 5.328 0 000 5.34v13.32A5.328 5.328 0 005.34 24h13.32A5.328 5.328 0 0024 18.66V5.34A5.328 5.328 0 0018.66 0zm6.525 2.568c2.336 0 4.448.902 6.056 2.587 1.224 1.272 1.912 2.619 2.264 4.392.12.59.12 2.2.007 2.864a8.506 8.506 0 01-3.24 5.296c-.608.46-2.096 1.261-2.336 1.261-.088 0-.096-.091-.056-.46.072-.592.144-.715.48-.856.536-.224 1.448-.874 2.008-1.435a7.644 7.644 0 002.008-3.536c.208-.824.184-2.656-.048-3.504-.728-2.696-2.928-4.792-5.624-5.352-.784-.16-2.208-.16-3 0-2.728.56-4.984 2.76-5.672 5.528-.184.752-.184 2.584 0 3.336.456 1.832 1.64 3.512 3.192 4.512.304.2.672.408.824.472.336.144.408.264.472.856.04.36.03.464-.056.464-.056 0-.464-.176-.896-.384l-.04-.03c-2.472-1.216-4.056-3.274-4.632-6.012-.144-.706-.168-2.392-.03-3.04.36-1.74 1.048-3.1 2.192-4.304 1.648-1.737 3.768-2.656 6.128-2.656zm.134 2.81c.409.004.803.04 1.106.106 2.784.62 4.76 3.408 4.376 6.174-.152 1.114-.536 2.03-1.216 2.88-.336.43-1.152 1.15-1.296 1.15-.023 0-.048-.272-.048-.603v-.605l.416-.496c1.568-1.878 1.456-4.502-.256-6.224-.664-.67-1.432-1.064-2.424-1.246-.64-.118-.776-.118-1.448-.008-1.02.167-1.81.562-2.512 1.256-1.72 1.704-1.832 4.342-.264 6.222l.413.496v.608c0 .336-.027.608-.06.608-.03 0-.264-.16-.512-.36l-.034-.011c-.832-.664-1.568-1.842-1.872-2.997-.184-.698-.184-2.024.008-2.72.504-1.878 1.888-3.335 3.808-4.019.41-.145 1.133-.22 1.814-.211zm-.13 2.99c.31 0 .62.06.844.178.488.253.888.745 1.04 1.259.464 1.578-1.208 2.96-2.72 2.254h-.015c-.712-.331-1.096-.956-1.104-1.77 0-.733.408-1.371 1.112-1.745.224-.117.534-.176.844-.176zm-.011 4.728c.988-.004 1.706.349 1.97.97.198.464.124 1.932-.218 4.302-.232 1.656-.36 2.074-.68 2.356-.44.39-1.064.498-1.656.288h-.003c-.716-.257-.87-.605-1.164-2.644-.341-2.37-.416-3.838-.218-4.302.262-.616.974-.966 1.97-.97z"

// navIconPaths maps icon keys to authentic 24x24 single-path brand and Primer
// Octicon vector shapes (Simple Icons for brand marks, Primer Octicons for
// general symbols).
var navIconPaths = map[string]string{
	"mastodon":       "M23.268 5.313c-.35-2.578-2.617-4.61-5.304-5.004C17.51.242 15.792 0 11.813 0h-.03c-3.98 0-4.835.242-5.288.309C3.882.692 1.496 2.518.917 5.127.64 6.412.61 7.837.661 9.143c.074 1.874.088 3.745.26 5.611.118 1.24.325 2.47.62 3.68.55 2.237 2.777 4.098 4.96 4.857 2.336.792 4.849.923 7.256.38.265-.061.527-.132.786-.213.585-.184 1.27-.39 1.774-.753a.057.057 0 0 0 .023-.043v-1.809a.052.052 0 0 0-.02-.041.053.053 0 0 0-.046-.01 20.282 20.282 0 0 1-4.709.545c-2.73 0-3.463-1.284-3.674-1.818a5.593 5.593 0 0 1-.319-1.433.053.053 0 0 1 .066-.054c1.517.363 3.072.546 4.632.546.376 0 .75 0 1.125-.01 1.57-.044 3.224-.124 4.768-.422.038-.008.077-.015.11-.024 2.435-.464 4.753-1.92 4.989-5.604.008-.145.03-1.52.03-1.67.002-.512.167-3.63-.024-5.545zm-3.748 9.195h-2.561V8.29c0-1.309-.55-1.976-1.67-1.976-1.23 0-1.846.79-1.846 2.35v3.403h-2.546V8.663c0-1.56-.617-2.35-1.848-2.35-1.112 0-1.668.668-1.67 1.977v6.218H4.822V8.102c0-1.31.337-2.35 1.011-3.12.696-.77 1.608-1.164 2.74-1.164 1.311 0 2.302.5 2.962 1.498l.638 1.06.638-1.06c.66-.999 1.65-1.498 2.96-1.498 1.13 0 2.043.395 2.74 1.164.675.77 1.012 1.81 1.012 3.12z",
	"bluesky":        "M5.202 2.857C7.954 4.922 10.913 9.11 12 11.358c1.087-2.247 4.046-6.436 6.798-8.501C20.783 1.366 24 .213 24 3.883c0 .732-.42 6.156-.667 7.037-.856 3.061-3.978 3.842-6.755 3.37 4.854.826 6.089 3.562 3.422 6.299-5.065 5.196-7.28-1.304-7.847-2.97-.104-.305-.152-.448-.153-.327 0-.121-.05.022-.153.327-.568 1.666-2.782 8.166-7.847 2.97-2.667-2.737-1.432-5.473 3.422-6.3-2.777.473-5.899-.308-6.755-3.369C.42 10.04 0 4.615 0 3.883c0-3.67 3.217-2.517 5.202-1.026",
	"linkedin":       "M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z",
	"discord":        "M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z",
	"twitch":         "M11.571 4.714h1.715v5.143H11.57zm4.715 0H18v5.143h-1.714zM6 0L1.714 4.286v15.428h5.143V24l4.286-4.286h3.428L22.286 12V0zm14.571 11.143l-3.428 3.428h-3.429l-3 3v-3H6.857V1.714h13.714Z",
	"youtube":        "M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z",
	"apple-podcasts": applePodcastsIconPath,
	"applepodcasts":  applePodcastsIconPath,
	"spotify":        "M12 0C5.4 0 0 5.4 0 12s5.4 12 12 12 12-5.4 12-12S18.66 0 12 0zm5.521 17.34c-.24.359-.66.48-1.021.24-2.82-1.74-6.36-2.101-10.561-1.141-.418.122-.779-.179-.899-.539-.12-.421.18-.78.54-.9 4.56-1.021 8.52-.6 11.64 1.32.42.18.479.659.301 1.02zm1.44-3.3c-.301.42-.841.6-1.262.3-3.239-1.98-8.159-2.58-11.939-1.38-.479.12-1.02-.12-1.14-.6-.12-.48.12-1.021.6-1.141C9.6 9.9 15 10.561 18.72 12.84c.361.181.54.78.241 1.2zm.12-3.36C15.24 8.4 8.82 8.16 5.16 9.301c-.6.179-1.2-.181-1.38-.721-.18-.601.18-1.2.72-1.381 4.26-1.26 11.28-1.02 15.721 1.621.539.3.719 1.02.419 1.56-.299.421-1.02.599-1.559.3z",
	"rss":            "M19.199 24C19.199 13.467 10.533 4.8 0 4.8V0c13.165 0 24 10.835 24 24h-4.801zM3.291 17.415c1.814 0 3.293 1.479 3.293 3.295 0 1.813-1.485 3.29-3.301 3.29C1.47 24 0 22.526 0 20.71s1.475-3.294 3.291-3.295zM15.909 24h-4.665c0-6.169-5.075-11.245-11.244-11.245V8.09c8.727 0 15.909 7.184 15.909 15.91z",
	"github":         "M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12",
	"podcast":        "M20.485 2.515a.75.75 0 0 0-1.06 1.06A10.465 10.465 0 0 1 22.5 11c0 2.9-1.174 5.523-3.075 7.424a.75.75 0 0 0 1.06 1.061A11.965 11.965 0 0 0 24 11c0-3.314-1.344-6.315-3.515-8.485Zm-15.91 1.06a.75.75 0 0 0-1.06-1.06A11.965 11.965 0 0 0 0 11c0 3.313 1.344 6.314 3.515 8.485a.75.75 0 0 0 1.06-1.06A10.465 10.465 0 0 1 1.5 11c0-2.9 1.174-5.524 3.075-7.425ZM8.11 7.11a.75.75 0 0 0-1.06-1.06A6.98 6.98 0 0 0 5 11a6.98 6.98 0 0 0 2.05 4.95.75.75 0 0 0 1.06-1.061 5.48 5.48 0 0 1-1.61-3.89 5.48 5.48 0 0 1 1.61-3.888Zm8.84-1.06a.75.75 0 1 0-1.06 1.06A5.48 5.48 0 0 1 17.5 11a5.48 5.48 0 0 1-1.61 3.889.75.75 0 1 0 1.06 1.06A6.98 6.98 0 0 0 19 11a6.98 6.98 0 0 0-2.05-4.949ZM14 11a2 2 0 0 1-1.25 1.855v8.395a.75.75 0 0 1-1.5 0v-8.395A2 2 0 1 1 14 11Z",
	"globe":          "M12 1c6.075 0 11 4.925 11 11s-4.925 11-11 11S1 18.075 1 12 5.925 1 12 1Zm3.241 10.5v-.001c-.1-2.708-.992-4.904-1.89-6.452a13.919 13.919 0 0 0-1.304-1.88L12 3.11l-.047.059c-.354.425-.828 1.06-1.304 1.88-.898 1.547-1.79 3.743-1.89 6.451Zm-12.728 0h4.745c.1-3.037 1.1-5.49 2.093-7.204.39-.672.78-1.233 1.119-1.673C6.11 3.329 2.746 7 2.513 11.5Zm18.974 0C21.254 7 17.89 3.329 13.53 2.623c.339.44.729 1.001 1.119 1.673.993 1.714 1.993 4.167 2.093 7.204ZM8.787 13c.182 2.478 1.02 4.5 1.862 5.953.382.661.818 1.29 1.304 1.88l.047.057.047-.059c.354-.425.828-1.06 1.304-1.88.842-1.451 1.679-3.471 1.862-5.951Zm-1.504 0H2.552a9.505 9.505 0 0 0 7.918 8.377 15.773 15.773 0 0 1-1.119-1.673C8.413 18.085 7.47 15.807 7.283 13Zm9.434 0c-.186 2.807-1.13 5.085-2.068 6.704-.39.672-.78 1.233-1.118 1.673A9.506 9.506 0 0 0 21.447 13Z",
	"blog":           "M0 3.75A.75.75 0 0 1 .75 3h7.497c1.566 0 2.945.8 3.751 2.014A4.495 4.495 0 0 1 15.75 3h7.5a.75.75 0 0 1 .75.75v15.063a.752.752 0 0 1-.755.75l-7.682-.052a3 3 0 0 0-2.142.878l-.89.891a.75.75 0 0 1-1.061 0l-.902-.901a2.996 2.996 0 0 0-2.121-.879H.75a.75.75 0 0 1-.75-.75Zm12.75 15.232a4.503 4.503 0 0 1 2.823-.971l6.927.047V4.5h-6.75a3 3 0 0 0-3 3ZM11.247 7.497a3 3 0 0 0-3-2.997H1.5V18h6.947c1.018 0 2.006.346 2.803.98Z",
	"heart":          "M14 20.408c-.492.308-.903.546-1.192.709-.153.086-.308.17-.463.252h-.002a.75.75 0 0 1-.686 0 16.709 16.709 0 0 1-.465-.252 31.147 31.147 0 0 1-4.803-3.34C3.8 15.572 1 12.331 1 8.513 1 5.052 3.829 2.5 6.736 2.5 9.03 2.5 10.881 3.726 12 5.605 13.12 3.726 14.97 2.5 17.264 2.5 20.17 2.5 23 5.052 23 8.514c0 3.818-2.801 7.06-5.389 9.262A31.146 31.146 0 0 1 14 20.408Z",
}

// renderNavIcon draws a crisp authentic 24x24 inline vector brand or symbol
// icon inside a nested SVG box positioned at (x, y) with width and height size.
func renderNavIcon(icon, color string, x, y, size float64) string {
	c := escapeXML(color)
	pathData, ok := navIconPaths[strings.ToLower(strings.TrimSpace(icon))]
	if !ok {
		pathData = cornerStarPath(12, 12, 9.2, 3.3)
	}

	return fmt.Sprintf(
		`<svg x="%s" y="%s" width="%s" height="%s" viewBox="0 0 24 24"><path d="%s" fill="%s"/></svg>`,
		formatFloat(x), formatFloat(y), formatFloat(size), formatFloat(size), pathData, c,
	)
}
