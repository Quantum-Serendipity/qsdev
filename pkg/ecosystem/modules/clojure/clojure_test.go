package clojure_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/clojure"
)

// newModule returns a fresh Module for testing.
func newModule() *clojure.Module {
	return &clojure.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*clojure.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "clojure", "Clojure", 3)
}

// --- Detection tests ---

func TestDetect_DepsEdn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "deps.edn"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !containsSubstr(r.Evidence, "deps.edn") {
		t.Errorf("Evidence = %v, want entry containing %q", r.Evidence, "deps.edn")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if r.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", r.Confidence)
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_NonEmpty(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
}

// --- Project-declared scanners ---

const (
	depsWithWatson = "{:deps {org.clojure/clojure {:mvn/version \"1.12.0\"}}\n" +
		" :aliases {:clj-watson {:replace-deps {io.github.clj-holmes/clj-watson {:git/tag \"v6.0.0\" :git/sha \"cb02879\"}}\n" +
		"                        :main-opts [\"-m\" \"clj-watson.cli\"]}}}\n"
	depsWithoutWatson   = "{:deps {org.clojure/clojure {:mvn/version \"1.12.0\"}}}\n"
	depsCommentedWatson = "{:deps {}\n ;; :aliases {:clj-watson {:main-opts [\"-m\" \"clj-watson.cli\"]}}\n}\n"
	leinWithNVD         = "(defproject x \"0.1.0\"\n  :plugins [[lein-nvd \"2.0.0\"]])\n"
	leinWithoutNVD      = "(defproject x \"0.1.0\"\n  :dependencies [[org.clojure/clojure \"1.12.0\"]])\n"
	leinCommentedNVD    = "(defproject x \"0.1.0\"\n  ; :plugins [[lein-nvd \"2.0.0\"]]\n  )\n"
)

// TestDetect_DeclaredScanners checks Detect records clj_watson and lein_nvd
// only when the project declares the tool, so CI never runs a scanner the
// project cannot resolve (U10-05).
func TestDetect_DeclaredScanners(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  map[string]string
	}{
		{name: "deps.edn with clj-watson alias", files: map[string]string{"deps.edn": depsWithWatson}, want: map[string]string{"clj_watson": "true"}},
		{name: "deps.edn without alias", files: map[string]string{"deps.edn": depsWithoutWatson}},
		{name: "deps.edn with commented-out alias", files: map[string]string{"deps.edn": depsCommentedWatson}},
		{name: "project.clj with lein-nvd", files: map[string]string{"project.clj": leinWithNVD}, want: map[string]string{"lein_nvd": "true"}},
		{name: "project.clj without lein-nvd", files: map[string]string{"project.clj": leinWithoutNVD}},
		{name: "project.clj with commented-out lein-nvd", files: map[string]string{"project.clj": leinCommentedNVD}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for f, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			extras := newModule().Detect(dir).SuggestedConfig.Extras
			for _, key := range []string{"clj_watson", "lein_nvd"} {
				got, ok := extras[key]
				want, wantOK := tt.want[key]
				if ok != wantOK || got != want {
					t.Errorf("Extras[%q] = %q (set %v), want %q (set %v); extras %v", key, got, ok, want, wantOK, extras)
				}
			}
		})
	}
}

// TestCICommands_DeclaredScanners checks each scanner step is emitted only
// when the project declares the tool, through its declared alias or plugin.
func TestCICommands_DeclaredScanners(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		extras map[string]string
		want   []string
	}{
		{name: "tools-deps without clj-watson"},
		{name: "tools-deps with clj-watson", extras: map[string]string{"clj_watson": "true"}, want: []string{"clojure -M:clj-watson scan -p deps.edn"}},
		{name: "leiningen without lein-nvd", extras: map[string]string{"build_tool": "leiningen"}},
		{name: "leiningen with lein-nvd", extras: map[string]string{"build_tool": "leiningen", "lein_nvd": "true"}, want: []string{"lein nvd check"}},
		{name: "tools-deps ignores lein-nvd", extras: map[string]string{"lein_nvd": "true"}},
		{name: "leiningen ignores clj-watson", extras: map[string]string{"build_tool": "leiningen", "clj_watson": "true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, c := range newModule().CICommands(ecosystem.ModuleConfig{Extras: tt.extras}) {
				got = append(got, c.Command)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CICommands commands = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCICommands_NeverToolInstall checks no configuration emits a -T tool
// invocation, which resolves (and installs) a tool the project never pinned.
func TestCICommands_NeverToolInstall(t *testing.T) {
	t.Parallel()
	for _, bt := range []string{"", "tools-deps", "leiningen"} {
		for _, declared := range []string{"", "true"} {
			cfg := ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": bt, "clj_watson": declared, "lein_nvd": declared}}
			for _, c := range newModule().CICommands(cfg) {
				for _, f := range strings.Fields(c.Command) {
					if strings.HasPrefix(f, "-T") {
						t.Errorf("build_tool=%q declared=%q: CI command %q uses %s", bt, declared, c.Command, f)
					}
				}
			}
		}
	}
}

// TestSetupWarnings_UndeclaredScanner checks a project that does not declare
// its build tool's scanner is told which snippet to add.
func TestSetupWarnings_UndeclaredScanner(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    map[string]string
		extras   map[string]string
		wantText []string // nil: no warning
	}{
		{name: "deps.edn without alias", files: map[string]string{"deps.edn": depsWithoutWatson},
			wantText: []string{"security scan not run", ":clj-watson", "io.github.clj-holmes/clj-watson", "deps.edn"}},
		{name: "deps.edn with alias", files: map[string]string{"deps.edn": depsWithWatson}},
		{name: "configured clj-watson", files: map[string]string{"deps.edn": depsWithoutWatson}, extras: map[string]string{"clj_watson": "true"}},
		{name: "project.clj without lein-nvd", files: map[string]string{"project.clj": leinWithoutNVD}, extras: map[string]string{"build_tool": "leiningen"},
			wantText: []string{"security scan not run", "lein-nvd", ":plugins", "project.clj"}},
		{name: "project.clj with lein-nvd", files: map[string]string{"project.clj": leinWithNVD}, extras: map[string]string{"build_tool": "leiningen"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for f, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var w ecosystem.SetupWarner = newModule()
			got := w.SetupWarnings(dir, ecosystem.ModuleConfig{Extras: tt.extras})
			if tt.wantText == nil {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("SetupWarnings() = %q, want exactly one warning", got)
			}
			for _, want := range tt.wantText {
				if !strings.Contains(got[0], want) {
					t.Errorf("SetupWarnings()[0] = %q, want it to mention %q", got[0], want)
				}
			}
		})
	}
}

// --- helpers ---

func containsSubstr(ss []string, substr string) bool {
	for _, s := range ss {
		if len(s) >= len(substr) && searchString(s, substr) {
			return true
		}
	}
	return false
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
