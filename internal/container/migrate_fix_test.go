package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempCompose(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp compose: %v", err)
	}
	return path
}

func TestApplyFixes_ImageQualification(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  web:
    image: nginx
`)
	issues := []MigrationIssue{{
		Category:    CategoryImageName,
		Severity:    SeverityInfo,
		File:        path,
		Service:     "web",
		AutoFixable: true,
		Fix: &MigrationFix{
			YAMLPath:  "services.web.image",
			YAMLValue: "docker.io/library/nginx",
		},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	if !strings.Contains(string(out), "docker.io/library/nginx") {
		t.Errorf("output does not contain qualified image:\n%s", out)
	}
}

func TestApplyFixes_PortRemap(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - "80:80"
      - "443:443"
`)
	issues := []MigrationIssue{
		{
			Category:    CategoryPrivPorts,
			Severity:    SeverityWarning,
			File:        path,
			Service:     "web",
			AutoFixable: true,
			Fix:         &MigrationFix{},
		},
	}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	result := string(out)
	if !strings.Contains(result, "8080:80") {
		t.Errorf("expected port 80 remapped to 8080:\n%s", result)
	}
	if !strings.Contains(result, "8443:443") {
		t.Errorf("expected port 443 remapped to 8443:\n%s", result)
	}
}

func TestApplyFixes_SocketPathReplacement(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  ci:
    image: docker.io/library/docker:dind
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
`)
	issues := []MigrationIssue{{
		Category:    CategorySocketMount,
		Severity:    SeverityWarning,
		File:        path,
		Service:     "ci",
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	if !strings.Contains(string(out), "${XDG_RUNTIME_DIR}/podman/podman.sock") {
		t.Errorf("output does not contain Podman socket path:\n%s", out)
	}
	// The host path should be replaced; the container mount point may still
	// reference the Docker socket path (that is expected and correct).
	result := string(out)
	hostReplaced := strings.Contains(result, "${XDG_RUNTIME_DIR}/podman/podman.sock:/var/run/docker.sock")
	if !hostReplaced {
		t.Errorf("host path was not replaced with Podman socket:\n%s", result)
	}
}

func TestApplyFixes_UsernsMode(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  app:
    image: docker.io/library/nginx:latest
    volumes:
      - ./src:/app
`)
	issues := []MigrationIssue{{
		Category:    CategoryVolumePerms,
		Severity:    SeverityWarning,
		File:        path,
		Service:     "app",
		AutoFixable: true,
		Fix: &MigrationFix{
			YAMLPath:  "services.app.userns_mode",
			YAMLValue: "keep-id",
		},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	if !strings.Contains(string(out), "userns_mode") || !strings.Contains(string(out), "keep-id") {
		t.Errorf("output does not contain userns_mode: keep-id:\n%s", out)
	}
}

func TestApplyFixes_SELinuxSuffix(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  app:
    image: docker.io/library/nginx:latest
    volumes:
      - ./data:/data
`)
	issues := []MigrationIssue{{
		Category:    CategorySELinux,
		Severity:    SeverityInfo,
		File:        path,
		Service:     "app",
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	if !strings.Contains(string(out), "./data:/data:Z") {
		t.Errorf("output does not contain :Z suffix:\n%s", out)
	}
}

func TestApplyFixes_SELinuxSuffix_ThreePartVolume(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  app:
    image: docker.io/library/nginx:latest
    volumes:
      - ./data:/data:rw
`)
	issues := []MigrationIssue{{
		Category:    CategorySELinux,
		Severity:    SeverityInfo,
		File:        path,
		Service:     "app",
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	result := string(out)
	if !strings.Contains(result, "./data:/data:rw,Z") {
		t.Errorf("expected comma-separated SELinux option, got:\n%s", result)
	}
	if strings.Contains(result, "./data:/data:rw:Z") {
		t.Errorf("should not use colon-separated SELinux option, got:\n%s", result)
	}
}

func TestApplyFixes_CommentPreservation(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `# Top comment
services:
  web:
    image: nginx # inline comment
`)
	issues := []MigrationIssue{{
		Category:    CategoryImageName,
		Severity:    SeverityInfo,
		File:        path,
		Service:     "web",
		AutoFixable: true,
		Fix: &MigrationFix{
			YAMLPath:  "services.web.image",
			YAMLValue: "docker.io/library/nginx",
		},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	result := string(out)
	if !strings.Contains(result, "inline comment") {
		t.Errorf("inline comment was lost:\n%s", result)
	}
	if !strings.Contains(result, "docker.io/library/nginx") {
		t.Errorf("image was not qualified:\n%s", result)
	}
}

func TestApplyFixes_MultipleFixes(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  web:
    image: nginx
    ports:
      - "80:80"
    volumes:
      - ./src:/app
`)
	issues := []MigrationIssue{
		{
			Category:    CategoryImageName,
			Severity:    SeverityInfo,
			File:        path,
			Service:     "web",
			AutoFixable: true,
			Fix:         &MigrationFix{YAMLPath: "services.web.image", YAMLValue: "docker.io/library/nginx"},
		},
		{
			Category:    CategoryPrivPorts,
			Severity:    SeverityWarning,
			File:        path,
			Service:     "web",
			AutoFixable: true,
			Fix:         &MigrationFix{},
		},
		{
			Category:    CategoryVolumePerms,
			Severity:    SeverityWarning,
			File:        path,
			Service:     "web",
			AutoFixable: true,
			Fix:         &MigrationFix{YAMLPath: "services.web.userns_mode", YAMLValue: "keep-id"},
		},
	}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	result := string(out)
	if !strings.Contains(result, "docker.io/library/nginx") {
		t.Error("image not qualified")
	}
	if !strings.Contains(result, "8080:80") {
		t.Error("port not remapped")
	}
	if !strings.Contains(result, "userns_mode") {
		t.Error("userns_mode not added")
	}
}

func TestApplyFixes_Idempotency(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  web:
    image: nginx
`)
	issues := []MigrationIssue{{
		Category:    CategoryImageName,
		Severity:    SeverityInfo,
		File:        path,
		Service:     "web",
		AutoFixable: true,
		Fix:         &MigrationFix{YAMLPath: "services.web.image", YAMLValue: "docker.io/library/nginx"},
	}}

	out1, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("first ApplyFixes() error = %v", err)
	}

	// Write the fixed output back and apply again.
	if err := os.WriteFile(path, out1, 0o644); err != nil {
		t.Fatalf("writing fixed file: %v", err)
	}

	out2, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("second ApplyFixes() error = %v", err)
	}

	if string(out1) != string(out2) {
		t.Errorf("ApplyFixes is not idempotent:\nfirst:\n%s\nsecond:\n%s", out1, out2)
	}
}

func TestRemapPort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"80:80", "8080:80"},
		{"443:443", "8443:443"},
		{"8080:80", "8080:80"},
		{"0.0.0.0:80:80", "0.0.0.0:8080:80"},
		{"80:80/tcp", "8080:80/tcp"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := remapPort(tt.input)
			if got != tt.want {
				t.Errorf("remapPort(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyFixes_NoApplicableIssues(t *testing.T) {
	t.Parallel()
	content := `services:
  web:
    image: docker.io/library/nginx:latest
`
	path := writeTempCompose(t, content)

	// Issue targets a different file.
	issues := []MigrationIssue{{
		Category:    CategoryImageName,
		Severity:    SeverityInfo,
		File:        "/nonexistent/docker-compose.yml",
		Service:     "web",
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	// Should return original content unchanged.
	if !strings.Contains(string(out), "docker.io/library/nginx:latest") {
		t.Errorf("content was unexpectedly modified:\n%s", out)
	}
}

func TestApplyFixes_NonExistentFile(t *testing.T) {
	t.Parallel()
	_, err := ApplyFixes("/nonexistent/docker-compose.yml", []MigrationIssue{{
		File:        "/nonexistent/docker-compose.yml",
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}})
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "reading") {
		t.Errorf("error should mention reading, got: %v", err)
	}
}

func TestApplyFixes_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `{{{invalid yaml`)
	_, err := ApplyFixes(path, []MigrationIssue{{
		File:        path,
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}})
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

func TestApplyFixes_MapStylePortRemap(t *testing.T) {
	t.Parallel()
	path := writeTempCompose(t, `services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - published: 80
        target: 80
`)
	issues := []MigrationIssue{{
		Category:    CategoryPrivPorts,
		Severity:    SeverityWarning,
		File:        path,
		Service:     "web",
		AutoFixable: true,
		Fix:         &MigrationFix{},
	}}

	out, err := ApplyFixes(path, issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	if !strings.Contains(string(out), "8080") {
		t.Errorf("expected remapped port 8080 in output:\n%s", out)
	}
}
