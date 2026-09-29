package devinit

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// answersFromInitArgs parses init flags and builds answers the way `init`
// does, without running the interactive wizard.
func answersFromInitArgs(t *testing.T, args ...string) types.WizardAnswers {
	t.Helper()
	cmd := &cobra.Command{Use: "init"}
	var opts InitOptions
	RegisterInitFlags(cmd, &opts)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parsing flags %v: %v", args, err)
	}
	answers, err := buildAnswersFromInputs(cmd, opts, t.TempDir(), types.DetectedProject{}, NewFlagSet(cmd))
	if err != nil {
		t.Fatalf("building answers for %v: %v", args, err)
	}
	return answers
}

// TestBuildAnswers_ProfileEnablesSelfProtection is the regression test for
// `init --profile` without --yes generating Claude Code config with no
// self-protection hook, while the same profile with --yes had one.
func TestBuildAnswers_ProfileEnablesSelfProtection(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"--profile", "go-web"},
		{"--profile", "go-web", "--yes"},
		{"--profile", "go-web", "--claude-hooks", "safety-block"},
	} {
		answers := answersFromInitArgs(t, args...)
		if !answers.ClaudeCode {
			t.Fatalf("%v: go-web should enable Claude Code", args)
		}
		if !answers.Hooks.SelfProtection {
			t.Errorf("%v: Hooks.SelfProtection = false, want true whenever Claude Code is enabled", args)
		}
	}
}

// TestBuildAnswers_ProfileKeepsFlagOnlyFields is the regression test for
// --profile dropping --env, --nix-hardening-guide, --agent-* and the profile
// name itself.
func TestBuildAnswers_ProfileKeepsFlagOnlyFields(t *testing.T) {
	t.Parallel()
	answers := answersFromInitArgs(t,
		"--profile", "go-web",
		"--env", "DATABASE_URL=postgres://localhost/app",
		"--nix-hardening-guide",
		"--agent-postmortem=true",
		"--agent-semble",
	)

	if got := answers.EnvVars["DATABASE_URL"]; got != "postgres://localhost/app" {
		t.Errorf("EnvVars[DATABASE_URL] = %q, want the --env value", got)
	}
	if !answers.NixHardeningGuide {
		t.Error("NixHardeningGuide = false, want true from --nix-hardening-guide")
	}
	if answers.ProjectTypeProfile != "go-web" {
		t.Errorf("ProjectTypeProfile = %q, want %q", answers.ProjectTypeProfile, "go-web")
	}
	if !answers.AgentTools.PostmortemEnabled {
		t.Error("AgentTools.PostmortemEnabled = false, want true from --agent-postmortem")
	}
	if !answers.AgentTools.SembleEnabled {
		t.Error("AgentTools.SembleEnabled = false, want true from --agent-semble")
	}
}

// TestBuildAnswers_ProfileKeepsAgentToolDefaults is the regression test for
// --profile without any --agent-* flag zeroing the agent tools, which turned
// off the postmortem and Version-Sentinel guardrails the flags default on.
func TestBuildAnswers_ProfileKeepsAgentToolDefaults(t *testing.T) {
	t.Parallel()
	withProfile := answersFromInitArgs(t, "--profile", "go-web").AgentTools
	withoutProfile := answersFromInitArgs(t, "--lang", "go", "--yes").AgentTools

	if withProfile != withoutProfile {
		t.Errorf("--profile AgentTools = %+v, want the flag defaults %+v", withProfile, withoutProfile)
	}
	if !withProfile.PostmortemEnabled || !withProfile.VersionSentinel {
		t.Errorf("--profile AgentTools = %+v, want postmortem and Version-Sentinel enabled by default", withProfile)
	}
}

func TestBuildAnswers_UnknownClaudeHookFails(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "init"}
	var opts InitOptions
	RegisterInitFlags(cmd, &opts)
	if err := cmd.ParseFlags([]string{"--claude-hooks", "safety_block"}); err != nil {
		t.Fatal(err)
	}
	if _, err := AnswersFromFlags(opts, t.TempDir()); err == nil {
		t.Fatal("expected an error for a misspelled --claude-hooks name")
	}
}

// TestMergeProfileWithFlags_AgentToolsPerField verifies one --agent-* flag
// does not reset the other agent settings to their flag defaults.
func TestMergeProfileWithFlags_AgentToolsPerField(t *testing.T) {
	t.Parallel()
	base := types.WizardAnswers{AgentTools: types.AgentToolsAnswers{
		PostmortemEnabled: true, VersionSentinel: false, SembleMode: "subagent",
	}}
	overrides := types.WizardAnswers{AgentTools: types.AgentToolsAnswers{
		PostmortemEnabled: true, VersionSentinel: true, SembleEnabled: true, SembleMode: "mcp",
	}}
	got := MergeProfileWithFlags(base, overrides, map[string]bool{"agent_semble": true}).AgentTools

	want := types.AgentToolsAnswers{PostmortemEnabled: true, SembleEnabled: true, SembleMode: "subagent"}
	if got != want {
		t.Errorf("AgentTools = %+v, want %+v", got, want)
	}
}

func TestMergeProfileWithFlags_EnvVarsMergePerKey(t *testing.T) {
	t.Parallel()
	base := types.WizardAnswers{EnvVars: map[string]string{"A": "base", "B": "base"}}
	overrides := types.WizardAnswers{EnvVars: map[string]string{"B": "flag", "C": "flag"}}
	got := MergeProfileWithFlags(base, overrides, map[string]bool{"env_vars": true}).EnvVars

	want := map[string]string{"A": "base", "B": "flag", "C": "flag"}
	if len(got) != len(want) {
		t.Fatalf("EnvVars = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("EnvVars[%s] = %q, want %q", k, got[k], v)
		}
	}
	if base.EnvVars["B"] != "base" {
		t.Error("merge mutated the base EnvVars map")
	}
}
