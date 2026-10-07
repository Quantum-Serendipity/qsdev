package elixir_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/elixir"
)

// newModule returns a fresh Module for testing.
func newModule() *elixir.Module {
	return &elixir.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*elixir.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "elixir", "Elixir", 3)
}

// --- Detection tests ---

func TestDetect_MixExs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(""), 0o644); err != nil {
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
	if !containsSubstr(r.Evidence, "mix.exs") {
		t.Errorf("Evidence = %v, want entry containing %q", r.Evidence, "mix.exs")
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

// TestPreCommitHooks pins the mix-format hook to the Elixir that
// languages.elixir provides. devenv's elixir module sets the same hook's
// package to that toolchain, so a second definition from pkgs.elixir was
// both a duplicate binary and, at equal priority, an evaluation error (U10-01).
func TestPreCommitHooks(t *testing.T) {
	t.Parallel()
	hooks := newModule().PreCommitHooks(ecosystem.ModuleConfig{})
	want := []ecosystem.HookConfig{{
		ID:              "mix-format",
		Name:            "mix-format",
		Description:     "Check Elixir code formatting with mix format",
		Entry:           "mix format --check-formatted",
		Language:        "system",
		Types:           []string{"elixir"},
		Stages:          []string{"pre-commit"},
		LanguagePackage: "elixir",
	}}
	if !reflect.DeepEqual(hooks, want) {
		t.Errorf("PreCommitHooks() = %+v, want %+v", hooks, want)
	}
}

// TestDevenvNixFragment_Hex checks Mix finds the Nix-built Hex (and rebar3):
// both are Mix archives/tools mix would otherwise offer to download into
// ~/.mix, which fails or prompts on a clean CI runner. Hex is a library, not
// a binary, so the CI provisioning invariant cannot see it.
func TestDevenvNixFragment_Hex(t *testing.T) {
	t.Parallel()
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	for _, want := range []string{
		"languages.elixir.enable = true;",
		`env.MIX_PATH = "${pkgs.beamPackages.hex}/lib/erlang/lib/hex/ebin";`,
		`env.MIX_REBAR3 = "${pkgs.rebar3}/bin/rebar3";`,
	} {
		if !strings.Contains(frag, want) {
			t.Errorf("DevenvNixFragment() missing %q:\n%s", want, frag)
		}
	}
}

const (
	mixWithAudit = "defmodule X.MixProject do\n  use Mix.Project\n\n  defp deps do\n    [\n" +
		"      {:mix_audit, \"~> 2.1\", only: [:dev, :test], runtime: false}\n    ]\n  end\nend\n"
	mixWithoutAudit   = "defmodule X.MixProject do\n  defp deps, do: [{:jason, \"~> 1.4\"}]\nend\n"
	mixCommentedAudit = "defmodule X.MixProject do\n  defp deps do\n    [\n      # {:mix_audit, \"~> 2.1\"}\n    ]\n  end\nend\n"
)

// TestDetect_MixAudit checks Detect records mix_audit only when mix.exs
// declares the dependency, so `mix deps.audit` resolves.
func TestDetect_MixAudit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mix  string
		want bool
	}{
		{name: "declared", mix: mixWithAudit, want: true},
		{name: "absent", mix: mixWithoutAudit},
		{name: "commented out", mix: mixCommentedAudit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(tt.mix), 0o644); err != nil {
				t.Fatal(err)
			}
			got, ok := newModule().Detect(dir).SuggestedConfig.Extras["mix_audit"]
			if ok != tt.want || (ok && got != "true") {
				t.Errorf("Extras[mix_audit] = %q (set %v), want set %v", got, ok, tt.want)
			}
		})
	}
}

// TestCICommands_MixAuditGated checks `mix deps.audit` runs only when the
// project declares mix_audit; the locked install always runs.
func TestCICommands_MixAuditGated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		extras map[string]string
		want   []string
	}{
		{name: "not declared", want: []string{"mix deps.get --check-locked"}},
		{name: "declared", extras: map[string]string{"mix_audit": "true"}, want: []string{"mix deps.get --check-locked", "mix deps.audit"}},
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

// TestSetupWarnings_NoMixAudit checks a project without mix_audit is told its
// dependency audit does not run and what to add.
func TestSetupWarnings_NoMixAudit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mix    string
		extras map[string]string
		want   bool
	}{
		{name: "absent", mix: mixWithoutAudit, want: true},
		{name: "declared", mix: mixWithAudit},
		{name: "configured", mix: mixWithoutAudit, extras: map[string]string{"mix_audit": "true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(tt.mix), 0o644); err != nil {
				t.Fatal(err)
			}
			var w ecosystem.SetupWarner = newModule()
			got := w.SetupWarnings(dir, ecosystem.ModuleConfig{Extras: tt.extras})
			if !tt.want {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], "security scan not run") || !strings.Contains(got[0], "{:mix_audit,") {
				t.Fatalf("SetupWarnings() = %q, want one warning naming the {:mix_audit, ...} dependency", got)
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
