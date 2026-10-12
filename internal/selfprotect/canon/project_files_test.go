package canon

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestSegments_MatchProtection pins that Segments lists exactly the segments
// IsProtected matches, so the hook sandbox's guardrails cover them all.
func TestSegments_MatchProtection(t *testing.T) {
	t.Parallel()
	segs := Segments()
	if len(segs) != len(brandedTables().segments) {
		t.Fatalf("Segments() has %d entries, the tables %d", len(segs), len(brandedTables().segments))
	}
	for _, want := range []string{HookLogDir + "/", ".envrc"} {
		if !slices.Contains(segs, want) {
			t.Errorf("Segments() = %v, missing %q", segs, want)
		}
	}
}

// TestSegments_ComposerConfigFilesBranded pins that the Composer home's
// policy files are protected as files, not as a directory, under the name the
// branding derives, and that the command scan names the same two files.
func TestSegments_ComposerConfigFilesBranded(t *testing.T) {
	t.Parallel()
	cfg := branding.Default()
	cfg.AppName = "acme"
	tables := newPathTables(cfg, func(string) string { return "" })

	files := []string{".acme-composer/config.json", ".acme-composer/.gitignore"}
	for _, f := range files {
		if !slices.Contains(tables.segments, segmentEntry{f, "config"}) {
			t.Errorf("segments miss %q (config); got %v", f, tables.segments)
		}
		if !slices.Contains(tables.substrings, f) {
			t.Errorf("substrings miss %q; got %v", f, tables.substrings)
		}
	}
	for _, seg := range tables.segments {
		if strings.HasPrefix(seg.segment, ".acme-composer") && !slices.Contains(files, seg.segment) {
			t.Errorf("segment %q protects more of the Composer home than its policy files", seg.segment)
		}
	}
	for _, s := range slices.Concat(tables.substrings, tables.tokens) {
		if s == ".acme-composer/" || s == ".acme-composer" {
			t.Errorf("command scan names the whole Composer home %q", s)
		}
	}
}

// TestIsPolicyDir_ComposerHome pins that the Composer home, which holds the
// protected policy files, is a policy directory under every spelling of its
// own path, while its entries and lookalikes are not, and that the .devenv
// runtime directory (whose answers mirror has a protected token of its own)
// is not one.
func TestIsPolicyDir_ComposerHome(t *testing.T) {
	t.Parallel()
	cfg := branding.Default()
	cfg.AppName = "acme"
	tables := newPathTables(cfg, func(string) string { return "" })
	exact := pathmatch.Options{}
	fold := pathmatch.Options{FoldCase: true}

	tests := []struct {
		p    string
		opts pathmatch.Options
		want bool
	}{
		{".acme-composer", exact, true},
		{".acme-composer/", exact, true},
		{"./.acme-composer/.", exact, true},
		{"/work/repo/.acme-composer", exact, true},
		{"/work/repo/.ACME-Composer", fold, true},
		{"/work/repo/.ACME-Composer", exact, false},
		{"/work/repo/.acme-composer/cache", exact, false},
		{"/work/repo/.acme-composer/config.json", exact, false},
		{"/work/repo/x.acme-composer", exact, false},
		{"/work/repo/.acme-composer-old", exact, false},
		{"/work/repo/.devenv", exact, false},
	}
	for _, tt := range tests {
		if got := tables.isPolicyDir(filepath.FromSlash(tt.p), tt.opts); got != tt.want {
			t.Errorf("isPolicyDir(%q, %+v) = %v, want %v", tt.p, tt.opts, got, tt.want)
		}
	}
	if want := []string{".acme-composer/config.json", ".acme-composer/.gitignore"}; !slices.Equal(tables.policyDirFiles, want) {
		t.Errorf("policyDirFiles = %v, want %v", tables.policyDirFiles, want)
	}
}

// TestDevenvSourceFiles_AreEnvSources pins that the project's devenv files
// are environment sources and not the shell startup files of a home
// directory.
func TestDevenvSourceFiles_AreEnvSources(t *testing.T) {
	t.Parallel()
	env := EnvSourceFiles()
	for _, f := range DevenvSourceFiles() {
		if !slices.Contains(env, f) {
			t.Errorf("devenv source %q is not in EnvSourceFiles() %v", f, env)
		}
	}
	if slices.Contains(DevenvSourceFiles(), ".bashrc") {
		t.Error("DevenvSourceFiles lists a shell startup file")
	}
}

func TestGuardedConfigFiles_Sorted(t *testing.T) {
	t.Parallel()
	got := GuardedConfigFiles()
	if !slices.IsSorted(got) || len(got) == 0 {
		t.Errorf("GuardedConfigFiles() = %v, want a sorted non-empty list", got)
	}
}
