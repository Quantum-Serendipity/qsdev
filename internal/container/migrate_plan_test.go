package container

import (
	"context"
	"strings"
	"testing"
)

// analyzeAndFix runs the analyzer on content with SELinux enforcing, applies
// the auto-fixes, and returns the report and the fixed file.
func analyzeAndFix(t *testing.T, content string) (*MigrationReport, string) {
	t.Helper()
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", content)
	prober := newAnalyzeProber()
	prober.files["/sys/fs/selinux/enforce"] = []byte("1")

	report, err := Analyze(context.Background(), dir, prober)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(report.ComposeFiles) != 1 {
		t.Fatalf("compose files = %v, want 1", report.ComposeFiles)
	}
	out, err := ApplyFixes(report.ComposeFiles[0], report.Issues)
	if err != nil {
		t.Fatalf("ApplyFixes() error = %v", err)
	}
	return report, string(out)
}

// issueFor returns the issue of the given category whose description mentions
// needle.
func issueFor(t *testing.T, report *MigrationReport, category IssueCategory, needle string) MigrationIssue {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Category == category && strings.Contains(issue.Description, needle) {
			return issue
		}
	}
	t.Fatalf("no %s issue mentioning %q in %+v", category, needle, report.Issues)
	return MigrationIssue{}
}

// TestSELinuxFix_OnlyRelabelsProjectPaths is the regression for relabelling
// host paths: :Z recursively relabels the host path with a label private to one
// container, which breaks the host for system paths, the home directory and
// sockets. Only project paths may be relabelled automatically; the rest are
// reported for a manual decision.
func TestSELinuxFix_OnlyRelabelsProjectPaths(t *testing.T) {
	t.Parallel()

	report, out := analyzeAndFix(t, `
services:
  app:
    image: docker.io/library/nginx:latest
    userns_mode: keep-id
    volumes:
      - ./data:/data
      - /etc/localtime:/etc/localtime:ro
      - ~/.:/home
      - /run/user/1000/podman/podman.sock:/run/podman.sock
      - ../outside:/outside
`)

	if !strings.Contains(out, "./data:/data:Z") {
		t.Errorf("project path not relabelled:\n%s", out)
	}
	for _, untouched := range []string{
		"/etc/localtime:/etc/localtime:ro\n",
		"~/.:/home\n",
		"/run/user/1000/podman/podman.sock:/run/podman.sock\n",
		"../outside:/outside\n",
	} {
		if !strings.Contains(out, untouched) {
			t.Errorf("%q must not be relabelled:\n%s", strings.TrimSpace(untouched), out)
		}
	}

	for _, host := range []string{"/etc/localtime", "~/.", "/run/user/1000/podman/podman.sock", "../outside"} {
		if issue := issueFor(t, report, CategorySELinux, host); issue.AutoFixable {
			t.Errorf("SELinux issue for %q is auto-fixable, want manual", host)
		}
	}
	if issue := issueFor(t, report, CategorySELinux, "./data"); !issue.AutoFixable {
		t.Error("SELinux issue for ./data should be auto-fixable")
	}
}

// TestSELinuxFix_SharedPathGetsSharedLabel pins that a project path mounted by
// several services gets the shared :z label; a private :Z label would leave it
// accessible only to the last container started.
func TestSELinuxFix_SharedPathGetsSharedLabel(t *testing.T) {
	t.Parallel()

	_, out := analyzeAndFix(t, `
services:
  web:
    image: docker.io/library/nginx:latest
    userns_mode: keep-id
    volumes:
      - ./shared:/srv
      - ./web-only:/cache
  worker:
    image: docker.io/library/nginx:latest
    userns_mode: keep-id
    volumes:
      - ./shared/:/work:ro
`)

	for _, want := range []string{"./shared:/srv:z", "./shared/:/work:ro,z", "./web-only:/cache:Z"} {
		if !strings.Contains(out, want) {
			t.Errorf("fixed file missing %q:\n%s", want, out)
		}
	}
}

// TestPortRemap_AvoidsPublishedPorts is the regression for remap collisions:
// port 80 must not become 8080 when another service already publishes 8080.
func TestPortRemap_AvoidsPublishedPorts(t *testing.T) {
	t.Parallel()

	report, out := analyzeAndFix(t, `
services:
  web:
    image: docker.io/library/nginx:latest
    ports:
      - "80:80"
      - "443:443"
  admin:
    image: docker.io/library/nginx:latest
    ports:
      - "8080:8080"
  dns:
    image: docker.io/library/nginx:latest
    ports:
      - "53:53/udp"
      - target: 22
        published: 22
`)

	if !strings.Contains(out, `"80:80"`) {
		t.Errorf("port 80 remapped onto the already-published 8080:\n%s", out)
	}
	if strings.Count(out, "8080:") != 1 {
		t.Errorf("host port 8080 published more than once:\n%s", out)
	}
	for _, want := range []string{`"8443:443"`, `"8053:53/udp"`, "published: 8022"} {
		if !strings.Contains(out, want) {
			t.Errorf("fixed file missing %s:\n%s", want, out)
		}
	}

	if issue := issueFor(t, report, CategoryPrivPorts, "port 80;"); issue.AutoFixable {
		t.Errorf("colliding remap of port 80 is auto-fixable, want manual: %+v", issue)
	}
	if issue := issueFor(t, report, CategoryPrivPorts, "port 443"); !issue.AutoFixable {
		t.Errorf("remap of port 443 should be auto-fixable: %+v", issue)
	}
}

func TestPortRemapTarget(t *testing.T) {
	t.Parallel()

	published := map[publishedPort]bool{
		{port: 8080, proto: "tcp"}: true,
		{port: 8053, proto: "udp"}: true,
	}
	tests := []struct {
		name   string
		in     publishedPort
		want   int
		wantOK bool
	}{
		{name: "free target", in: publishedPort{port: 443, proto: "tcp"}, want: 8443, wantOK: true},
		{name: "taken target", in: publishedPort{port: 80, proto: "tcp"}, wantOK: false},
		{name: "same number other protocol is free", in: publishedPort{port: 80, proto: "udp"}, want: 8080, wantOK: true},
		{name: "taken udp target", in: publishedPort{port: 53, proto: "udp"}, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := portRemapTarget(tt.in, published)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("portRemapTarget(%+v) = (%d, %v), want (%d, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
