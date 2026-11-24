package discord

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientFetchInviteSuccess(t *testing.T) {
	t.Parallel()

	invite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wimpysworld" || r.URL.Query().Get("with_counts") != "true" {
			http.NotFound(w, r)
			return
		}
		if gotUA := r.Header.Get("User-Agent"); gotUA != "vitrine" {
			t.Errorf("User-Agent = %q, want %q", gotUA, "vitrine")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"guild":{"name":"Wimpy's World"},"approximate_presence_count":173,"approximate_member_count":2450}`))
	}))
	defer invite.Close()

	widgetCalled := false
	widget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		widgetCalled = true
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer widget.Close()

	shieldsCalled := false
	shields := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		shieldsCalled = true
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer shields.Close()

	client := &Client{
		HTTPClient:     invite.Client(),
		InviteBaseURL:  invite.URL,
		WidgetBaseURL:  widget.URL,
		ShieldsBaseURL: shields.URL,
	}

	status, err := client.Fetch(context.Background(), "wimpysworld", "1234567890", "https://wimpysworld.com/discord")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if widgetCalled || shieldsCalled {
		t.Fatalf("fallback endpoints should not be called when invite succeeds (widget=%v shields=%v)", widgetCalled, shieldsCalled)
	}

	want := Status{
		Name:    "Wimpy's World",
		Online:  173,
		Members: 2450,
		URL:     "https://wimpysworld.com/discord",
		Source:  sourceInvite,
	}
	if status != want {
		t.Fatalf("got status %+v, want %+v", status, want)
	}
}

func TestClientFetchWidgetFallbackOnInviteFailure(t *testing.T) {
	t.Parallel()

	invite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer invite.Close()

	widget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1234567890/widget.json" {
			http.NotFound(w, r)
			return
		}
		if gotUA := r.Header.Get("User-Agent"); gotUA != "vitrine" {
			t.Errorf("User-Agent = %q, want %q", gotUA, "vitrine")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1234567890","name":"Wimpy's World","presence_count":98}`))
	}))
	defer widget.Close()

	shieldsCalled := false
	shields := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		shieldsCalled = true
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer shields.Close()

	client := &Client{
		HTTPClient:     invite.Client(),
		InviteBaseURL:  invite.URL,
		WidgetBaseURL:  widget.URL,
		ShieldsBaseURL: shields.URL,
	}

	status, err := client.Fetch(context.Background(), "wimpysworld", "1234567890", "https://wimpysworld.com/discord")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if shieldsCalled {
		t.Fatal("shields endpoint should not be called when widget succeeds")
	}

	want := Status{
		Name:   "Wimpy's World",
		Online: 98,
		URL:    "https://wimpysworld.com/discord",
		Source: sourceWidget,
	}
	if status != want {
		t.Fatalf("got status %+v, want %+v", status, want)
	}
}

func TestClientFetchShieldsFallbackOnWidgetFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		inviteCode  string
		shieldsBody string
		wantOnline  int
	}{
		{
			name:        "shields value field",
			inviteCode:  "wimpysworld",
			shieldsBody: `{"name":"discord","value":"173 online"}`,
			wantOnline:  173,
		},
		{
			name:        "shields message field with comma",
			inviteCode:  "",
			shieldsBody: `{"schemaVersion":1,"label":"discord","message":"1,234 online"}`,
			wantOnline:  1234,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			}))
			defer invite.Close()

			widget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "widget disabled", http.StatusForbidden)
			}))
			defer widget.Close()

			shields := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/1234567890.json" {
					http.NotFound(w, r)
					return
				}
				if gotUA := r.Header.Get("User-Agent"); gotUA != "vitrine" {
					t.Errorf("User-Agent = %q, want %q", gotUA, "vitrine")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.shieldsBody))
			}))
			defer shields.Close()

			client := &Client{
				HTTPClient:     shields.Client(),
				InviteBaseURL:  invite.URL,
				WidgetBaseURL:  widget.URL,
				ShieldsBaseURL: shields.URL,
			}

			status, err := client.Fetch(context.Background(), tc.inviteCode, "1234567890", "https://wimpysworld.com/discord")
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			want := Status{
				Online: tc.wantOnline,
				URL:    "https://wimpysworld.com/discord",
				Source: sourceShields,
			}
			if status != want {
				t.Fatalf("got status %+v, want %+v", status, want)
			}
		})
	}
}

func TestClientFetchAllFailAndValidation(t *testing.T) {
	t.Parallel()

	invite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer invite.Close()

	widget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Missing Count"}`))
	}))
	defer widget.Close()

	shields := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schemaVersion":1,"label":"discord","message":"widget disabled"}`))
	}))
	defer shields.Close()

	client := &Client{
		HTTPClient:     invite.Client(),
		InviteBaseURL:  invite.URL,
		WidgetBaseURL:  widget.URL,
		ShieldsBaseURL: shields.URL,
	}

	_, err := client.Fetch(context.Background(), "wimpysworld", "1234567890", "https://wimpysworld.com/discord")
	if err == nil {
		t.Fatal("expected error when all endpoints fail")
	}
	for _, want := range []string{"invite:", "widget:", "shields:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing expected tier context %q", err.Error(), want)
		}
	}

	if _, err := client.Fetch(context.Background(), " ", " ", "https://wimpysworld.com/discord"); err == nil {
		t.Fatal("expected error when both invite code and guild ID are empty")
	}

	newClient := NewClient(nil)
	if newClient.InviteBaseURL != DefaultInviteBaseURL ||
		newClient.WidgetBaseURL != DefaultWidgetBaseURL ||
		newClient.ShieldsBaseURL != DefaultShieldsBaseURL {
		t.Fatalf("unexpected defaults from NewClient: %+v", newClient)
	}
}
