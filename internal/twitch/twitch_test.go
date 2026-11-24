package twitch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchPrimaryLive(t *testing.T) {
	t.Parallel()

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wimpysworld" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("1 hour, 24 minutes\n"))
	}))
	defer primary.Close()

	fallbackCalled := false
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fallbackCalled = true
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer fallback.Close()

	client := &Client{
		HTTPClient:      primary.Client(),
		PrimaryBaseURL:  primary.URL,
		FallbackBaseURL: fallback.URL,
	}

	status, err := client.Fetch(context.Background(), "wimpysworld")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if fallbackCalled {
		t.Fatal("fallback endpoint should not be called when primary succeeds")
	}
	if !status.Live || status.Uptime != "1 hour, 24 minutes" || status.Source != sourcePrimary {
		t.Fatalf("unexpected status: %+v", status)
	}
	if status.Channel != "wimpysworld" || status.URL != "https://www.twitch.tv/wimpysworld" {
		t.Fatalf("unexpected channel metadata: %+v", status)
	}
}

func TestClientFetchPrimaryOffline(t *testing.T) {
	t.Parallel()

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("wimpysworld is offline"))
	}))
	defer primary.Close()

	client := &Client{
		HTTPClient:      primary.Client(),
		PrimaryBaseURL:  primary.URL,
		FallbackBaseURL: "http://127.0.0.1:0",
	}

	status, err := client.Fetch(context.Background(), "wimpysworld")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if status.Live || status.Uptime != "" || status.Source != sourcePrimary {
		t.Fatalf("unexpected offline status: %+v", status)
	}
}

func TestClientFetchFallbackOnPrimaryFailures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		primaryStatus  int
		primaryBody    string
		fallbackBody   string
		wantLive       bool
		wantSourceName string
	}{
		{
			name:           "primary 503 fallback live via message",
			primaryStatus:  http.StatusServiceUnavailable,
			primaryBody:    "unavailable",
			fallbackBody:   `{"schemaVersion":1,"label":"twitch","message":"live"}`,
			wantLive:       true,
			wantSourceName: sourceFallback,
		},
		{
			name:           "primary error text fallback live via value",
			primaryStatus:  http.StatusOK,
			primaryBody:    "Could not retrieve uptime for channel",
			fallbackBody:   `{"name":"twitch","value":"live"}`,
			wantLive:       true,
			wantSourceName: sourceFallback,
		},
		{
			name:           "primary not found text fallback offline",
			primaryStatus:  http.StatusOK,
			primaryBody:    "No user with the name foo found",
			fallbackBody:   `{"schemaVersion":1,"label":"twitch","message":"offline"}`,
			wantLive:       false,
			wantSourceName: sourceFallback,
		},
		{
			name:           "primary generic error fallback offline",
			primaryStatus:  http.StatusOK,
			primaryBody:    "Error: rate limited",
			fallbackBody:   `{"name":"twitch","value":"offline"}`,
			wantLive:       false,
			wantSourceName: sourceFallback,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.primaryStatus)
				_, _ = w.Write([]byte(tc.primaryBody))
			}))
			defer primary.Close()

			fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/wimpysworld.json" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.fallbackBody))
			}))
			defer fallback.Close()

			client := &Client{
				HTTPClient:      primary.Client(),
				PrimaryBaseURL:  primary.URL,
				FallbackBaseURL: fallback.URL,
			}

			status, err := client.Fetch(context.Background(), "wimpysworld")
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}
			if status.Live != tc.wantLive || status.Source != tc.wantSourceName {
				t.Fatalf("got status %+v, want Live=%v Source=%q", status, tc.wantLive, tc.wantSourceName)
			}
		})
	}
}

func TestClientFetchBothFailAndValidation(t *testing.T) {
	t.Parallel()

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer fallback.Close()

	client := &Client{
		HTTPClient:      primary.Client(),
		PrimaryBaseURL:  primary.URL,
		FallbackBaseURL: fallback.URL,
	}

	if _, err := client.Fetch(context.Background(), "wimpysworld"); err == nil {
		t.Fatal("expected error when both primary and fallback fail")
	}

	if _, err := client.Fetch(context.Background(), "   "); err == nil {
		t.Fatal("expected error for blank channel")
	}

	newClient := NewClient(nil)
	if newClient.PrimaryBaseURL != DefaultPrimaryBaseURL || newClient.FallbackBaseURL != DefaultFallbackBaseURL {
		t.Fatalf("unexpected defaults from NewClient: %+v", newClient)
	}
}
