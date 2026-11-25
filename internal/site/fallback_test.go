package site

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestMergeFallbackRestoresFullGitHubOutage(t *testing.T) {
	t.Parallel()

	fallback := &SiteData{
		DefaultSort: "stars",
		HostLogos: map[string]string{
			"apt.wimpys.world": "<svg>apt</svg>",
		},
		Stats: StatsData{
			TotalStars:     9500,
			StarsFormatted: "9,500",
			Followers:      1200,
			PublicRepos:    90,
			TotalCommits:   4200,
			ProjectCount:   2,
			HostCount:      1,
			ActiveSponsors: 12,
			PastSponsors:   40,
			DiscordOnline:  180,
			DiscordMembers: 1500,
		},
		Projects: []ProjectCard{
			{
				ID:             "deb-get",
				FullName:       "wimpysworld/deb-get",
				Name:           "Deb Get",
				Slug:           "deb-get",
				Description:    "apt-get for third-party debs",
				Language:       "Shell",
				LanguageColor:  "#89e051",
				Stars:          4200,
				StarsFormatted: "4,200",
				StarsCompact:   "4.2k",
				LatestTag:      "v0.4.5",
				ReleaseURL:     "https://github.com/wimpysworld/deb-get/releases/tag/v0.4.5",
				LogoSVG:        "<svg>deb</svg>",
				Hosts:          []ProjectHostRef{{Name: "apt", Host: "apt.wimpys.world", URL: "https://apt.wimpys.world/#deb-get"}},
			},
			{
				ID:             "tailor",
				FullName:       "wimpysworld/tailor",
				Name:           "Tailor",
				Slug:           "tailor",
				Description:    "Ready-to-wear project templates",
				Language:       "Go",
				LanguageColor:  "#00ADD8",
				Stars:          250,
				StarsFormatted: "250",
				StarsCompact:   "250",
				LatestTag:      "v0.7.0",
			},
		},
		Packages: []PackageHostCard{
			{ID: "apt", Name: "apt", Host: "apt.wimpys.world", PackageCount: 6},
		},
		Links: []LinkCard{
			{
				ID:           "github",
				Badge:        "12 active",
				Stats:        []StatRow{{Label: "STARS", Value: "9,500"}, {Label: "FOLLOWERS", Value: "1,200"}},
				Meta:         []string{"9,500 stars"},
				Sponsors:     []SponsorRef{{Login: "alice", URL: "https://github.com/alice"}},
				PastSponsors: []SponsorRef{{Login: "bob", URL: "https://github.com/bob"}},
			},
			{
				ID:    "discord",
				Badge: "180 online",
				Stats: []StatRow{{Label: "ONLINE", Value: "180"}, {Label: "MEMBERS", Value: "1,500"}},
			},
		},
		Media: []MediaCard{
			{ID: "ep-94", Kind: "episode", Title: "Episode 94"},
			{ID: "post-1", Kind: "post", Title: "Old Post"},
			{ID: "vid-1", Kind: "video", Title: "Disabled Video"},
		},
	}

	// Simulate GitHub + Haberdasher + Podcast outage while blog RSS still succeeds
	// and configured project deb-get has zero live stars/description.
	current := &SiteData{
		DefaultSort: "stars",
		Profile:     ProfileData{Title: "Fresh Title"},
		HostLogos:   map[string]string{},
		Projects: []ProjectCard{
			{
				ID:       "deb-get",
				FullName: "wimpysworld/deb-get",
				Name:     "Deb Get",
				Slug:     "deb-get",
			},
		},
		Packages: []PackageHostCard{},
		Links: []LinkCard{
			{
				ID:          "github",
				Title:       "Updated GitHub Title",
				Description: "Updated description",
				Stats:       []StatRow{{Label: "GITHUB", Value: "@flexiondotorg"}},
			},
			{
				ID:    "discord",
				Title: "Discord",
				Stats: []StatRow{{Label: "SITE", Value: "wimpysworld.io"}},
			},
		},
		Media: []MediaCard{
			{ID: "post-new", Kind: "post", Title: "Fresh Blog Post"},
		},
	}

	enabled := map[string]bool{
		"episode": true,
		"post":    true,
		"video":   false,
	}

	report := MergeFallback(current, fallback, enabled)
	if !report.Used() {
		t.Fatal("expected fallback report to indicate fallback was used")
	}
	if report.ProjectsEnriched != 1 || report.ProjectsRestored != 1 {
		t.Errorf("projects report = enriched:%d restored:%d, want 1 and 1", report.ProjectsEnriched, report.ProjectsRestored)
	}
	if len(current.Projects) != 2 || current.Projects[0].Stars != 4200 || current.Projects[1].Slug != "tailor" {
		t.Errorf("unexpected merged projects: %+v", current.Projects)
	}
	if !report.PackagesRestored || len(current.Packages) != 1 || current.HostLogos["apt.wimpys.world"] == "" {
		t.Errorf("expected packages and host logos restored: %+v %+v", current.Packages, current.HostLogos)
	}
	if !report.StatsRestored || current.Stats.TotalStars != 9500 || current.Stats.ActiveSponsors != 12 || current.Stats.DiscordMembers != 1500 {
		t.Errorf("unexpected merged stats: %+v", current.Stats)
	}
	if report.LinksRestored != 2 {
		t.Errorf("LinksRestored = %d, want 2", report.LinksRestored)
	}
	if current.Links[0].Title != "Updated GitHub Title" || len(current.Links[0].Sponsors) != 1 {
		t.Errorf("expected link copy preserved and sponsors restored: %+v", current.Links[0])
	}
	if len(current.Media) != 2 {
		t.Fatalf("len(Media) = %d, want 2 (fresh post + restored episode, no disabled video): %+v", len(current.Media), current.Media)
	}
	if current.Media[0].Title != "Fresh Blog Post" || current.Media[1].Title != "Episode 94" {
		t.Errorf("unexpected merged media: %+v", current.Media)
	}
	if current.Profile.Title != "Fresh Title" {
		t.Errorf("profile title overwritten: %q", current.Profile.Title)
	}
	if report.Summary() == "" {
		t.Error("expected non-empty fallback summary")
	}
}

func TestLoadDataAndSaveDataCache(t *testing.T) {
	t.Parallel()

	sample := &SiteData{
		Profile: ProfileData{Title: "Wimpy's World"},
		Stats:   StatsData{TotalStars: 1234},
	}

	cachePath := filepath.Join(t.TempDir(), "nested", "site-data.json")
	if err := SaveDataCache(cachePath, sample); err != nil {
		t.Fatalf("SaveDataCache error: %v", err)
	}

	fromFile, err := LoadData(context.Background(), nil, cachePath)
	if err != nil {
		t.Fatalf("LoadData(file) error: %v", err)
	}
	if fromFile.Stats.TotalStars != 1234 {
		t.Fatalf("fromFile.Stats.TotalStars = %d, want 1234", fromFile.Stats.TotalStars)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sample)
	}))
	t.Cleanup(srv.Close)

	fromURL, err := LoadData(context.Background(), srv.Client(), srv.URL+"/data.json")
	if err != nil {
		t.Fatalf("LoadData(url) error: %v", err)
	}
	if fromURL.Profile.Title != "Wimpy's World" {
		t.Fatalf("fromURL.Profile.Title = %q, want Wimpy's World", fromURL.Profile.Title)
	}
}
