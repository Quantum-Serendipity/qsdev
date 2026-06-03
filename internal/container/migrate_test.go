package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeCompose(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

func newAnalyzeProber() *mockProber {
	return &mockProber{
		lookPathResults: map[string]string{
			"podman": "/usr/bin/podman",
		},
		outputResults: map[string]outputResult{
			"podman version --format {{.Client.Version}}":      {output: []byte("4.9.3\n")},
			"podman info --format {{.Host.Security.Rootless}}": {output: []byte("true\n")},
		},
		files: map[string][]byte{},
		env: map[string]string{
			"XDG_RUNTIME_DIR": "/run/user/1000",
		},
		user: "testuser",
	}
}

func TestAnalyze_EmptyProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(report.ComposeFiles) != 0 {
		t.Errorf("ComposeFiles = %v, want empty", report.ComposeFiles)
	}
	if len(report.Issues) != 0 {
		t.Errorf("Issues = %v, want empty", report.Issues)
	}
	if report.Summary.Total != 0 {
		t.Errorf("Summary.Total = %d, want 0", report.Summary.Total)
	}
}

func TestAnalyze_CleanCompose(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: docker.io/library/nginx:latest
    ports:
      - "8080:80"
    userns_mode: keep-id
    volumes:
      - ./data:/data
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(report.ComposeFiles) != 1 {
		t.Fatalf("ComposeFiles = %d, want 1", len(report.ComposeFiles))
	}
	if len(report.Issues) != 0 {
		t.Errorf("Issues = %v, want empty for clean compose", report.Issues)
	}
}

func TestAnalyze_UnqualifiedImage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: nginx
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategoryImageName {
			found = true
			if !issue.AutoFixable {
				t.Error("image qualification should be auto-fixable")
			}
			if issue.Severity != SeverityInfo {
				t.Errorf("image qualification severity = %v, want info", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected image qualification issue for unqualified image")
	}
}

func TestAnalyze_BindMountNoKeepID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: docker.io/library/nginx:latest
    volumes:
      - ./src:/app
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategoryVolumePerms {
			found = true
			if !issue.AutoFixable {
				t.Error("volume permissions should be auto-fixable")
			}
			if issue.Severity != SeverityWarning {
				t.Errorf("volume permissions severity = %v, want warning", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected volume permissions issue for bind mount without keep-id")
	}
}

func TestAnalyze_PrivilegedPort(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - "80:80"
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategoryPrivPorts {
			found = true
			if !issue.AutoFixable {
				t.Error("privileged port should be auto-fixable")
			}
		}
	}
	if !found {
		t.Error("expected privileged port issue for port 80")
	}
}

func TestAnalyze_DockerSocketMount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  ci:
    image: docker.io/library/docker:dind
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategorySocketMount {
			found = true
			if !issue.AutoFixable {
				t.Error("socket mount should be auto-fixable")
			}
		}
	}
	if !found {
		t.Error("expected docker socket mount issue")
	}
}

func TestAnalyze_PrivilegedContainer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  infra:
    image: docker.io/library/alpine:latest
    privileged: true
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategoryPrivileged {
			found = true
			if issue.AutoFixable {
				t.Error("privileged mode should NOT be auto-fixable")
			}
			if issue.Severity != SeverityCritical {
				t.Errorf("privileged mode severity = %v, want critical", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected privileged mode issue")
	}
}

func TestAnalyze_SELinuxMissingLabels(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()
	// Enable SELinux detection.
	prober.files["/sys/fs/selinux/enforce"] = []byte("1")

	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: docker.io/library/nginx:latest
    userns_mode: keep-id
    volumes:
      - ./data:/data
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategorySELinux {
			found = true
			if !issue.AutoFixable {
				t.Error("SELinux label should be auto-fixable")
			}
			if issue.Severity != SeverityInfo {
				t.Errorf("SELinux label severity = %v, want info", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected SELinux label issue for bind mount without :Z")
	}
}

func TestAnalyze_AlreadyCorrectVolumes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()
	prober.files["/sys/fs/selinux/enforce"] = []byte("1")

	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: docker.io/library/nginx:latest
    userns_mode: keep-id
    volumes:
      - ./data:/data:Z
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	for _, issue := range report.Issues {
		if issue.Category == CategoryVolumePerms {
			t.Error("should not flag volume permissions when userns_mode: keep-id is set")
		}
		if issue.Category == CategorySELinux {
			t.Error("should not flag SELinux when :Z suffix is present")
		}
	}
}

func TestAnalyze_MixedServices(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  clean:
    image: docker.io/library/nginx:latest
    userns_mode: keep-id
    ports:
      - "8080:80"
  dirty:
    image: redis
    privileged: true
    ports:
      - "443:443"
    volumes:
      - ./data:/data
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Clean service should have no issues.
	for _, issue := range report.Issues {
		if issue.Service == "clean" {
			t.Errorf("clean service should have no issues, got: %s", issue.Category)
		}
	}

	// Dirty service should have multiple issues.
	dirtyIssues := 0
	for _, issue := range report.Issues {
		if issue.Service == "dirty" {
			dirtyIssues++
		}
	}
	if dirtyIssues < 3 {
		t.Errorf("dirty service has %d issues, want at least 3 (image, privileged, port/volume)", dirtyIssues)
	}
}

func TestAnalyze_PortAsInteger(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	// YAML integers in ports list are valid compose syntax.
	writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - 80
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	// An integer port like "80" is a container port, not a host mapping.
	// extractHostPort returns the int directly, but parseHostPortFromString
	// treats single values as container-only. For integer entries, they are
	// host port bindings.
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategoryPrivPorts {
			found = true
		}
	}
	if !found {
		t.Error("expected privileged port issue for integer port 80")
	}
}

func TestAnalyze_PortAsMap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - published: 443
        target: 443
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Category == CategoryPrivPorts {
			found = true
		}
	}
	if !found {
		t.Error("expected privileged port issue for map-style port 443")
	}
}

func TestAnalyze_NamedVolumeNoFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
volumes:
  dbdata:

services:
  db:
    image: docker.io/library/postgres:15
    volumes:
      - dbdata:/var/lib/postgresql/data
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	for _, issue := range report.Issues {
		if issue.Category == CategoryVolumePerms && issue.Service == "db" {
			t.Error("named volumes should not trigger volume permission issues")
		}
	}
}

func TestAnalyze_MultipleComposeFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	prober := newAnalyzeProber()

	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: nginx
`)
	writeCompose(t, dir, "compose.yaml", `
services:
  worker:
    image: redis
`)

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(report.ComposeFiles) != 2 {
		t.Errorf("ComposeFiles = %d, want 2", len(report.ComposeFiles))
	}

	// Both files should contribute image issues.
	imgIssues := 0
	for _, issue := range report.Issues {
		if issue.Category == CategoryImageName {
			imgIssues++
		}
	}
	if imgIssues != 2 {
		t.Errorf("image qualification issues = %d, want 2", imgIssues)
	}
}

func TestBuildSummary(t *testing.T) {
	t.Parallel()
	issues := []MigrationIssue{
		{Severity: SeverityCritical, AutoFixable: false},
		{Severity: SeverityWarning, AutoFixable: true},
		{Severity: SeverityWarning, AutoFixable: true},
		{Severity: SeverityInfo, AutoFixable: true},
	}

	s := buildSummary(issues)
	if s.Total != 4 {
		t.Errorf("Total = %d, want 4", s.Total)
	}
	if s.Critical != 1 {
		t.Errorf("Critical = %d, want 1", s.Critical)
	}
	if s.Warning != 2 {
		t.Errorf("Warning = %d, want 2", s.Warning)
	}
	if s.Info != 1 {
		t.Errorf("Info = %d, want 1", s.Info)
	}
	if s.AutoFixable != 3 {
		t.Errorf("AutoFixable = %d, want 3", s.AutoFixable)
	}
	if s.ManualOnly != 1 {
		t.Errorf("ManualOnly = %d, want 1", s.ManualOnly)
	}
}

func TestQualifyImageName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"nginx", "docker.io/library/nginx"},
		{"nginx:latest", "docker.io/library/nginx:latest"},
		{"bitnami/redis", "docker.io/bitnami/redis"},
		{"bitnami/redis:7.2", "docker.io/bitnami/redis:7.2"},
		{"docker.io/library/nginx", "docker.io/library/nginx"},
		{"ghcr.io/owner/image:tag", "ghcr.io/owner/image:tag"},
		{"localhost/myimage", "localhost/myimage"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := qualifyImageName(tt.input)
			if got != tt.want {
				t.Errorf("qualifyImageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsBindMount(t *testing.T) {
	t.Parallel()
	topLevel := map[string]bool{"dbdata": true}
	tests := []struct {
		vol  string
		want bool
	}{
		{"./src:/app", true},
		{"/data:/data", true},
		{"~/data:/data", true},
		{"dbdata:/var/lib/data", false},
		{"unknownvol:/data", false},
	}
	for _, tt := range tests {
		t.Run(tt.vol, func(t *testing.T) {
			t.Parallel()
			got := isBindMount(tt.vol, topLevel)
			if got != tt.want {
				t.Errorf("isBindMount(%q) = %v, want %v", tt.vol, got, tt.want)
			}
		})
	}
}

func TestParseHostPortFromString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{"80:80", 80},
		{"8080:80", 8080},
		{"0.0.0.0:443:443", 443},
		{"80", 0},         // container-only
		{"80:80/tcp", 80}, // with protocol
		{"8080:80/udp", 8080},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := parseHostPortFromString(tt.input)
			if got != tt.want {
				t.Errorf("parseHostPortFromString(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
