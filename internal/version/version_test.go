package version

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestInfoDefaults(t *testing.T) {
	info := Info()

	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}

func TestInfoStringFormat(t *testing.T) {
	info := Info()
	s := info.String()

	if !strings.Contains(s, info.GoVersion) {
		t.Errorf("String() = %q, missing GoVersion %q", s, info.GoVersion)
	}
	if !strings.Contains(s, info.OS+"/"+info.Arch) {
		t.Errorf("String() = %q, missing OS/Arch", s)
	}
}

func TestCommitTruncation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
	}{
		{"short", "abc123", 12},
		{"exact", "abcdef123456", 12},
		{"long", "abcdef1234567890abcdef", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := commit
			commit = tt.input
			defer func() { commit = old }()

			info := Info()
			if len(info.Commit) > tt.maxLen {
				t.Errorf("Commit %q has length %d, want <= %d", info.Commit, len(info.Commit), tt.maxLen)
			}
		})
	}
}

func TestCommitDirtyTruncation(t *testing.T) {
	old := commit
	commit = "abcdef1234567890abcdef-dirty"
	defer func() { commit = old }()

	info := Info()
	if !strings.HasSuffix(info.Commit, "-dirty") {
		t.Errorf("Commit %q should end with -dirty", info.Commit)
	}
	if len(info.Commit) > 18 {
		t.Errorf("Commit %q length %d, want <= 18", info.Commit, len(info.Commit))
	}
}

// TestGatewayImageDigest pins that only a well-formed index digest stamped by
// the release is reported; anything else is ignored rather than producing an
// unpullable image reference.
func TestGatewayImageDigest(t *testing.T) {
	valid := "sha256:" + strings.Repeat("0123456789abcdef", 4)
	tests := []struct {
		name    string
		stamped string
		want    string
	}{
		{"unstamped", "", ""},
		{"valid", valid, valid},
		{"padded", " " + valid + "\n", valid},
		{"uppercase hex", "sha256:" + strings.Repeat("A", 64), ""},
		{"short", "sha256:" + strings.Repeat("a", 63), ""},
		{"long", "sha256:" + strings.Repeat("a", 65), ""},
		{"other algorithm", "sha512:" + strings.Repeat("a", 64), ""},
		{"bare hex", strings.Repeat("a", 64), ""},
		{"at-prefixed", "@" + valid, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := gatewayImageDigest
			gatewayImageDigest = tt.stamped
			defer func() { gatewayImageDigest = old }()

			if got := GatewayImageDigest(); got != tt.want {
				t.Errorf("GatewayImageDigest() with %q = %q, want %q", tt.stamped, got, tt.want)
			}
		})
	}
}

// ldflagTargetRe captures the package path and variable name of a
// "-X <pkg>.<var>=<value>" linker flag.
var ldflagTargetRe = regexp.MustCompile(`-X\s+([^\s=]+)\.([A-Za-z_][A-Za-z0-9_]*)=`)

// TestGoreleaserLdflagsTargetVersionVars fails when .goreleaser.yaml stamps a
// variable this package does not declare: the linker silently ignores an -X
// whose target does not exist, so a rename here would ship an unstamped
// release.
func TestGoreleaserLdflagsTargetVersionVars(t *testing.T) {
	t.Parallel()

	stampable := map[string]*string{
		"version":            &version,
		"commit":             &commit,
		"date":               &date,
		"builtBy":            &builtBy,
		"gatewayImageDigest": &gatewayImageDigest,
	}
	pkgPath := reflect.TypeFor[BuildInfo]().PkgPath()

	b, err := os.ReadFile(filepath.Join("..", "..", ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("reading .goreleaser.yaml: %v", err)
	}
	matches := ldflagTargetRe.FindAllStringSubmatch(string(b), -1)
	if len(matches) == 0 {
		t.Fatal(".goreleaser.yaml has no -X ldflags; the build stamps nothing")
	}
	stamped := make(map[string]bool, len(matches))
	for _, m := range matches {
		if m[1] != pkgPath {
			t.Errorf("ldflag -X %s.%s targets %s, want %s (all build metadata lives in this package)", m[1], m[2], m[1], pkgPath)
			continue
		}
		if _, ok := stampable[m[2]]; !ok {
			t.Errorf("ldflag -X %s.%s names no string var in %s", m[1], m[2], pkgPath)
		}
		stamped[m[2]] = true
	}
	for name := range stampable {
		if !stamped[name] {
			t.Errorf("%s.%s is never stamped by .goreleaser.yaml", pkgPath, name)
		}
	}
}
