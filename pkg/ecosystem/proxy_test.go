package ecosystem

import (
	"net/url"
	"testing"
)

func TestResolveProxyURL_BaseOnly(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		ecosystem string
		want      string
	}{
		{"npm", "https://nexus.corp.example.com", "npm", "https://nexus.corp.example.com/repository/npm-proxy/"},
		{"pypi", "https://nexus.corp.example.com", "pypi", "https://nexus.corp.example.com/repository/pypi-proxy/simple/"},
		{"go", "https://nexus.corp.example.com", "go", "https://nexus.corp.example.com/repository/go-proxy/"},
		{"maven", "https://nexus.corp.example.com", "maven", "https://nexus.corp.example.com/repository/maven-central/"},
		{"cargo", "https://nexus.corp.example.com", "cargo", "https://nexus.corp.example.com/repository/cargo-proxy/"},
		{"nuget", "https://nexus.corp.example.com", "nuget", "https://nexus.corp.example.com/repository/nuget-proxy/v3/index.json"},
		{"composer", "https://nexus.corp.example.com", "composer", "https://nexus.corp.example.com/repository/composer-proxy/"},
		{"trailing-slash", "https://nexus.corp.example.com/", "npm", "https://nexus.corp.example.com/repository/npm-proxy/"},
		{"unknown-ecosystem", "https://nexus.corp.example.com", "unknown", ""},
		{"empty-base", "", "npm", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveProxyURL(tt.baseURL, nil, tt.ecosystem)
			if got != tt.want {
				t.Errorf("ResolveProxyURL(%q, nil, %q) = %q, want %q", tt.baseURL, tt.ecosystem, got, tt.want)
			}
		})
	}
}

func TestResolveProxyURL_WithOverride(t *testing.T) {
	overrides := map[string]string{
		"npm": "https://custom-npm.corp.example.com/",
	}

	got := ResolveProxyURL("https://nexus.corp.example.com", overrides, "npm")
	want := "https://custom-npm.corp.example.com/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	got = ResolveProxyURL("https://nexus.corp.example.com", overrides, "pypi")
	want = "https://nexus.corp.example.com/repository/pypi-proxy/simple/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveProxyURL_OverrideOnlyNoBase(t *testing.T) {
	overrides := map[string]string{
		"npm": "https://custom-npm.corp.example.com/",
	}
	got := ResolveProxyURL("", overrides, "npm")
	if got != "https://custom-npm.corp.example.com/" {
		t.Errorf("got %q, want override URL", got)
	}
	got = ResolveProxyURL("", overrides, "pypi")
	if got != "" {
		t.Errorf("got %q, want empty for non-overridden ecosystem with no base", got)
	}
}

func TestJoinProxyURL_HostPreserved(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		base    string
		path    string
		want    string
		wantErr bool
	}{
		{"plain path", "https://a.io", "/repository/npm/", "https://a.io/repository/npm/", false},
		{"base trailing slash", "https://a.io/", "/repository/npm/", "https://a.io/repository/npm/", false},
		{"base with path", "https://a.io/art", "/api/npm/", "https://a.io/art/api/npm/", false},
		{"dot-dot stays on host", "https://a.io", "/../../evil", "https://a.io/evil", false},
		{"query and fragment escaped", "https://a.io", "/a?b#c", "https://a.io/a%3Fb%23c", false},
		{"loopback http with port", "http://127.0.0.1:8081", "/npm/", "http://127.0.0.1:8081/npm/", false},
		{"at-sign host rewrite", "https://a.io", "@evil.io/npm/", "", true},
		{"bare host", "https://a.io", "evil.com/x", "", true},
		{"empty path", "https://a.io", "", "", true},
		{"relative path", "https://a.io", "npm/", "", true},
		{"protocol-relative", "https://a.io", "//evil.com/x", "", true},
		{"newline in base", "https://a.io\nevil", "/npm/", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := JoinProxyURL(tt.base, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("JoinProxyURL(%q, %q) = %q, want error", tt.base, tt.path, got)
				}
				if got != "" {
					t.Errorf("JoinProxyURL(%q, %q) returned %q alongside an error", tt.base, tt.path, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("JoinProxyURL(%q, %q) error: %v", tt.base, tt.path, err)
			}
			if got != tt.want {
				t.Errorf("JoinProxyURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
			}
			b, err := url.Parse(tt.base)
			if err != nil {
				t.Fatalf("parse base: %v", err)
			}
			r, err := url.Parse(got)
			if err != nil {
				t.Fatalf("parse result %q: %v", got, err)
			}
			if r.Scheme != b.Scheme || r.User.String() != b.User.String() || r.Host != b.Host {
				t.Errorf("JoinProxyURL(%q, %q) = %q changed scheme/userinfo/host (got %q %q %q, want %q %q %q)",
					tt.base, tt.path, got, r.Scheme, r.User, r.Host, b.Scheme, b.User, b.Host)
			}
		})
	}
}

func TestResolveProxyURL_InvalidPathOrBaseYieldsEmpty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		base        string
		customPaths map[string]string
	}{
		{"at-sign custom path", "https://artifactory.corp.io", map[string]string{"npm": "@attacker.io/npm/"}},
		{"newline base default path", "https://nexus.corp.io\nattacker", nil},
		{"newline base custom path", "https://nexus.corp.io\nattacker", map[string]string{"npm": "/npm/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveProxyURL(tt.base, nil, "npm", tt.customPaths); got != "" {
				t.Errorf("ResolveProxyURL(%q, nil, npm, %v) = %q, want \"\"", tt.base, tt.customPaths, got)
			}
		})
	}
}
