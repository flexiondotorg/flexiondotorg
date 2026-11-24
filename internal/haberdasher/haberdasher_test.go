package haberdasher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
)

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
}

func TestFetchCatalogueHostsLogoChainAndSorting(t *testing.T) {
	t.Parallel()

	pngBytes := append(slices.Clone(pngMagicHeader), []byte("fake-png-body")...)
	expectedPNGURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)

	pagesPNGBytes := append(slices.Clone(pngMagicHeader), []byte("pages-icon-png")...)
	expectedPagesPNGURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pagesPNGBytes)

	rootPNGBytes := append(slices.Clone(pngMagicHeader), []byte("root-icon-png")...)
	expectedRootPNGURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(rootPNGBytes)

	overrideDir := t.TempDir()
	localSVGPath := filepath.Join(overrideDir, "override.svg")
	if err := os.WriteFile(localSVGPath, []byte(`<svg id="local-override"></svg>`), 0o600); err != nil {
		t.Fatalf("write local override SVG: %v", err)
	}

	tTailor := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	tPagesPNG := time.Date(2026, 11, 21, 18, 0, 0, 0, time.UTC)
	tRootSVG := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	tRootPNG := time.Date(2026, 11, 21, 6, 0, 0, 0, time.UTC)
	tIAGet := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	tWonKey := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) // Tie with ia-get, broken by Stars desc.

	habServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/haberdasher.yaml":
			_, _ = w.Write([]byte("sources: [wimpysworld/ia-get, wimpysworld/WonKey, wimpysworld/tailor, wimpysworld/pages-png, wimpysworld/root-svg, wimpysworld/root-png, linuxmatters/jive-vocals, wimpysworld/nyala]\n"))
		case "/hosts/apt.wimpys.world/haberdasher.json":
			writeJSON(t, w, map[string]any{
				"assets": []map[string]any{
					{"identity": map[string]any{"name": "tailor"}, "source": map[string]any{"repository": "wimpysworld/tailor"}},
					{"identity": map[string]any{"name": "tailor"}, "source": map[string]any{"repository": "wimpysworld/tailor"}},
					{"identity": map[string]any{"name": "ia-get"}, "source": map[string]any{"repository": "wimpysworld/ia-get"}},
					{"identity": map[string]any{"name": "retired"}, "source": map[string]any{"repository": "wimpysworld/retired"}},
				},
				"homepages": map[string]string{
					"wimpysworld/tailor": "https://tailor.wimpys.world/",
					"wimpysworld/ia-get": "https://ia-get.wimpys.world",
				},
				"state": map[string]any{
					"active_identities": []map[string]any{
						{"name": "tailor"},
						{"name": "ia-get"},
					},
				},
			})
		case "/hosts/apt.wimpys.world/logo.svg":
			_, _ = w.Write([]byte(`<svg id="apt-logo"></svg>`))
		case "/hosts/nix.wimpys.world/haberdasher.json":
			// Matches production nix.wimpys.world: assets is empty while homepages lists every repo.
			writeJSON(t, w, map[string]any{
				"assets": []any{},
				"homepages": map[string]string{
					"wimpysworld/tailor":       "https://tailor.wimpys.world/",
					"wimpysworld/WonKey":       "https://wonkey.wimpys.world/",
					"linuxmatters/jive-vocals": "https://linuxmatters.sh/jive/vocals/",
				},
				"state": map[string]any{
					"active_identities": []map[string]any{
						{"name": "tailor"},
						{"name": "wonkey"},
						{"name": "jive-vocals"},
					},
				},
			})
		case "/hosts/nix.wimpys.world/logo.svg":
			_, _ = w.Write([]byte(`<svg id="nix-logo"></svg>`))
		case "/hosts/flatpak.wimpys.world/haberdasher.json":
			writeJSON(t, w, map[string]any{
				"assets": []map[string]any{
					{"identity": map[string]any{"name": "app/world.wimpys.nyala/x86_64/stable"}, "source": map[string]any{"repository": "wimpysworld/nyala"}},
					{"identity": map[string]any{"name": "app/world.wimpys.nyala/aarch64/stable"}, "source": map[string]any{"repository": "wimpysworld/nyala"}},
				},
				"homepages": map[string]string{
					"wimpysworld/nyala": "https://nyala.wimpys.world/",
				},
				"state": map[string]any{
					"active_identities": []map[string]any{
						{"name": "app/world.wimpys.nyala/x86_64/stable"},
						{"name": "app/world.wimpys.nyala/aarch64/stable"},
					},
				},
			})
		case "/hosts/flatpak.wimpys.world/logo.svg":
			http.NotFound(w, r)
		case "/override/remote.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer habServer.Close()

	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// GitHub REST metadata
		case "/repos/wimpysworld/tailor":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/tailor", "name": "tailor", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Tailor", "homepage": "https://tailor.wimpys.world/", "html_url": "https://github.com/wimpysworld/tailor",
				"default_branch": "main", "language": "Go", "stargazers_count": 40,
			})
		case "/repos/wimpysworld/tailor/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name": "v0.6.6", "html_url": "https://github.com/wimpysworld/tailor/releases/tag/v0.6.6", "published_at": tTailor,
			})
		case "/repos/wimpysworld/pages-png":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/pages-png", "name": "pages-png", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Pages PNG test", "html_url": "https://github.com/wimpysworld/pages-png",
				"default_branch": "main", "language": "Go", "stargazers_count": 12,
			})
		case "/repos/wimpysworld/pages-png/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name": "v1.0.0", "html_url": "https://github.com/wimpysworld/pages-png/releases/tag/v1.0.0", "published_at": tPagesPNG,
			})
		case "/repos/wimpysworld/root-svg":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/root-svg", "name": "root-svg", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Root SVG test", "html_url": "https://github.com/wimpysworld/root-svg",
				"default_branch": "main", "language": "Go", "stargazers_count": 11,
			})
		case "/repos/wimpysworld/root-svg/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name": "v1.0.0", "html_url": "https://github.com/wimpysworld/root-svg/releases/tag/v1.0.0", "published_at": tRootSVG,
			})
		case "/repos/wimpysworld/root-png":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/root-png", "name": "root-png", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Root PNG test", "html_url": "https://github.com/wimpysworld/root-png",
				"default_branch": "main", "language": "Go", "stargazers_count": 10,
			})
		case "/repos/wimpysworld/root-png/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name": "v1.0.0", "html_url": "https://github.com/wimpysworld/root-png/releases/tag/v1.0.0", "published_at": tRootPNG,
			})
		case "/repos/wimpysworld/ia-get":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/ia-get", "name": "ia-get", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Internet Archive downloader", "html_url": "https://github.com/wimpysworld/ia-get",
				"default_branch": "main", "language": "Rust", "stargazers_count": 100,
			})
		case "/repos/wimpysworld/ia-get/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name": "v1.2.0", "html_url": "https://github.com/wimpysworld/ia-get/releases/tag/v1.2.0", "published_at": tIAGet,
			})
		case "/repos/wimpysworld/WonKey":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/WonKey", "name": "WonKey", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Keyboard tool", "html_url": "https://github.com/wimpysworld/WonKey",
				"default_branch": "main", "language": "Go", "stargazers_count": 200,
			})
		case "/repos/wimpysworld/WonKey/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name": "v0.2.4", "html_url": "https://github.com/wimpysworld/WonKey/releases/tag/v0.2.4", "published_at": tWonKey,
			})
		case "/repos/linuxmatters/jive-vocals":
			writeJSON(t, w, map[string]any{
				"full_name": "linuxmatters/jive-vocals", "name": "jive-vocals", "owner": map[string]string{"login": "linuxmatters"},
				"description": "Vocal separator", "html_url": "https://github.com/linuxmatters/jive-vocals",
				"default_branch": "main", "language": "Python", "stargazers_count": 15,
			})
		case "/repos/linuxmatters/jive-vocals/releases/latest":
			http.NotFound(w, r)
		case "/repos/wimpysworld/nyala":
			writeJSON(t, w, map[string]any{
				"full_name": "wimpysworld/nyala", "name": "nyala", "owner": map[string]string{"login": "wimpysworld"},
				"description": "Terminal browser", "html_url": "https://github.com/wimpysworld/nyala",
				"default_branch": "main", "language": "Go", "stargazers_count": 30,
			})
		case "/repos/wimpysworld/nyala/releases/latest":
			http.NotFound(w, r)

		// Raw files (steps 1-5 of logo chain, including lower-priority files to verify precedence).
		case "/wimpysworld/tailor/main/pages/icon.svg":
			_, _ = w.Write([]byte(`<svg id="tailor-icon"></svg>`))
		case "/wimpysworld/tailor/main/pages/icon.png":
			_, _ = w.Write(pagesPNGBytes)
		case "/wimpysworld/pages-png/main/pages/icon.png":
			_, _ = w.Write(pagesPNGBytes)
		case "/wimpysworld/pages-png/main/icon.svg":
			_, _ = w.Write([]byte(`<svg id="lower-priority-root-svg"></svg>`))
		case "/wimpysworld/root-svg/main/icon.svg":
			_, _ = w.Write([]byte(`<svg id="root-svg-icon"></svg>`))
		case "/wimpysworld/root-svg/main/icon.png":
			_, _ = w.Write(rootPNGBytes)
		case "/wimpysworld/root-png/main/icon.png":
			_, _ = w.Write(rootPNGBytes)
		case "/wimpysworld/root-png/main/assets/unused.png":
			_, _ = w.Write(pngBytes)
		case "/wimpysworld/ia-get/main/assets/ia-get.png":
			_, _ = w.Write(pngBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ghServer.Close()

	ghClient := &gh.Client{
		HTTPClient: ghServer.Client(),
		APIBaseURL: ghServer.URL,
		RawBaseURL: ghServer.URL,
	}
	fetcher := NewFetcher(habServer.Client(), ghClient)

	cat, err := fetcher.Fetch(context.Background(), Options{
		ConfigURL: habServer.URL + "/haberdasher.yaml",
		Hosts:     []string{"apt.wimpys.world", "flatpak.wimpys.world", "nix.wimpys.world"},
		PNGFallbacks: map[string]string{
			"wimpysworld/root-png": "assets/unused.png",
			"wimpysworld/ia-get":   "assets/ia-get.png",
		},
		LogoOverrides: map[string]string{
			"wimpysworld/wonkey": localSVGPath,
			"wimpysworld/nyala":  habServer.URL + "/override/remote.png",
		},
		HostBaseURLOverride: func(host string) string {
			return habServer.URL + "/hosts/" + host
		},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	// Verify PackageHosts order, counts, and logos.
	if len(cat.Hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(cat.Hosts))
	}
	if cat.Hosts[0].Name != "apt" || cat.Hosts[0].PackageCount != 2 || cat.Hosts[0].LogoSVG != `<svg id="apt-logo"></svg>` {
		t.Fatalf("unexpected apt host: %+v", cat.Hosts[0])
	}
	if cat.Hosts[1].Name != "flatpak" || cat.Hosts[1].PackageCount != 1 || cat.Hosts[1].LogoSVG != "" {
		t.Fatalf("unexpected flatpak host: %+v", cat.Hosts[1])
	}
	if cat.Hosts[2].Name != "nix" || cat.Hosts[2].PackageCount != 3 || cat.Hosts[2].LogoSVG != `<svg id="nix-logo"></svg>` {
		t.Fatalf("unexpected nix host: %+v", cat.Hosts[2])
	}

	// Verify default Projects sorting: Stars desc, then PublishedAt desc, then FullName asc.
	wantStarsOrder := []string{
		"wimpysworld/WonKey",       // 200 stars
		"wimpysworld/ia-get",       // 100 stars
		"wimpysworld/tailor",       // 40 stars
		"wimpysworld/nyala",        // 30 stars
		"linuxmatters/jive-vocals", // 15 stars
		"wimpysworld/pages-png",    // 12 stars
		"wimpysworld/root-svg",     // 11 stars
		"wimpysworld/root-png",     // 10 stars
	}
	if len(cat.Projects) != len(wantStarsOrder) {
		t.Fatalf("expected %d projects, got %d", len(wantStarsOrder), len(cat.Projects))
	}
	byName := make(map[string]Project, len(cat.Projects))
	for i, want := range wantStarsOrder {
		if cat.Projects[i].FullName != want {
			t.Fatalf("stars project[%d] = %q, want %q", i, cat.Projects[i].FullName, want)
		}
		byName[cat.Projects[i].FullName] = cat.Projects[i]
	}

	// Verify explicit release sorting: PublishedAt desc, then Stars desc, then FullName asc.
	releaseCat, err := fetcher.Fetch(context.Background(), Options{
		ConfigURL: habServer.URL + "/haberdasher.yaml",
		Hosts:     []string{"apt.wimpys.world", "flatpak.wimpys.world", "nix.wimpys.world"},
		Sort:      SortRelease,
		PNGFallbacks: map[string]string{
			"wimpysworld/root-png": "assets/unused.png",
			"wimpysworld/ia-get":   "assets/ia-get.png",
		},
		LogoOverrides: map[string]string{
			"wimpysworld/wonkey": localSVGPath,
			"wimpysworld/nyala":  habServer.URL + "/override/remote.png",
		},
		HostBaseURLOverride: func(host string) string {
			return habServer.URL + "/hosts/" + host
		},
	})
	if err != nil {
		t.Fatalf("Fetch (release sort) returned error: %v", err)
	}
	wantReleaseOrder := []string{
		"wimpysworld/tailor",       // 2026-11-22
		"wimpysworld/pages-png",    // 2026-11-21 18:00
		"wimpysworld/root-svg",     // 2026-11-21 12:00
		"wimpysworld/root-png",     // 2026-11-21 06:00
		"wimpysworld/WonKey",       // 2026-11-20, 200 stars
		"wimpysworld/ia-get",       // 2026-11-20, 100 stars
		"wimpysworld/nyala",        // no release, 30 stars
		"linuxmatters/jive-vocals", // no release, 15 stars
	}
	if len(releaseCat.Projects) != len(wantReleaseOrder) {
		t.Fatalf("expected %d release-sorted projects, got %d", len(wantReleaseOrder), len(releaseCat.Projects))
	}
	for i, want := range wantReleaseOrder {
		if releaseCat.Projects[i].FullName != want {
			t.Fatalf("release project[%d] = %q, want %q", i, releaseCat.Projects[i].FullName, want)
		}
	}

	// Verify 7-step logo resolution order and host mapping.
	tailor := byName["wimpysworld/tailor"]
	if tailor.LogoSVG != `<svg id="tailor-icon"></svg>` || tailor.LogoDataURI != "" {
		t.Fatalf("step 1 (pages/icon.svg) failed for tailor: svg=%q uri=%q", tailor.LogoSVG, tailor.LogoDataURI)
	}
	if !slices.Equal(tailor.Hosts, []string{"apt.wimpys.world", "nix.wimpys.world"}) {
		t.Fatalf("unexpected canonical hosts for tailor: %v", tailor.Hosts)
	}

	pagesPNG := byName["wimpysworld/pages-png"]
	if pagesPNG.LogoSVG != "" || pagesPNG.LogoDataURI != expectedPagesPNGURI {
		t.Fatalf("step 2 (pages/icon.png) failed for pages-png: svg=%q uri=%q", pagesPNG.LogoSVG, pagesPNG.LogoDataURI)
	}

	rootSVG := byName["wimpysworld/root-svg"]
	if rootSVG.LogoSVG != `<svg id="root-svg-icon"></svg>` || rootSVG.LogoDataURI != "" {
		t.Fatalf("step 3 (root icon.svg) failed for root-svg: svg=%q uri=%q", rootSVG.LogoSVG, rootSVG.LogoDataURI)
	}

	rootPNG := byName["wimpysworld/root-png"]
	if rootPNG.LogoSVG != "" || rootPNG.LogoDataURI != expectedRootPNGURI {
		t.Fatalf("step 4 (root icon.png) failed for root-png: svg=%q uri=%q", rootPNG.LogoSVG, rootPNG.LogoDataURI)
	}

	wonkey := byName["wimpysworld/WonKey"]
	if wonkey.Slug != "wonkey" || wonkey.LogoSVG != `<svg id="local-override"></svg>` || wonkey.Homepage != "https://wonkey.wimpys.world/" {
		t.Fatalf("step 6 (local SVG override) or homepage fallback failed for WonKey: %+v", wonkey)
	}

	iaget := byName["wimpysworld/ia-get"]
	if iaget.LogoSVG != "" || iaget.LogoDataURI != expectedPNGURI {
		t.Fatalf("step 5 (PNGFallbacks) failed for ia-get: svg=%q uri=%q", iaget.LogoSVG, iaget.LogoDataURI)
	}

	nyala := byName["wimpysworld/nyala"]
	if nyala.LogoSVG != "" || nyala.LogoDataURI != expectedPNGURI {
		t.Fatalf("step 6 (remote PNG override) failed for nyala: svg=%q uri=%q", nyala.LogoSVG, nyala.LogoDataURI)
	}

	jive := byName["linuxmatters/jive-vocals"]
	if jive.LogoSVG != "" || jive.LogoDataURI != "" {
		t.Fatalf("step 7 (empty fallback) failed for jive-vocals: svg=%q uri=%q", jive.LogoSVG, jive.LogoDataURI)
	}
	if !strings.EqualFold(jive.Homepage, "https://linuxmatters.sh/jive/vocals/") {
		t.Fatalf("expected homepage fallback from nix manifest, got %q", jive.Homepage)
	}
}

func TestSortProjectsModesAndTieBreakers(t *testing.T) {
	t.Parallel()

	tNewer := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	tOlder := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

	input := []Project{
		{FullName: "org/zebra", Stars: 50, PublishedAt: tOlder},
		{FullName: "org/Beta", Stars: 50, PublishedAt: tOlder},
		{FullName: "org/alpha", Stars: 50, PublishedAt: tOlder},
		{FullName: "org/fresh-tie", Stars: 50, PublishedAt: tNewer},
		{FullName: "org/top-stars", Stars: 120, PublishedAt: time.Time{}},
		{FullName: "org/newest-low", Stars: 5, PublishedAt: tNewer},
		{FullName: "org/Zero-B", Stars: 5, PublishedAt: time.Time{}},
		{FullName: "org/zero-a", Stars: 5, PublishedAt: time.Time{}},
	}

	for _, mode := range []string{"", SortStars, " STARS "} {
		got := slices.Clone(input)
		sortProjects(got, mode)
		want := []string{
			"org/top-stars",  // 120 stars
			"org/fresh-tie",  // 50 stars, newer release
			"org/alpha",      // 50 stars, older release, name tie-breaker (case-insensitive)
			"org/Beta",       // 50 stars, older release, name tie-breaker
			"org/zebra",      // 50 stars, older release, name tie-breaker
			"org/newest-low", // 5 stars, newer release
			"org/zero-a",     // 5 stars, no release, name tie-breaker
			"org/Zero-B",     // 5 stars, no release, name tie-breaker
		}
		for i, wantName := range want {
			if got[i].FullName != wantName {
				t.Errorf("sortProjects(%q)[%d] = %q, want %q", mode, i, got[i].FullName, wantName)
			}
		}
	}

	gotRelease := slices.Clone(input)
	sortProjects(gotRelease, SortRelease)
	wantRelease := []string{
		"org/fresh-tie",  // tNewer, 50 stars
		"org/newest-low", // tNewer, 5 stars
		"org/alpha",      // tOlder, 50 stars, case-insensitive name
		"org/Beta",       // tOlder, 50 stars
		"org/zebra",      // tOlder, 50 stars
		"org/top-stars",  // no release, 120 stars
		"org/zero-a",     // no release, 5 stars, case-insensitive name
		"org/Zero-B",     // no release, 5 stars
	}
	for i, wantName := range wantRelease {
		if gotRelease[i].FullName != wantName {
			t.Errorf("sortProjects(release)[%d] = %q, want %q", i, gotRelease[i].FullName, wantName)
		}
	}

	f := NewFetcher(nil, stubRepoFetcher{})
	if _, err := f.Fetch(context.Background(), Options{Sort: "invalid"}); err == nil {
		t.Fatal("expected error for invalid sort mode")
	}
}

type stubRepoFetcher struct {
	rawFunc func(ctx context.Context, owner, repo, branch, filePath string) ([]byte, error)
}

func (s stubRepoFetcher) FetchRepoMeta(context.Context, string) (gh.RepoMeta, error) {
	return gh.RepoMeta{}, nil
}

func (s stubRepoFetcher) FetchRawFile(ctx context.Context, owner, repo, branch, filePath string) ([]byte, error) {
	return s.rawFunc(ctx, owner, repo, branch, filePath)
}

func TestResolveLogoPropagatesNonNotFoundErrors(t *testing.T) {
	t.Parallel()

	boom := errors.New("upstream 500")
	f := NewFetcher(nil, stubRepoFetcher{
		rawFunc: func(_ context.Context, _, _, _, filePath string) ([]byte, error) {
			if filePath == "pages/icon.svg" {
				return nil, gh.ErrNotFound
			}
			if filePath == "pages/icon.png" {
				return nil, boom
			}
			return nil, gh.ErrNotFound
		},
	})

	_, _, err := f.resolveLogo(context.Background(), "wimpysworld/demo", "wimpysworld", "demo", "main", Options{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected %v, got %v", boom, err)
	}
}

func TestExtractHostPackagesActiveFallback(t *testing.T) {
	t.Parallel()

	manifest := hostManifest{
		State: manifestState{
			ActiveIdentities: []manifestIdentity{
				{Name: "alpha"},
				{Name: "beta"},
				{Name: "Alpha"},
			},
		},
	}
	repos, count := extractHostPackages(manifest)
	if len(repos) != 0 || count != 2 {
		t.Fatalf("expected 0 repos and fallback count 2, got repos=%v count=%d", repos, count)
	}
}

func TestCanonicalHostsComplete(t *testing.T) {
	t.Parallel()

	want := []string{
		"apt.wimpys.world",
		"rpm.wimpys.world",
		"pkg.wimpys.world",
		"apk.wimpys.world",
		"flatpak.wimpys.world",
		"brew.wimpys.world",
		"scoop.wimpys.world",
		"nix.wimpys.world",
	}
	if !slices.Equal(CanonicalHosts, want) {
		t.Fatalf("CanonicalHosts = %v, want %v", CanonicalHosts, want)
	}
}

func TestFetchNilGitHubAndSlugify(t *testing.T) {
	t.Parallel()

	f := NewFetcher(nil, nil)
	if _, err := f.Fetch(context.Background(), Options{}); err == nil {
		t.Fatal("expected error when GitHub RepoFetcher is nil")
	}

	cases := []struct {
		input string
		want  string
	}{
		{input: "WonKey", want: "wonkey"},
		{input: "ia-get", want: "ia-get"},
		{input: "Jive_Vocals", want: "jive-vocals"},
		{input: "  My App!! ", want: "my-app"},
	}
	for _, tc := range cases {
		if got := slugify(tc.input); got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
