package bugreport

import (
	"strings"
	"testing"
)

func TestFormatIssueBodyMinimal(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Description: "Something broke.",
	}
	body := r.FormatIssueBody()

	if !strings.Contains(body, "## Bug Report") {
		t.Error("missing '## Bug Report' header")
	}
	if !strings.Contains(body, "### Description") {
		t.Error("missing '### Description' header")
	}
	if !strings.Contains(body, "Something broke.") {
		t.Error("missing description text")
	}
	if !strings.Contains(body, "Filed via") {
		t.Error("missing footer attribution")
	}
}

func TestFormatIssueBodySteps(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Description: "Crash on init.",
		Steps:       "1. Run qsdev init\n2. Select Go\n3. Crash",
	}
	body := r.FormatIssueBody()

	if !strings.Contains(body, "### Steps to Reproduce") {
		t.Error("missing steps header")
	}
	if !strings.Contains(body, "1. Run qsdev init") {
		t.Error("missing steps content")
	}
}

func TestFormatIssueBodyNoSteps(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Description: "Bug description.",
		Steps:       "",
	}
	body := r.FormatIssueBody()

	if strings.Contains(body, "### Steps to Reproduce") {
		t.Error("steps header should be absent when Steps is empty")
	}
}

func TestFormatIssueBodyClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity string
		category string
		wantHdr  bool
		wantSev  bool
		wantCat  bool
	}{
		{
			name:     "both severity and category",
			severity: "major",
			category: "init-wizard",
			wantHdr:  true,
			wantSev:  true,
			wantCat:  true,
		},
		{
			name:     "severity only",
			severity: "critical",
			category: "",
			wantHdr:  true,
			wantSev:  true,
			wantCat:  false,
		},
		{
			name:     "category only",
			severity: "",
			category: "security-hardening",
			wantHdr:  true,
			wantSev:  false,
			wantCat:  true,
		},
		{
			name:     "neither",
			severity: "",
			category: "",
			wantHdr:  false,
			wantSev:  false,
			wantCat:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &BugReport{
				Description: "desc",
				Severity:    tt.severity,
				Category:    tt.category,
			}
			body := r.FormatIssueBody()

			hasHdr := strings.Contains(body, "### Classification")
			if hasHdr != tt.wantHdr {
				t.Errorf("Classification header presence = %v, want %v", hasHdr, tt.wantHdr)
			}
			hasSev := strings.Contains(body, "**Severity:**")
			if hasSev != tt.wantSev {
				t.Errorf("Severity field presence = %v, want %v", hasSev, tt.wantSev)
			}
			hasCat := strings.Contains(body, "**Category:**")
			if hasCat != tt.wantCat {
				t.Errorf("Category field presence = %v, want %v", hasCat, tt.wantCat)
			}
		})
	}
}

func TestFormatIssueBodyEnvironment(t *testing.T) {
	t.Parallel()

	env := Environment{
		QsdevVersion: "0.8.0",
		Commit:       "abc123",
		GoVersion:    "go1.22.0",
		OS:           "linux",
		Arch:         "amd64",
		Family:       "nixos",
		Shell:        "zsh",
		HasNix:       true,
	}

	t.Run("included", func(t *testing.T) {
		t.Parallel()
		r := &BugReport{
			Description: "desc",
			Environment: env,
			IncludeEnv:  true,
		}
		body := r.FormatIssueBody()

		if !strings.Contains(body, "### Environment") {
			t.Error("missing Environment header")
		}
		if !strings.Contains(body, "0.8.0") {
			t.Error("missing version in environment table")
		}
		if !strings.Contains(body, "linux/amd64") {
			t.Error("missing OS/arch in environment table")
		}
	})

	t.Run("excluded", func(t *testing.T) {
		t.Parallel()
		r := &BugReport{
			Description: "desc",
			Environment: env,
			IncludeEnv:  false,
		}
		body := r.FormatIssueBody()

		if strings.Contains(body, "### Environment") {
			t.Error("Environment header should be absent when IncludeEnv is false")
		}
	})
}

func TestFormatIssueBodyLogExcerpt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		logExcerpt  string
		sessionInfo string
		wantDetails bool
		wantSession bool
	}{
		{
			name:        "with log and session info",
			logExcerpt:  `{"level":"error","msg":"boom"}`,
			sessionInfo: "1 session(s), 5 lines",
			wantDetails: true,
			wantSession: true,
		},
		{
			name:        "with log no session info",
			logExcerpt:  `{"level":"info","msg":"ok"}`,
			sessionInfo: "",
			wantDetails: true,
			wantSession: false,
		},
		{
			name:        "no log excerpt",
			logExcerpt:  "",
			sessionInfo: "",
			wantDetails: false,
			wantSession: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &BugReport{
				Description: "desc",
				LogExcerpt:  tt.logExcerpt,
				SessionInfo: tt.sessionInfo,
			}
			body := r.FormatIssueBody()

			hasDetails := strings.Contains(body, "<details>") && strings.Contains(body, "Log excerpt")
			if hasDetails != tt.wantDetails {
				t.Errorf("log details presence = %v, want %v", hasDetails, tt.wantDetails)
			}
			if tt.wantSession {
				if !strings.Contains(body, tt.sessionInfo) {
					t.Errorf("missing session info %q", tt.sessionInfo)
				}
			}
			if tt.wantDetails && !strings.Contains(body, "```jsonl") {
				t.Error("missing jsonl code fence")
			}
		})
	}
}

func TestFormatIssueBodyLogExcerptTrailingNewline(t *testing.T) {
	t.Parallel()

	// Log excerpt without trailing newline should still be properly fenced.
	r := &BugReport{
		Description: "desc",
		LogExcerpt:  `{"msg":"no newline at end"}`,
	}
	body := r.FormatIssueBody()

	// Should contain the content followed by a newline before the closing fence.
	if !strings.Contains(body, "no newline at end\"}\n```") {
		t.Error("log excerpt without trailing newline should have newline added before fence")
	}
}

func TestFormatIssueBodyLogExcerptWithTrailingNewline(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Description: "desc",
		LogExcerpt:  "{\"msg\":\"has newline\"}\n",
	}
	body := r.FormatIssueBody()

	// Should not double the newline.
	if strings.Contains(body, "has newline\"}\n\n```\n") {
		t.Error("log excerpt with trailing newline should not get double newline")
	}
}

func TestFormatIssueBodyExtLogExcerpt(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Description:   "desc",
		ExtLogExcerpt: "--- npm (5 entries) ---\n[ERROR] ENOENT\n",
	}
	body := r.FormatIssueBody()

	if !strings.Contains(body, "External tool logs") {
		t.Error("missing external tool logs details summary")
	}
	if !strings.Contains(body, "--- npm (5 entries) ---") {
		t.Error("missing external log content")
	}
}

func TestFormatIssueBodyNoExtLogExcerpt(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Description:   "desc",
		ExtLogExcerpt: "",
	}
	body := r.FormatIssueBody()

	if strings.Contains(body, "External tool logs") {
		t.Error("external tool logs section should be absent when empty")
	}
}

func TestFormatIssueBodyFooter(t *testing.T) {
	t.Parallel()

	r := &BugReport{Description: "desc"}
	body := r.FormatIssueBody()

	if !strings.Contains(body, "---\n*Filed via `qsdev report bug`*") {
		t.Error("missing footer with branding")
	}
}

func TestFormatIssueBodyFullReport(t *testing.T) {
	t.Parallel()

	r := &BugReport{
		Title:       "Init wizard crashes on NixOS",
		Description: "Running qsdev init causes a panic when Go ecosystem is selected.",
		Steps:       "1. Run qsdev init\n2. Select Go ecosystem\n3. Observe panic",
		Severity:    "critical",
		Category:    "init-wizard",
		Environment: Environment{
			QsdevVersion: "0.8.0",
			Commit:       "abc123def4",
			GoVersion:    "go1.22.0",
			OS:           "linux",
			Arch:         "amd64",
			Family:       "nixos",
			Shell:        "zsh",
			HasNix:       true,
			DevenvVer:    "1.0.0",
			Ecosystems:   []string{"go", "node"},
		},
		IncludeEnv:    true,
		LogExcerpt:    `{"level":"error","msg":"nil pointer dereference"}`,
		SessionInfo:   "1 session(s), 42 lines",
		ExtLogExcerpt: "--- devenv (3 entries) ---\n[WARN] outdated flake\n",
	}

	body := r.FormatIssueBody()

	sections := []string{
		"## Bug Report",
		"### Description",
		"### Steps to Reproduce",
		"### Classification",
		"**Severity:** critical",
		"**Category:** init-wizard",
		"### Environment",
		"<details>",
		"Log excerpt (1 session(s), 42 lines)",
		"```jsonl",
		"External tool logs",
		"Filed via",
	}

	for _, section := range sections {
		if !strings.Contains(body, section) {
			t.Errorf("full report missing section: %q", section)
		}
	}
}
