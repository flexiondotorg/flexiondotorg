// Package haberdasher fetches the curated Haberdasher project configuration
// and package repository manifests, merges GitHub metadata, and resolves
// inline project logos.
package haberdasher

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigURL points to the canonical Haberdasher source configuration.
	DefaultConfigURL = "https://raw.githubusercontent.com/wimpysworld/haberdasher/main/haberdasher.yaml"
	// DefaultUserAgent is sent when Fetcher.UserAgent is empty.
	DefaultUserAgent = "vitrine/1.0"
	// SortStars sorts projects descending by GitHub star count.
	SortStars = "stars"
	// SortRelease sorts projects descending by latest release timestamp.
	SortRelease = "release"

	defaultHTTPTimeout   = 30 * time.Second
	maxConfigBytes       = 2 << 20
	maxManifestBytes     = 16 << 20
	maxLogoBytes         = 2 << 20
	defaultDefaultBranch = "main"
)

type repoLogoCandidate struct {
	path  string
	isPNG bool
}

var defaultRepoLogoCandidates = []repoLogoCandidate{
	{path: "pages/icon.svg", isPNG: false},
	{path: "pages/icon.png", isPNG: true},
	{path: "icon.svg", isPNG: false},
	{path: "icon.png", isPNG: true},
}

// CanonicalHosts lists the eight package repository hosts in display order.
var CanonicalHosts = []string{
	"apt.wimpys.world",
	"rpm.wimpys.world",
	"pkg.wimpys.world",
	"apk.wimpys.world",
	"flatpak.wimpys.world",
	"brew.wimpys.world",
	"scoop.wimpys.world",
	"nix.wimpys.world",
}

var (
	defaultHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	pngMagicHeader    = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
)

// PackageHost describes one Haberdasher package repository host.
type PackageHost struct {
	Name         string
	Host         string
	URL          string
	LogoSVG      string
	PackageCount int
	Repos        []string
}

// Project holds merged Haberdasher and GitHub metadata for one curated project.
type Project struct {
	FullName    string
	Owner       string
	Name        string
	Slug        string
	Description string
	Homepage    string
	RepoURL     string
	Language    string
	Stars       int
	LatestTag   string
	ReleaseURL  string
	PublishedAt time.Time
	Hosts       []string
	LogoSVG     string
	LogoDataURI string
}

// Catalogue holds the resolved curated projects and package hosts.
type Catalogue struct {
	Projects []Project
	Hosts    []PackageHost
}

// Options configures a Haberdasher catalogue fetch.
type Options struct {
	ConfigURL           string
	Hosts               []string
	Sort                string
	LogoOverrides       map[string]string
	PNGFallbacks        map[string]string
	HostBaseURLOverride func(host string) string
}

// RepoFetcher retrieves GitHub repository metadata and raw repository files.
type RepoFetcher interface {
	FetchRepoMeta(ctx context.Context, fullName string) (gh.RepoMeta, error)
	FetchRawFile(ctx context.Context, owner, repo, branch, filePath string) ([]byte, error)
}

// Fetcher reads Haberdasher manifests and enriches projects through GitHub.
type Fetcher struct {
	HTTPClient *http.Client
	GitHub     RepoFetcher
	UserAgent  string
}

// NewFetcher constructs a Fetcher with the given HTTP and GitHub clients.
func NewFetcher(httpClient *http.Client, ghClient RepoFetcher) *Fetcher {
	return &Fetcher{
		HTTPClient: httpClient,
		GitHub:     ghClient,
		UserAgent:  DefaultUserAgent,
	}
}

type configDocument struct {
	Sources []string `yaml:"sources"`
}

type hostManifest struct {
	Assets    []manifestAsset   `json:"assets"`
	Homepages map[string]string `json:"homepages"`
	State     manifestState     `json:"state"`
}

type manifestAsset struct {
	Identity manifestIdentity `json:"identity"`
	Source   manifestSource   `json:"source"`
}

type manifestIdentity struct {
	Name string `json:"name"`
}

type manifestSource struct {
	Repository string `json:"repository"`
}

type manifestState struct {
	ActiveIdentities []manifestIdentity `json:"active_identities"`
}

func (f *Fetcher) httpClient() *http.Client {
	if f != nil && f.HTTPClient != nil {
		return f.HTTPClient
	}
	return defaultHTTPClient
}

func (f *Fetcher) userAgent() string {
	if f != nil && strings.TrimSpace(f.UserAgent) != "" {
		return strings.TrimSpace(f.UserAgent)
	}
	return DefaultUserAgent
}

func (f *Fetcher) getBytes(ctx context.Context, targetURL string, maxBytes int64) ([]byte, string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("create request for %s: %w", targetURL, err)
	}
	req.Header.Set("User-Agent", f.userAgent())

	resp, err := f.httpClient().Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("request %s: %w", targetURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, fmt.Errorf("%s returned HTTP %d", targetURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, fmt.Errorf("read %s: %w", targetURL, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, fmt.Errorf("%s exceeds %d bytes", targetURL, maxBytes)
	}
	return data, resp.Header.Get("Content-Type"), resp.StatusCode, nil
}

// Fetch downloads haberdasher.yaml and every configured package host manifest,
// enriches each curated repository via GitHub, resolves project logos, and
// sorts projects according to opts.Sort (defaulting to stars descending).
func (f *Fetcher) Fetch(ctx context.Context, opts Options) (*Catalogue, error) {
	if f == nil || f.GitHub == nil {
		return nil, errors.New("haberdasher: GitHub RepoFetcher must not be nil")
	}

	sortMode, err := normalizeSortMode(opts.Sort)
	if err != nil {
		return nil, err
	}

	configURL := strings.TrimSpace(opts.ConfigURL)
	if configURL == "" {
		configURL = DefaultConfigURL
	}
	sources, err := f.fetchSources(ctx, configURL)
	if err != nil {
		return nil, err
	}

	hosts := opts.Hosts
	if len(hosts) == 0 {
		hosts = CanonicalHosts
	}

	packageHosts := make([]PackageHost, 0, len(hosts))
	repoHosts := make(map[string][]string)
	repoHomepages := make(map[string]string)

	for _, rawHost := range hosts {
		host := strings.TrimSpace(rawHost)
		if host == "" {
			continue
		}
		base := "https://" + host
		if opts.HostBaseURLOverride != nil {
			base = strings.TrimRight(strings.TrimSpace(opts.HostBaseURLOverride(host)), "/")
		}

		pkgHost, homepages, err := f.fetchHost(ctx, host, base)
		if err != nil {
			return nil, err
		}
		packageHosts = append(packageHosts, pkgHost)

		for _, repo := range pkgHost.Repos {
			key := strings.ToLower(repo)
			repoHosts[key] = append(repoHosts[key], host)
		}
		for repo, hp := range homepages {
			key := strings.ToLower(strings.TrimSpace(repo))
			if key != "" && strings.TrimSpace(hp) != "" && repoHomepages[key] == "" {
				repoHomepages[key] = strings.TrimSpace(hp)
			}
		}
	}

	projects := make([]Project, 0, len(sources))
	for _, fullName := range sources {
		meta, err := f.GitHub.FetchRepoMeta(ctx, fullName)
		if err != nil {
			return nil, fmt.Errorf("fetch GitHub metadata for %s: %w", fullName, err)
		}

		resolvedFull := meta.FullName
		if resolvedFull == "" {
			resolvedFull = fullName
		}
		key := strings.ToLower(resolvedFull)

		homepage := meta.Homepage
		if homepage == "" {
			homepage = repoHomepages[key]
		}

		proj := Project{
			FullName:    resolvedFull,
			Owner:       meta.Owner,
			Name:        meta.Name,
			Slug:        slugify(meta.Name),
			Description: meta.Description,
			Homepage:    homepage,
			RepoURL:     meta.HTMLURL,
			Language:    meta.Language,
			Stars:       meta.Stars,
			Hosts:       slices.Clone(repoHosts[key]),
		}
		if meta.LatestRelease != nil {
			proj.LatestTag = meta.LatestRelease.TagName
			proj.ReleaseURL = meta.LatestRelease.ReleaseURL
			proj.PublishedAt = meta.LatestRelease.PublishedAt
		}

		branch := meta.DefaultBranch
		if branch == "" {
			branch = defaultDefaultBranch
		}
		logoSVG, logoDataURI, err := f.resolveLogo(ctx, proj.FullName, proj.Owner, proj.Name, branch, opts)
		if err != nil {
			return nil, fmt.Errorf("resolve logo for %s: %w", proj.FullName, err)
		}
		proj.LogoSVG = logoSVG
		proj.LogoDataURI = logoDataURI

		projects = append(projects, proj)
	}

	sortProjects(projects, sortMode)

	return &Catalogue{
		Projects: projects,
		Hosts:    packageHosts,
	}, nil
}

func normalizeSortMode(sortMode string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(sortMode))
	switch mode {
	case "", SortStars:
		return SortStars, nil
	case SortRelease:
		return SortRelease, nil
	default:
		return "", fmt.Errorf("haberdasher: unknown sort mode %q", sortMode)
	}
}

func sortProjects(projects []Project, sortMode string) {
	mode := strings.ToLower(strings.TrimSpace(sortMode))
	if mode == SortRelease {
		slices.SortFunc(projects, func(a, b Project) int {
			if c := comparePublishedDesc(a, b); c != 0 {
				return c
			}
			if a.Stars != b.Stars {
				return cmp.Compare(b.Stars, a.Stars)
			}
			return compareFullNameAsc(a, b)
		})
		return
	}

	slices.SortFunc(projects, func(a, b Project) int {
		if a.Stars != b.Stars {
			return cmp.Compare(b.Stars, a.Stars)
		}
		if c := comparePublishedDesc(a, b); c != 0 {
			return c
		}
		return compareFullNameAsc(a, b)
	})
}

func comparePublishedDesc(a, b Project) int {
	if a.PublishedAt.Equal(b.PublishedAt) {
		return 0
	}
	if a.PublishedAt.After(b.PublishedAt) {
		return -1
	}
	return 1
}

func compareFullNameAsc(a, b Project) int {
	if c := cmp.Compare(strings.ToLower(a.FullName), strings.ToLower(b.FullName)); c != 0 {
		return c
	}
	return cmp.Compare(a.FullName, b.FullName)
}

func (f *Fetcher) fetchSources(ctx context.Context, configURL string) ([]string, error) {
	data, _, _, err := f.getBytes(ctx, configURL, maxConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("fetch haberdasher config: %w", err)
	}

	var doc configDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse haberdasher config %s: %w", configURL, err)
	}

	seen := make(map[string]bool, len(doc.Sources))
	sources := make([]string, 0, len(doc.Sources))
	for _, src := range doc.Sources {
		trimmed := strings.TrimSpace(src)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		sources = append(sources, trimmed)
	}
	return sources, nil
}

func (f *Fetcher) fetchHost(ctx context.Context, host, baseURL string) (PackageHost, map[string]string, error) {
	manifestURL := baseURL + "/haberdasher.json"
	data, _, _, err := f.getBytes(ctx, manifestURL, maxManifestBytes)
	if err != nil {
		return PackageHost{}, nil, fmt.Errorf("fetch manifest for %s: %w", host, err)
	}

	var manifest hostManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return PackageHost{}, nil, fmt.Errorf("parse manifest for %s: %w", host, err)
	}

	repos, pkgCount := extractHostPackages(manifest)

	logoSVG := ""
	logoURL := baseURL + "/logo.svg"
	logoData, _, statusCode, logoErr := f.getBytes(ctx, logoURL, maxLogoBytes)
	if logoErr == nil {
		logoSVG = strings.TrimSpace(string(logoData))
	} else if statusCode != http.StatusNotFound {
		return PackageHost{}, nil, fmt.Errorf("fetch logo for %s: %w", host, logoErr)
	}

	return PackageHost{
		Name:         subdomainPrefix(host),
		Host:         host,
		URL:          "https://" + host,
		LogoSVG:      logoSVG,
		PackageCount: pkgCount,
		Repos:        repos,
	}, manifest.Homepages, nil
}

func extractHostPackages(manifest hostManifest) ([]string, int) {
	activeNames := make(map[string]bool, len(manifest.State.ActiveIdentities))
	for _, id := range manifest.State.ActiveIdentities {
		name := strings.TrimSpace(id.Name)
		if name != "" {
			activeNames[strings.ToLower(name)] = true
		}
	}

	reposByKey := make(map[string]string)
	for _, asset := range manifest.Assets {
		repo := strings.TrimSpace(asset.Source.Repository)
		if repo == "" {
			continue
		}
		idName := strings.TrimSpace(asset.Identity.Name)
		if len(activeNames) > 0 && idName != "" && !activeNames[strings.ToLower(idName)] {
			continue
		}
		key := strings.ToLower(repo)
		if _, exists := reposByKey[key]; !exists {
			reposByKey[key] = repo
		}
	}

	// Homepages captures hosts such as nix.wimpys.world whose assets list is
	// empty while every published repository is listed under homepages.
	for repo := range manifest.Homepages {
		trimmed := strings.TrimSpace(repo)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := reposByKey[key]; !exists {
			reposByKey[key] = trimmed
		}
	}

	repos := make([]string, 0, len(reposByKey))
	for _, repo := range reposByKey {
		repos = append(repos, repo)
	}
	slices.Sort(repos)

	pkgCount := len(repos)
	if pkgCount == 0 {
		pkgCount = len(activeNames)
	}
	return repos, pkgCount
}

func (f *Fetcher) resolveLogo(
	ctx context.Context,
	fullName, owner, repo, branch string,
	opts Options,
) (logoSVG, logoDataURI string, err error) {
	// Steps 1-4: automatic repository icons (pages/icon.svg -> pages/icon.png -> icon.svg -> icon.png).
	if owner != "" && repo != "" {
		for _, candidate := range defaultRepoLogoCandidates {
			raw, fetchErr := f.GitHub.FetchRawFile(ctx, owner, repo, branch, candidate.path)
			if fetchErr != nil {
				if errors.Is(fetchErr, gh.ErrNotFound) {
					continue
				}
				return "", "", fetchErr
			}
			if candidate.isPNG {
				if len(raw) > 0 {
					return "", encodePNGDataURI(raw), nil
				}
				continue
			}
			if trimmed := strings.TrimSpace(string(raw)); trimmed != "" {
				return trimmed, "", nil
			}
		}
	}

	// Step 5: configured PNG fallback fetched from the repository.
	if pngPath, ok := lookupCaseInsensitive(opts.PNGFallbacks, fullName); ok && pngPath != "" && owner != "" && repo != "" {
		rawPNG, err := f.GitHub.FetchRawFile(ctx, owner, repo, branch, pngPath)
		if err == nil && len(rawPNG) > 0 {
			return "", encodePNGDataURI(rawPNG), nil
		}
		if err != nil && !errors.Is(err, gh.ErrNotFound) {
			return "", "", err
		}
	}

	// Step 6: configured logo override (local file path or HTTP/HTTPS URL).
	if override, ok := lookupCaseInsensitive(opts.LogoOverrides, fullName); ok && override != "" {
		return f.loadLogoOverride(ctx, override)
	}

	// Step 7: empty strings signal internal/svg to draw the fallback placeholder mark.
	return "", "", nil
}

func (f *Fetcher) loadLogoOverride(ctx context.Context, location string) (logoSVG, logoDataURI string, err error) {
	trimmed := strings.TrimSpace(location)
	var data []byte
	var contentType string

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		data, contentType, _, err = f.getBytes(ctx, trimmed, maxLogoBytes)
		if err != nil {
			return "", "", fmt.Errorf("fetch logo override %s: %w", trimmed, err)
		}
	} else {
		data, err = readBoundedFile(trimmed, maxLogoBytes)
		if err != nil {
			return "", "", fmt.Errorf("read logo override %s: %w", trimmed, err)
		}
	}

	if isPNG(data, trimmed, contentType) {
		return "", encodePNGDataURI(data), nil
	}
	return strings.TrimSpace(string(data)), "", nil
}

func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	cleaned := filepath.Clean(path)
	dir := filepath.Dir(cleaned)
	base := filepath.Base(cleaned)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	file, err := root.Open(base)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file %s exceeds %d bytes", cleaned, maxBytes)
	}
	return data, nil
}

func isPNG(data []byte, location, contentType string) bool {
	if bytes.HasPrefix(data, pngMagicHeader) {
		return true
	}
	if strings.Contains(strings.ToLower(contentType), "image/png") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(location), ".png")
}

func encodePNGDataURI(data []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

func lookupCaseInsensitive(m map[string]string, key string) (string, bool) {
	if len(m) == 0 {
		return "", false
	}
	if val, ok := m[key]; ok {
		return strings.TrimSpace(val), true
	}
	lower := strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == lower {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func subdomainPrefix(host string) string {
	name, _, _ := strings.Cut(host, ".")
	return name
}

func slugify(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
