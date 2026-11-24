package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/flexiondotorg/flexiondotorg/internal/render"
)

var (
	mdLinkRe    = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	mdBoldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	mdItalicRe  = regexp.MustCompile(`\*([^*]+)\*`)
	mdCodeRe    = regexp.MustCompile("`([^`]+)`")
	pictureRe   = regexp.MustCompile(`<picture>\s*<source\s+media="\(prefers-color-scheme:\s*dark\)"\s+srcset="([^"]+)"\s*/?>\s*(<img\b[^>]*>)\s*</picture>`)
	imgSrcRe    = regexp.MustCompile(`\bsrc="([^"]+)"`)
	lightFileRe = regexp.MustCompile(`-light\.svg"`)
)

func buildPreviewHTML(markdown string, data render.RenderData, twitchLiveBase, twitchOfflineBase string) []byte {
	bodyHTML := markdownToHTML(markdown)
	darkBodyHTML := resolvePreviewTheme(bodyHTML, true)
	lightBodyHTML := resolvePreviewTheme(bodyHTML, false)

	title := "Vitrine Preview"
	if data.Config != nil && strings.TrimSpace(data.Config.Profile.HeroTitle) != "" {
		title = data.Config.Profile.HeroTitle + " - Vitrine Preview"
	}

	darkPrototypeStrip := buildTwitchPrototypeStrip(data, twitchLiveBase+"-dark.svg", twitchOfflineBase+"-dark.svg")
	lightPrototypeStrip := buildTwitchPrototypeStrip(data, twitchLiveBase+"-light.svg", twitchOfflineBase+"-light.svg")

	doc := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
  * { box-sizing: border-box; }
  body {
    margin: 0;
    padding: 32px 24px 48px;
    background: #161b22;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif;
    line-height: 1.5;
  }
  body.mode-dark { background: #0d1117; padding: 20px; }
  body.mode-light { background: #ffffff; padding: 20px; }
  body.mode-dark .panel-light { display: none; }
  body.mode-light .panel-dark { display: none; }
  .preview-stack {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 36px;
  }
  .gh-panel {
    width: 892px;
    max-width: 100%%;
    border-radius: 12px;
    padding: 24px 26px 28px;
  }
  .panel-dark {
    background: #0d1117;
    color: #e6edf3;
    border: 1px solid #30363d;
  }
  .panel-light {
    background: #ffffff;
    color: #1f2328;
    border: 1px solid #d0d7de;
  }
  .panel-badge {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    margin: 0 0 14px;
    opacity: 0.65;
  }
  .markdown-body {
    width: 840px;
    max-width: 100%%;
    margin: 0 auto;
    font-size: 14px;
  }
  .markdown-body p { margin: 0 0 14px; }
  .markdown-body p[align="center"] {
    text-align: center;
    line-height: 1.35;
  }
  .markdown-body p[align="center"] a {
    display: inline-block;
    margin: 3px 3px;
    vertical-align: middle;
  }
  .markdown-body img {
    max-width: 100%%;
    height: auto;
    vertical-align: middle;
  }
  .markdown-body img[align="right"] {
    float: right;
    margin: 0 0 12px 18px;
    border-radius: 10px;
  }
  .markdown-body h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 18px 0 8px;
    padding-bottom: 4px;
  }
  .panel-dark .markdown-body h3 { border-bottom: 1px solid #21262d; }
  .panel-light .markdown-body h3 { border-bottom: 1px solid #d8dee4; }
  .markdown-body ul {
    margin: 6px 0 14px;
    padding-left: 22px;
  }
  .markdown-body li { margin: 4px 0; }
  .markdown-body code {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 85%%;
    padding: 0.15em 0.4em;
    border-radius: 6px;
  }
  .panel-dark .markdown-body code { background: rgba(110, 118, 129, 0.25); color: #cba6f7; }
  .panel-light .markdown-body code { background: rgba(175, 184, 193, 0.22); color: #8839ef; }
  .panel-dark .markdown-body a { color: #58a6ff; text-decoration: none; }
  .panel-light .markdown-body a { color: #0969da; text-decoration: none; }
  .markdown-body a:hover { text-decoration: underline; }
  .markdown-body details {
    margin: 10px 0;
    padding: 8px 12px;
    border-radius: 6px;
  }
  .panel-dark .markdown-body details { background: #161b22; border: 1px solid #30363d; }
  .panel-light .markdown-body details { background: #f6f8fa; border: 1px solid #d0d7de; }
  .markdown-body summary { cursor: pointer; }
  .panel-dark .markdown-body sub { color: #8b949e; }
  .panel-light .markdown-body sub { color: #656d76; }
  .twitch-prototype {
    margin-top: 22px;
    padding-top: 14px;
    text-align: center;
  }
  .panel-dark .twitch-prototype { border-top: 1px dashed #30363d; }
  .panel-light .twitch-prototype { border-top: 1px dashed #d0d7de; }
  .twitch-prototype-label {
    font-size: 11px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    opacity: 0.7;
    margin-bottom: 10px;
  }
  .twitch-prototype-row {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 14px;
    flex-wrap: wrap;
  }
</style>
</head>
<body>
<div class="preview-stack">
  <section id="github-dark" class="gh-panel panel-dark">
    <div class="panel-badge">GitHub Dark (#0d1117)</div>
    <article class="markdown-body">
%s
%s
    </article>
  </section>
  <section id="github-light" class="gh-panel panel-light">
    <div class="panel-badge">GitHub Light (#ffffff)</div>
    <article class="markdown-body">
%s
%s
    </article>
  </section>
</div>
<script>
  (function () {
    var params = new URLSearchParams(window.location.search);
    var mode = (params.get("mode") || window.location.hash.replace(/^#/, "")).toLowerCase();
    if (mode === "dark") {
      document.body.classList.add("mode-dark");
    } else if (mode === "light") {
      document.body.classList.add("mode-light");
    }
  })();
</script>
</body>
</html>
`, html.EscapeString(title), darkBodyHTML, darkPrototypeStrip, lightBodyHTML, lightPrototypeStrip)

	return []byte(doc)
}

func buildTwitchPrototypeStrip(data render.RenderData, liveFile, offlineFile string) string {
	liveSrc := render.JoinAssetURL(data.AssetBaseURL, liveFile)
	offlineSrc := render.JoinAssetURL(data.AssetBaseURL, offlineFile)
	twitchURL := "#"
	if data.Config != nil && strings.TrimSpace(data.Config.Twitch.URL) != "" {
		twitchURL = data.Config.Twitch.URL
	}

	return fmt.Sprintf(`<div class="twitch-prototype">
  <div class="twitch-prototype-label">Twitch Status Tile Prototypes (opt-in via vitrine.yaml)</div>
  <div class="twitch-prototype-row">
    <a href="%s"><img src="%s" alt="Live on Twitch prototype"></a>
    <a href="%s"><img src="%s" alt="Twitch offline prototype"></a>
  </div>
</div>`,
		html.EscapeString(twitchURL), html.EscapeString(liveSrc),
		html.EscapeString(twitchURL), html.EscapeString(offlineSrc),
	)
}

// resolvePreviewTheme flattens <picture> tags so each preview panel displays
// its own theme asset directly regardless of the viewer's OS colour scheme.
func resolvePreviewTheme(htmlBody string, dark bool) string {
	resolved := pictureRe.ReplaceAllStringFunc(htmlBody, func(match string) string {
		parts := pictureRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		darkSrc := parts[1]
		imgTag := parts[2]
		if !dark {
			return imgTag
		}
		return imgSrcRe.ReplaceAllString(imgTag, `src="`+darkSrc+`"`)
	})
	if dark {
		resolved = lightFileRe.ReplaceAllString(resolved, `-dark.svg"`)
	}
	return resolved
}

func markdownToHTML(md string) string {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var out strings.Builder
	inList := false

	closeList := func() {
		if inList {
			out.WriteString("</ul>\n")
			inList = false
		}
	}

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" {
			closeList()
			continue
		}

		if isRawHTMLLine(trimmed) {
			closeList()
			out.WriteString(rawLine)
			out.WriteByte('\n')
			continue
		}

		if level, heading, ok := parseHeading(trimmed); ok {
			closeList()
			fmt.Fprintf(&out, "<h%d>%s</h%d>\n", level, renderInlineMarkdown(heading), level)
			continue
		}

		if after, ok := strings.CutPrefix(trimmed, "- "); ok {
			if !inList {
				out.WriteString("<ul>\n")
				inList = true
			}
			fmt.Fprintf(&out, "<li>%s</li>\n", renderInlineMarkdown(after))
			continue
		}

		closeList()
		fmt.Fprintf(&out, "<p>%s</p>\n", renderInlineMarkdown(trimmed))
	}
	closeList()

	return out.String()
}

func isRawHTMLLine(trimmed string) bool {
	prefixes := []string{
		"<p", "</p>", "<picture", "</picture>", "<source", "<div", "</div>",
		"<a ", "</a>", "<img ", "<details", "</details>",
		"<summary", "</summary>", "<br", "<hr", "<!--", "<sub", "</sub>",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func parseHeading(trimmed string) (int, string, bool) {
	for level := 4; level >= 1; level-- {
		prefix := strings.Repeat("#", level) + " "
		if after, ok := strings.CutPrefix(trimmed, prefix); ok {
			return level, strings.TrimSpace(after), true
		}
	}
	return 0, "", false
}

func renderInlineMarkdown(text string) string {
	escaped := html.EscapeString(text)
	escaped = mdCodeRe.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = mdLinkRe.ReplaceAllString(escaped, `<a href="$2">$1</a>`)
	escaped = mdBoldRe.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = mdItalicRe.ReplaceAllString(escaped, "<em>$1</em>")
	return escaped
}
