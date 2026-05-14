package ecosystem

import "testing"

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

func TestProxyKeyForLanguage(t *testing.T) {
	tests := []struct {
		lang, pm, want string
	}{
		{"javascript", "npm", "npm"},
		{"javascript", "pnpm", "npm"},
		{"python", "pip", "pypi"},
		{"go", "", "go"},
		{"java", "maven", "maven"},
		{"java", "gradle", "gradle"},
		{"rust", "", "cargo"},
		{"dotnet", "", "nuget"},
		{"php", "", "composer"},
		{"unknown", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.pm, func(t *testing.T) {
			got := ProxyKeyForLanguage(tt.lang, tt.pm)
			if got != tt.want {
				t.Errorf("ProxyKeyForLanguage(%q, %q) = %q, want %q", tt.lang, tt.pm, got, tt.want)
			}
		})
	}
}
