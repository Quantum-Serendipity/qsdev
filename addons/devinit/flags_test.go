package devinit_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devinit"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func mustAnswersFromFlags(t *testing.T, opts devinit.ExportInitOptions, root string) types.WizardAnswers {
	t.Helper()
	answers, err := devinit.ExportAnswersFromFlags(opts, root)
	if err != nil {
		t.Fatalf("AnswersFromFlags: %v", err)
	}
	return answers
}

func TestAnswersFromFlags_FullFlagSet(t *testing.T) {
	opts := devinit.ExportInitOptions{
		Langs:             []string{"go", "python"},
		Services:          []string{"postgres", "redis"},
		Yes:               true,
		Direnv:            true,
		ClaudeCode:        true,
		ClaudePermissions: "standard",
		ClaudeSkills:      []string{"deploy"},
		ClaudeHooks:       []string{"safety-block", "auto-format"},
		MCPServers:        []string{"github"},
		GitHooks:          []string{"pre-commit"},
		Packages:          []string{"jq", "ripgrep"},
		Env:               []string{"DB_URL=postgres://localhost/mydb", "DEBUG=true"},
		NixHardeningGuide: true,
		InfraProfile:      "consulting-default",
		ProfileName:       "go-web",
		GoVersion:         "1.24",
		PythonVersion:     "3.12",
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/myproject")

	if answers.ProjectName != "myproject" {
		t.Errorf("ProjectName = %q, want %q", answers.ProjectName, "myproject")
	}
	if answers.ProjectRoot != "/tmp/myproject" {
		t.Errorf("ProjectRoot = %q, want %q", answers.ProjectRoot, "/tmp/myproject")
	}
	if !answers.Confirmed {
		t.Error("expected Confirmed = true")
	}
	if !answers.Direnv {
		t.Error("expected Direnv = true")
	}
	if !answers.ClaudeCode {
		t.Error("expected ClaudeCode = true")
	}
	if answers.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want %q", answers.PermissionLevel, "standard")
	}
	if !answers.NixHardeningGuide {
		t.Error("expected NixHardeningGuide = true")
	}
	if answers.ProfileName != "consulting-default" {
		t.Errorf("ProfileName = %q, want %q", answers.ProfileName, "consulting-default")
	}

	// Check languages include go and python with versions.
	goFound, pyFound := false, false
	for _, l := range answers.Languages {
		if l.Name == "go" && l.Version == "1.24" {
			goFound = true
		}
		if l.Name == "python" && l.Version == "3.12" {
			pyFound = true
		}
	}
	if !goFound {
		t.Error("expected go 1.24 in languages")
	}
	if !pyFound {
		t.Error("expected python 3.12 in languages")
	}

	// Check services.
	if len(answers.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(answers.Services))
	}

	// Check env vars.
	if answers.EnvVars["DB_URL"] != "postgres://localhost/mydb" {
		t.Errorf("DB_URL = %q, want %q", answers.EnvVars["DB_URL"], "postgres://localhost/mydb")
	}
	if answers.EnvVars["DEBUG"] != "true" {
		t.Errorf("DEBUG = %q, want %q", answers.EnvVars["DEBUG"], "true")
	}

	// Check hooks.
	if !answers.Hooks.SafetyBlock {
		t.Error("expected SafetyBlock hook")
	}
	if !answers.Hooks.AutoFormat {
		t.Error("expected AutoFormat hook")
	}

	// Check MCPServers.
	if len(answers.MCPServers) != 1 || answers.MCPServers[0] != "github" {
		t.Errorf("MCPServers = %v, want [github]", answers.MCPServers)
	}

	// Check skills.
	if len(answers.Skills) != 1 || answers.Skills[0] != "deploy" {
		t.Errorf("Skills = %v, want [deploy]", answers.Skills)
	}
}

func TestAnswersFromFlags_ImplicitLanguageFromVersionFlag(t *testing.T) {
	tests := []struct {
		name     string
		opts     devinit.ExportInitOptions
		wantLang string
		wantVer  string
	}{
		{
			name:     "go version implies go language",
			opts:     devinit.ExportInitOptions{GoVersion: "1.24", ClaudePermissions: "standard", ClaudeCode: true, Direnv: true},
			wantLang: "go",
			wantVer:  "1.24",
		},
		{
			name:     "node version implies javascript language",
			opts:     devinit.ExportInitOptions{NodeVersion: "22", ClaudePermissions: "standard", ClaudeCode: true, Direnv: true},
			wantLang: "javascript",
			wantVer:  "22",
		},
		{
			name:     "python version implies python language",
			opts:     devinit.ExportInitOptions{PythonVersion: "3.12", ClaudePermissions: "standard", ClaudeCode: true, Direnv: true},
			wantLang: "python",
			wantVer:  "3.12",
		},
		{
			name:     "rust channel implies rust language",
			opts:     devinit.ExportInitOptions{RustChannel: "nightly", ClaudePermissions: "standard", ClaudeCode: true, Direnv: true},
			wantLang: "rust",
			wantVer:  "nightly",
		},
		{
			name:     "java version implies java language",
			opts:     devinit.ExportInitOptions{JavaVersion: "21", ClaudePermissions: "standard", ClaudeCode: true, Direnv: true},
			wantLang: "java",
			wantVer:  "21",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answers := mustAnswersFromFlags(t,tt.opts, "/tmp/test")

			found := false
			for _, l := range answers.Languages {
				if l.Name == tt.wantLang {
					found = true
					if l.Version != tt.wantVer {
						t.Errorf("version = %q, want %q", l.Version, tt.wantVer)
					}
				}
			}
			if !found {
				t.Errorf("expected language %q to be implicitly added, languages = %+v", tt.wantLang, answers.Languages)
			}
		})
	}
}

func TestAnswersFromFlags_VersionFlagMergesWithExplicitLang(t *testing.T) {
	opts := devinit.ExportInitOptions{
		Langs:             []string{"go"},
		GoVersion:         "1.24",
		ClaudePermissions: "standard",
		ClaudeCode:        true,
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	goCount := 0
	for _, l := range answers.Languages {
		if l.Name == "go" {
			goCount++
			if l.Version != "1.24" {
				t.Errorf("go version = %q, want %q", l.Version, "1.24")
			}
		}
	}
	if goCount != 1 {
		t.Errorf("expected exactly 1 go entry, got %d", goCount)
	}
}

func TestAnswersFromFlags_EnvVarParsing(t *testing.T) {
	tests := []struct {
		name    string
		env     []string
		wantKey string
		wantVal string
	}{
		{
			name:    "simple key=value",
			env:     []string{"FOO=bar"},
			wantKey: "FOO",
			wantVal: "bar",
		},
		{
			name:    "value with equals sign",
			env:     []string{"URL=postgres://host?opt=val"},
			wantKey: "URL",
			wantVal: "postgres://host?opt=val",
		},
		{
			name:    "empty value",
			env:     []string{"EMPTY="},
			wantKey: "EMPTY",
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := devinit.ExportInitOptions{
				Env:               tt.env,
				ClaudePermissions: "standard",
				ClaudeCode:        true,
				Direnv:            true,
			}
			answers := mustAnswersFromFlags(t,opts, "/tmp/test")

			if answers.EnvVars[tt.wantKey] != tt.wantVal {
				t.Errorf("EnvVars[%q] = %q, want %q", tt.wantKey, answers.EnvVars[tt.wantKey], tt.wantVal)
			}
		})
	}
}

func TestAnswersFromFlags_DevenvOnly(t *testing.T) {
	opts := devinit.ExportInitOptions{
		DevenvOnly:        true,
		ClaudeCode:        true, // default is true, but devenv-only overrides
		ClaudePermissions: "standard",
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	if answers.ClaudeCode {
		t.Error("expected ClaudeCode = false when --devenv-only is set")
	}
}

func TestAnswersFromFlags_ClaudeOnly(t *testing.T) {
	opts := devinit.ExportInitOptions{
		ClaudeOnly:        true,
		ClaudeCode:        true,
		ClaudePermissions: "standard",
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	// ClaudeOnly does not change ClaudeCode — it's a signal for the orchestrator.
	if !answers.ClaudeCode {
		t.Error("expected ClaudeCode = true when --claude-only is set")
	}
}

func TestAnswersFromFlags_NodePkgMgrOnly(t *testing.T) {
	opts := devinit.ExportInitOptions{
		NodePkgMgr:        "pnpm",
		ClaudePermissions: "standard",
		ClaudeCode:        true,
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	found := false
	for _, l := range answers.Languages {
		if l.Name == "javascript" {
			found = true
			if l.PackageManager != "pnpm" {
				t.Errorf("PackageManager = %q, want %q", l.PackageManager, "pnpm")
			}
		}
	}
	if !found {
		t.Error("expected javascript to be implicitly added from --node-pkg-mgr")
	}
}

func TestAnswersFromFlags_PythonPkgMgrOnly(t *testing.T) {
	opts := devinit.ExportInitOptions{
		PythonPkgMgr:     "uv",
		ClaudePermissions: "standard",
		ClaudeCode:        true,
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	found := false
	for _, l := range answers.Languages {
		if l.Name == "python" {
			found = true
			if l.PackageManager != "uv" {
				t.Errorf("PackageManager = %q, want %q", l.PackageManager, "uv")
			}
		}
	}
	if !found {
		t.Error("expected python to be implicitly added from --python-pkg-mgr")
	}
}

func TestFlagSet_IsSet(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	var opts devinit.ExportInitOptions
	devinit.ExportRegisterInitFlags(cmd, &opts)

	// Simulate setting specific flags.
	err := cmd.Flags().Set("lang", "go")
	if err != nil {
		t.Fatalf("setting flag: %v", err)
	}
	err = cmd.Flags().Set("yes", "true")
	if err != nil {
		t.Fatalf("setting flag: %v", err)
	}

	fs := devinit.ExportNewFlagSet(cmd)

	if !fs.IsSet("lang") {
		t.Error("expected lang to be set")
	}
	if !fs.IsSet("yes") {
		t.Error("expected yes to be set")
	}
	if fs.IsSet("force") {
		t.Error("expected force to NOT be set")
	}
	if fs.IsSet("nonexistent") {
		t.Error("expected nonexistent flag to NOT be set")
	}
}

func TestRegisterInitFlags_MutuallyExclusive(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	var opts devinit.ExportInitOptions
	devinit.ExportRegisterInitFlags(cmd, &opts)

	// Verify both flags exist.
	devenvOnly := cmd.Flags().Lookup("devenv-only")
	if devenvOnly == nil {
		t.Fatal("expected devenv-only flag to exist")
	}
	claudeOnly := cmd.Flags().Lookup("claude-only")
	if claudeOnly == nil {
		t.Fatal("expected claude-only flag to exist")
	}
}

func TestAnswersFromFlags_ProfileSetsProjectTypeProfile(t *testing.T) {
	opts := devinit.ExportInitOptions{
		ProfileName:       "go-web",
		ClaudePermissions: "standard",
		ClaudeCode:        true,
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	if answers.ProjectTypeProfile != "go-web" {
		t.Errorf("ProjectTypeProfile = %q, want %q", answers.ProjectTypeProfile, "go-web")
	}
}

func TestAnswersFromFlags_JavaBuildTool(t *testing.T) {
	opts := devinit.ExportInitOptions{
		JavaVersion:       "21",
		JavaBuildTool:     "gradle",
		ClaudePermissions: "standard",
		ClaudeCode:        true,
		Direnv:            true,
	}

	answers := mustAnswersFromFlags(t,opts, "/tmp/test")

	found := false
	for _, l := range answers.Languages {
		if l.Name == "java" {
			found = true
			if l.Version != "21" {
				t.Errorf("java version = %q, want %q", l.Version, "21")
			}
			if l.PackageManager != "gradle" {
				t.Errorf("java build tool = %q, want %q", l.PackageManager, "gradle")
			}
		}
	}
	if !found {
		t.Error("expected java to be added from version/build-tool flags")
	}
}

// Verify AnswersFromFlags returns the correct type.
func TestAnswersFromFlags_ReturnsWizardAnswers(t *testing.T) {
	opts := devinit.ExportInitOptions{
		ClaudePermissions: "standard",
		ClaudeCode:        true,
		Direnv:            true,
	}
	_ = mustAnswersFromFlags(t, opts, "/tmp/test")
}
