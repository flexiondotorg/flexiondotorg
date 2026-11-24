package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed web/* web/vendor/*
var webAssets embed.FS

type indexTemplateData struct {
	Title       string
	Description string
	URL         string
	Domain      string
	Year        int
	RepoURL     string
	LicenseURL  string
	SiteData    *SiteData
	Data        *SiteData
}

// Write emits the static portfolio site (index.html, data.json, style.css,
// app.js, icon.svg, vendor/pixi.min.mjs, vendor/pixi.LICENSE, .nojekyll,
// and optional CNAME) into outDir.
func Write(outDir string, data *SiteData) error {
	if data == nil {
		return errors.New("site: data must not be nil")
	}

	cleaned := filepath.Clean(strings.TrimSpace(outDir))
	if cleaned == "" || cleaned == "." {
		return errors.New("site: output directory must not be empty")
	}

	if err := os.MkdirAll(cleaned, 0o750); err != nil {
		return fmt.Errorf("site: create output directory %q: %w", outDir, err)
	}

	root, err := os.OpenRoot(cleaned)
	if err != nil {
		return fmt.Errorf("site: open output directory %q: %w", outDir, err)
	}
	defer func() { _ = root.Close() }()

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("site: marshal data.json: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')
	if err := writeRootFile(root, "data.json", jsonBytes); err != nil {
		return err
	}

	indexHTML, err := renderIndexHTML(data)
	if err != nil {
		return err
	}
	if err := writeRootFile(root, "index.html", indexHTML); err != nil {
		return err
	}

	for _, asset := range []string{"style.css", "app.js", "icon.svg"} {
		content, readErr := fs.ReadFile(webAssets, "web/"+asset)
		if readErr != nil {
			return fmt.Errorf("site: read embedded asset %q: %w", asset, readErr)
		}
		if err := writeRootFile(root, asset, content); err != nil {
			return err
		}
	}

	if err := root.Mkdir("vendor", 0o750); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("site: create vendor directory: %w", err)
	}
	for _, asset := range []string{"vendor/pixi.min.mjs", "vendor/pixi.LICENSE"} {
		content, readErr := fs.ReadFile(webAssets, "web/"+asset)
		if readErr != nil {
			return fmt.Errorf("site: read embedded asset %q: %w", asset, readErr)
		}
		if err := writeRootFile(root, asset, content); err != nil {
			return err
		}
	}

	if err := writeRootFile(root, ".nojekyll", nil); err != nil {
		return err
	}

	if domain := strings.TrimSpace(data.Profile.Domain); domain != "" {
		if err := writeRootFile(root, "CNAME", []byte(domain+"\n")); err != nil {
			return err
		}
	}

	return nil
}

func renderIndexHTML(data *SiteData) ([]byte, error) {
	raw, err := fs.ReadFile(webAssets, "web/index.html")
	if err != nil {
		return nil, fmt.Errorf("site: read embedded index.html: %w", err)
	}

	tmpl, err := template.New("index.html").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("site: parse index.html template: %w", err)
	}

	title := strings.TrimSpace(data.Profile.Title)
	if title == "" {
		title = "Wimpy's World"
	}
	desc := strings.TrimSpace(data.Profile.Subtitle)
	if desc == "" {
		desc = data.Profile.BioPlainText
	}

	year := time.Now().UTC().Year()
	if parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(data.GeneratedAt)); parseErr == nil && parsed.Year() > 0 {
		year = parsed.Year()
	}

	handle := strings.TrimSpace(data.Profile.Handle)
	repoURL := "https://github.com/flexiondotorg/flexiondotorg"
	if handle != "" {
		repoURL = "https://github.com/" + handle + "/" + handle
	}
	licenseURL := repoURL + "/blob/main/LICENSE"

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, indexTemplateData{
		Title:       title,
		Description: desc,
		URL:         strings.TrimSpace(data.Profile.URL),
		Domain:      strings.TrimSpace(data.Profile.Domain),
		Year:        year,
		RepoURL:     repoURL,
		LicenseURL:  licenseURL,
		SiteData:    data,
		Data:        data,
	})
	if err != nil {
		return nil, fmt.Errorf("site: execute index.html template: %w", err)
	}
	return buf.Bytes(), nil
}

func writeRootFile(root *os.Root, name string, content []byte) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("site: open %q: %w", name, err)
	}
	_, writeErr := f.Write(content)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("site: write %q: %w", name, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("site: close %q: %w", name, closeErr)
	}
	return nil
}
