package extlog

import (
	"strings"
	"testing"
)

func TestScrubberPathReplacements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		homeDir     string
		projectRoot string
		input       string
		wantContain string
		wantMissing string
	}{
		{
			name:        "replaces home directory with tilde",
			homeDir:     "/home/testuser",
			projectRoot: "/home/testuser/projects/myapp",
			input:       "error in /home/testuser/.config/something",
			wantContain: "~/.config/something",
			wantMissing: "/home/testuser",
		},
		{
			name:        "replaces project root with dot",
			homeDir:     "",
			projectRoot: "/opt/myapp",
			input:       "compiling /opt/myapp/src/main.go",
			wantContain: "./src/main.go",
			wantMissing: "/opt/myapp",
		},
		{
			name:        "replaces both home and project paths",
			homeDir:     "/home/testuser",
			projectRoot: "/opt/myapp",
			input:       "cache at /home/testuser/.cache, source at /opt/myapp/lib",
			wantContain: "~/.cache",
			wantMissing: "/opt/myapp/lib",
		},
		{
			name:        "empty home dir does not break",
			homeDir:     "",
			projectRoot: "/opt/project",
			input:       "file at /opt/project/README.md",
			wantContain: "./README.md",
		},
		{
			name:        "empty project root does not break",
			homeDir:     "/home/user",
			projectRoot: "",
			input:       "config in /home/user/.bashrc",
			wantContain: "~/.bashrc",
		},
		{
			name:        "no paths to replace",
			homeDir:     "/home/testuser",
			projectRoot: "/home/testuser/projects/myapp",
			input:       "simple message with no paths",
			wantContain: "simple message with no paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewScrubber(tt.homeDir, tt.projectRoot)
			got := s.Scrub(tt.input)
			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("Scrub(%q) = %q, want to contain %q", tt.input, got, tt.wantContain)
			}
			if tt.wantMissing != "" && strings.Contains(got, tt.wantMissing) {
				t.Errorf("Scrub(%q) = %q, should not contain %q", tt.input, got, tt.wantMissing)
			}
		})
	}
}

func TestScrubberTokenPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "npm auth token",
			input: "_authToken=npm_AbCdEfGhIjKlMnOpQrStUvWxYz0123456789",
		},
		{
			name:  "scoped npm auth token",
			input: "//registry.npmjs.org/:_authToken=secret-token-value",
		},
		{
			name:  "nix access-tokens",
			input: "access-tokens = github.com=ghp_abcdef123456",
		},
		{
			name:  "pip index URL with credentials",
			input: "--index-url https://user:pass@pypi.internal.com/simple",
		},
		{
			name:  "pip extra index URL",
			input: "--extra-index-url https://token@internal.pypi.org/simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewScrubber("/home/user", "/project")
			got := s.Scrub(tt.input)
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("Scrub(%q) = %q, expected [REDACTED] in output", tt.input, got)
			}
		})
	}
}

func TestScrubberRedactsKnownSecretPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "AWS access key",
			input: "using key AKIAIOSFODNN7EXAMPLE",
		},
		{
			name:  "GitHub PAT ghp",
			input: "token=ghp_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		},
		{
			name:  "GitHub PAT github_pat",
			input: "auth github_pat_AAAAAAAAAAAAAAAAAAAAAA",
		},
		{
			name:  "GitLab PAT",
			input: "GL_TOKEN=glpat-AAAAAAAAAAAAAAAAAAAAAA",
		},
		{
			name:  "Stripe key",
			input: "stripe_key=sk_live_ABCDEFGHIJKLMNOPQRSTUVWXyz",
		},
		{
			name:  "URL with credentials",
			input: "connecting to https://admin:supersecret@db.example.com/mydb",
		},
		{
			name:  "JWT token",
			input: "bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewScrubber("/home/user", "/project")
			got := s.Scrub(tt.input)
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("Scrub(%q) = %q, expected [REDACTED]", tt.input, got)
			}
		})
	}
}

func TestScrubberPreservesCleanContent(t *testing.T) {
	t.Parallel()

	cleanLines := []string{
		"npm warn deprecated package@1.0.0",
		"resolved https://registry.npmjs.org/express/-/express-4.18.2.tgz",
		"building package list from https://github.com/NixOS/nixpkgs",
		"info: downloading toolchain stable-x86_64-unknown-linux-gnu",
		"go: downloading github.com/stretchr/testify v1.8.4",
		"",
		"   ",
		"2024-01-15T10:30:00 INFO starting server on :8080",
	}

	s := NewScrubber("/home/testuser", "/home/testuser/project")
	for _, line := range cleanLines {
		t.Run("clean_line", func(t *testing.T) {
			got := s.Scrub(line)
			if strings.Contains(got, "[REDACTED]") {
				t.Errorf("Scrub(%q) = %q, should not contain [REDACTED] for clean content", line, got)
			}
		})
	}
}

func TestScrubberEmptyAndWhitespace(t *testing.T) {
	t.Parallel()

	s := NewScrubber("/home/user", "/project")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"single space", " ", " "},
		{"tabs", "\t\t", "\t\t"},
		{"newline", "\n", "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.Scrub(tt.input)
			if got != tt.want {
				t.Errorf("Scrub(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestScrubberPublicHosts(t *testing.T) {
	t.Parallel()

	s := NewScrubber("/home/user", "/project")

	// Public host URLs should not be redacted.
	publicURLs := []string{
		"https://registry.npmjs.org/express",
		"https://pypi.org/project/flask",
		"https://crates.io/crates/serde",
		"https://github.com/user/repo",
	}

	for _, u := range publicURLs {
		t.Run("public_url", func(t *testing.T) {
			got := s.Scrub(u)
			if strings.Contains(got, "[REDACTED]") {
				t.Errorf("Scrub(%q) = %q, public URL should not be redacted", u, got)
			}
		})
	}
}

func TestNewScrubberInitialization(t *testing.T) {
	t.Parallel()

	s := NewScrubber("/home/user", "/opt/project")

	if s == nil {
		t.Fatal("NewScrubber returned nil")
		return
	}
	if s.homeDir != "/home/user" {
		t.Errorf("homeDir = %q, want %q", s.homeDir, "/home/user")
	}
	if s.projectRoot != "/opt/project" {
		t.Errorf("projectRoot = %q, want %q", s.projectRoot, "/opt/project")
	}
	if s.redactor == nil {
		t.Error("redactor is nil")
	}
	if len(s.extraPats) == 0 {
		t.Error("extraPats is empty, expected compiled patterns")
	}
	if len(s.publicHosts) == 0 {
		t.Error("publicHosts is empty, expected public hosts")
	}
}
