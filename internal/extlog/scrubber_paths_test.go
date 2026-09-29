package extlog

import (
	"strings"
	"testing"
)

func TestScrubberProjectUnderHome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		homeDir     string
		projectRoot string
		input       string
		want        string
	}{
		{
			name:        "project under home is normalized to dot",
			homeDir:     "/home/alice",
			projectRoot: "/home/alice/work/acme-secret-client",
			input:       "error in /home/alice/work/acme-secret-client/src/index.js",
			want:        "error in ./src/index.js",
		},
		{
			name:    "sibling home directory is not mangled",
			homeDir: "/home/alice",
			input:   "reading /home/alice2/file",
			want:    "reading /home/alice2/file",
		},
		{
			name:        "sibling project directory is not mangled",
			homeDir:     "/home/alice",
			projectRoot: "/home/alice/app",
			input:       "see /home/alice/app-old/x",
			want:        "see ~/app-old/x",
		},
		{
			name:    "home at end of sentence",
			homeDir: "/home/alice",
			input:   "cwd was /home/alice.",
			want:    "cwd was ~.",
		},
		{
			name:    "home embedded in a longer path is left alone",
			homeDir: "/home/alice",
			input:   "mounted at /mnt/home/alice/data",
			want:    "mounted at /mnt/home/alice/data",
		},
		{
			name:    "quoted home path",
			homeDir: "/home/alice",
			input:   `path "/home/alice" missing`,
			want:    `path "~" missing`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewScrubber(tt.homeDir, tt.projectRoot).Scrub(tt.input)
			if got != tt.want {
				t.Errorf("Scrub(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestScrubberHostRedaction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantContain string
		wantMissing string
	}{
		{
			name:        "private registry host is redacted",
			input:       "GET https://artifactory.megacorp.internal/api/npm/left-pad 401",
			wantContain: "https://" + redactedHost + "/api/npm/left-pad",
			wantMissing: "megacorp",
		},
		{
			name:        "private host with port is redacted",
			input:       "fetch http://cache.corp.example:8080/nix-cache-info failed",
			wantContain: "http://" + redactedHost + ":8080/nix-cache-info",
			wantMissing: "corp.example",
		},
		{
			name:        "private IP host is redacted",
			input:       "connecting to https://10.1.2.3/simple",
			wantContain: redactedHost,
			wantMissing: "10.1.2.3",
		},
		{
			name:        "public host is kept",
			input:       "resolved https://registry.npmjs.org/express/-/express-4.18.2.tgz",
			wantContain: "https://registry.npmjs.org/express",
		},
		{
			name:        "subdomain of public host is kept",
			input:       "GET https://api.github.com/repos/x/y",
			wantContain: "https://api.github.com/repos",
		},
		{
			name:        "private scp-style git remote is redacted",
			input:       "npm ERR! git dep preparation failed: git@git.megacorp.internal:team/repo.git",
			wantContain: "git@" + redactedHost + ":team/repo.git",
			wantMissing: "megacorp",
		},
		{
			name:        "public scp-style git remote is kept",
			input:       "cloning git@github.com:owner/repo.git",
			wantContain: "git@github.com:owner/repo.git",
		},
		{
			name:        "lookalike of public host is redacted",
			input:       "GET https://github.com.evil.internal/x",
			wantMissing: "evil.internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewScrubber("/home/user", "/project").Scrub(tt.input)
			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("Scrub(%q) = %q, want to contain %q", tt.input, got, tt.wantContain)
			}
			if tt.wantMissing != "" && strings.Contains(got, tt.wantMissing) {
				t.Errorf("Scrub(%q) = %q, must not contain %q", tt.input, got, tt.wantMissing)
			}
		})
	}
}
