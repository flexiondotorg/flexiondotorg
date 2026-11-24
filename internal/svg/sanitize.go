package svg

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	reURLRef        = regexp.MustCompile(`(?i)url\(\s*['"]?#([^'")\s]+)['"]?\s*\)`)
	reExternalURL   = regexp.MustCompile(`(?i)(?:https?:|javascript:)[^\s'")<>]*`)
	reCSSImport     = regexp.MustCompile(`(?i)@import\b[^;]*;?`)
	reCSSHoverBlock = regexp.MustCompile(`(?is)[^{}]*:hover[^{}]*\{[^{}]*\}`)
	reNumericPrefix = regexp.MustCompile(`^\s*([0-9]+(?:\.[0-9]+)?)`)
)

// forbiddenSVGElements are stripped along with their subtrees during inline
// SVG sanitisation so embedded logos remain safe under GitHub's <img> CSP.
var forbiddenSVGElements = map[string]bool{
	"script":        true,
	"foreignobject": true,
	"iframe":        true,
	"object":        true,
	"embed":         true,
	"audio":         true,
	"video":         true,
	"metadata":      true,
}

type xmlNode struct {
	name     string
	attrs    []xml.Attr
	children []*xmlNode
	text     string
}

// SanitizeInlineSVG extracts the inner geometry and viewBox of rawSVG, removes
// unsafe elements and external URLs, namespaces internal id and url(#...)
// references with a deterministic per-placement prefix, and returns a nested
// <svg> element positioned at (x, y) with width and height equal to size.
func SanitizeInlineSVG(rawSVG string, x, y, size float64) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(rawSVG, "\ufeff"))
	if trimmed == "" {
		return "", errors.New("svg: empty inline SVG")
	}
	if size <= 0 {
		return "", errors.New("svg: inline SVG size must be positive")
	}

	root, err := parseSVGRoot(trimmed)
	if err != nil {
		return "", err
	}

	viewBox := extractViewBox(root)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%.2f:%.2f:%.2f:%s", x, y, size, trimmed)))
	prefix := fmt.Sprintf("vlogo-%x-", sum[:4])

	var inner bytes.Buffer
	for _, child := range root.children {
		writeSanitizedNode(&inner, child, prefix)
	}

	return fmt.Sprintf(
		`<svg x="%s" y="%s" width="%s" height="%s" viewBox="%s" preserveAspectRatio="xMidYMid meet">%s</svg>`,
		formatFloat(x),
		formatFloat(y),
		formatFloat(size),
		formatFloat(size),
		escapeXML(viewBox),
		inner.String(),
	), nil
}

// RenderDataURIImage validates a base64 image data URI and returns a safe
// inline <image> element positioned at (x, y) with width and height size.
func RenderDataURIImage(dataURI string, x, y, size float64) (string, error) {
	trimmed := strings.TrimSpace(dataURI)
	if size <= 0 {
		return "", errors.New("svg: image size must be positive")
	}
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "data:image/") || !strings.Contains(lower, ";base64,") {
		return "", errors.New("svg: data URI must be a base64 data:image/ URI")
	}
	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "javascript:") {
		return "", errors.New("svg: data URI contains disallowed URL scheme")
	}

	return fmt.Sprintf(
		`<image href="%s" x="%s" y="%s" width="%s" height="%s" preserveAspectRatio="xMidYMid meet"/>`,
		escapeXML(trimmed),
		formatFloat(x),
		formatFloat(y),
		formatFloat(size),
		formatFloat(size),
	), nil
}

func parseSVGRoot(raw string) (*xmlNode, error) {
	dec := xml.NewDecoder(strings.NewReader(raw))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose

	var stack []*xmlNode
	var svgRoot *xmlNode

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("svg: parse inline SVG: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			node := &xmlNode{
				name:  t.Name.Local,
				attrs: t.Attr,
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
			}
			if svgRoot == nil && strings.EqualFold(t.Name.Local, "svg") {
				svgRoot = node
			}
			stack = append(stack, node)
		case xml.EndElement:
			for i := len(stack) - 1; i >= 0; i-- {
				if strings.EqualFold(stack[i].name, t.Name.Local) {
					stack = stack[:i]
					break
				}
			}
		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, &xmlNode{
					text: string(t),
				})
			}
		}
	}

	if svgRoot == nil {
		return nil, errors.New("svg: missing root <svg> element")
	}
	return svgRoot, nil
}

func extractViewBox(root *xmlNode) string {
	var width, height string
	for _, attr := range root.attrs {
		switch strings.ToLower(attr.Name.Local) {
		case "viewbox":
			if vb := strings.TrimSpace(attr.Value); vb != "" {
				return vb
			}
		case "width":
			width = attr.Value
		case "height":
			height = attr.Value
		}
	}

	w, okW := parseSVGDimension(width)
	h, okH := parseSVGDimension(height)
	if okW && okH && w > 0 && h > 0 {
		return fmt.Sprintf("0 0 %s %s", formatFloat(w), formatFloat(h))
	}
	return "0 0 100 100"
}

func parseSVGDimension(raw string) (float64, bool) {
	m := reNumericPrefix.FindStringSubmatch(raw)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func writeSanitizedNode(buf *bytes.Buffer, node *xmlNode, prefix string) {
	if node.name == "" {
		if strings.TrimSpace(node.text) != "" {
			buf.WriteString(escapeXML(stripExternalURLs(node.text)))
		}
		return
	}

	lowerName := strings.ToLower(node.name)
	if forbiddenSVGElements[lowerName] {
		return
	}

	// Unwrap internal <a> elements while preserving their nested vector artwork.
	if lowerName == "a" {
		for _, child := range node.children {
			writeSanitizedNode(buf, child, prefix)
		}
		return
	}

	// Sanitize inline <style> blocks against external URLs, :hover, and ID clashes.
	if lowerName == "style" {
		var cssBuf strings.Builder
		for _, child := range node.children {
			cssBuf.WriteString(child.text)
		}
		cleaned := sanitizeEmbeddedCSS(cssBuf.String(), prefix)
		if strings.TrimSpace(cleaned) != "" {
			buf.WriteString("<style>")
			buf.WriteString(escapeXML(cleaned))
			buf.WriteString("</style>")
		}
		return
	}

	buf.WriteByte('<')
	buf.WriteString(node.name)

	seenAttrs := make(map[string]bool, len(node.attrs))
	for _, attr := range node.attrs {
		name, val, keep := sanitizeSVGAttr(attr, prefix)
		if !keep {
			continue
		}
		key := strings.ToLower(name)
		if seenAttrs[key] {
			continue
		}
		seenAttrs[key] = true
		buf.WriteByte(' ')
		buf.WriteString(name)
		buf.WriteString(`="`)
		buf.WriteString(escapeXML(val))
		buf.WriteByte('"')
	}

	if len(node.children) == 0 {
		buf.WriteString("/>")
		return
	}

	buf.WriteByte('>')
	for _, child := range node.children {
		writeSanitizedNode(buf, child, prefix)
	}
	buf.WriteString("</")
	buf.WriteString(node.name)
	buf.WriteByte('>')
}

func sanitizeSVGAttr(attr xml.Attr, prefix string) (name, value string, keep bool) {
	local := attr.Name.Local
	lowerLocal := strings.ToLower(local)

	// Drop XML namespace declarations inside nested geometry.
	if lowerLocal == "xmlns" || strings.EqualFold(attr.Name.Space, "xmlns") {
		return "", "", false
	}

	// Drop inline DOM event handlers such as onclick or onload.
	if strings.HasPrefix(lowerLocal, "on") {
		return "", "", false
	}

	// Normalize xlink:href to SVG 2 href and enforce safe targets only.
	if lowerLocal == "href" || lowerLocal == "src" {
		trimmed := strings.TrimSpace(attr.Value)
		lowerVal := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(trimmed, "#"):
			return "href", "#" + prefix + strings.TrimPrefix(trimmed, "#"), true
		case strings.HasPrefix(lowerVal, "data:image/") && !reExternalURL.MatchString(trimmed):
			return "href", trimmed, true
		default:
			return "", "", false
		}
	}

	if lowerLocal == "id" {
		id := strings.TrimSpace(attr.Value)
		if id == "" {
			return "", "", false
		}
		return "id", prefix + id, true
	}

	val := rewriteURLRefs(attr.Value, prefix)
	if lowerLocal == "style" {
		val = sanitizeEmbeddedCSS(val, prefix)
	}
	if reExternalURL.MatchString(val) {
		return "", "", false
	}

	return local, val, true
}

func rewriteURLRefs(s, prefix string) string {
	return reURLRef.ReplaceAllString(s, "url(#"+prefix+"$1)")
}

func sanitizeEmbeddedCSS(css, prefix string) string {
	cleaned := reCSSImport.ReplaceAllString(css, "")
	cleaned = reCSSHoverBlock.ReplaceAllString(cleaned, "")
	cleaned = rewriteURLRefs(cleaned, prefix)
	cleaned = stripExternalURLs(cleaned)
	// Ensure no stray :hover selector remains even inside multi-selector rules.
	cleaned = strings.ReplaceAll(cleaned, ":hover", "")
	cleaned = strings.ReplaceAll(cleaned, ":HOVER", "")
	return cleaned
}

func stripExternalURLs(s string) string {
	return reExternalURL.ReplaceAllString(s, "")
}
