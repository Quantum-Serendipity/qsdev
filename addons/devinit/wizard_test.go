package devinit_test

import (
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestMapFormToAnswers_QuickPath(t *testing.T) {
	detected := types.DetectedProject{
		HasGoMod:  true,
		GoVersion: "1.24",
	}

	fs := devinit.NewExportFormState(
		devinit.WithQuickChoice("yes"),
		devinit.WithConfirmed(true),
		devinit.WithClaudeCode(true),
		devinit.WithDirenv(true),
		devinit.WithPermissionLevel("standard"),
	)

	answers := devinit.ExportMapFormToAnswers(fs, "/tmp/project", "myproject", detected)

	if !answers.Confirmed {
		t.Error("expected Confirmed=true on quick path")
	}
	if answers.QuickChoice != "yes" {
		t.Errorf("expected QuickChoice=%q, got %q", "yes", answers.QuickChoice)
	}
	// On quick path, FillDefaults should populate languages from detected.
	if len(answers.Languages) == 0 {
		t.Error("expected at least one language from FillDefaults on quick path")
	}
	found := false
	for _, lang := range answers.Languages {
		if lang.Name == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Go language from FillDefaults when HasGoMod=true")
	}
}

func TestMapFormToAnswers_CustomizePath(t *testing.T) {
	detected := types.DetectedProject{}

	fs := devinit.NewExportFormState(
		devinit.WithQuickChoice("customize"),
		devinit.WithConfirmed(true),
		devinit.WithSelectedLanguages([]string{"go", "python"}),
		devinit.WithGoVersion("1.24"),
		devinit.WithPythonVersion("3.12"),
		devinit.WithSelectedServices([]string{"postgres", "redis"}),
		devinit.WithDirenv(true),
		devinit.WithClaudeCode(true),
		devinit.WithPermissionLevel("minimal"),
		devinit.WithSkills([]string{"deploy"}),
		devinit.WithMCPServers([]string{"github"}),
	)

	answers := devinit.ExportMapFormToAnswers(fs, "/tmp/project", "myproject", detected)

	if len(answers.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "go" || answers.Languages[0].Version != "1.24" {
		t.Errorf("expected Go 1.24, got %s %s", answers.Languages[0].Name, answers.Languages[0].Version)
	}
	if answers.Languages[1].Name != "python" || answers.Languages[1].Version != "3.12" {
		t.Errorf("expected Python 3.12, got %s %s", answers.Languages[1].Name, answers.Languages[1].Version)
	}

	if len(answers.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(answers.Services))
	}
	if answers.Services[0].Name != "postgres" {
		t.Errorf("expected postgres service, got %s", answers.Services[0].Name)
	}
	if answers.Services[1].Name != "redis" {
		t.Errorf("expected redis service, got %s", answers.Services[1].Name)
	}

	if answers.PermissionLevel != "minimal" {
		t.Errorf("expected permission level %q, got %q", "minimal", answers.PermissionLevel)
	}
	if len(answers.Skills) != 1 || answers.Skills[0] != "deploy" {
		t.Errorf("expected skills [deploy], got %v", answers.Skills)
	}
	if len(answers.MCPServers) != 1 || answers.MCPServers[0] != "github" {
		t.Errorf("expected MCP servers [github], got %v", answers.MCPServers)
	}
}

func TestMapFormToAnswers_ExtraPackagesParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"comma separated", "jq, ripgrep, fd", []string{"jq", "ripgrep", "fd"}},
		{"no spaces", "jq,ripgrep,fd", []string{"jq", "ripgrep", "fd"}},
		{"trailing comma", "jq, ripgrep,", []string{"jq", "ripgrep"}},
		{"single item", "jq", []string{"jq"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := devinit.ExportParseExtraPackages(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseExtraPackages(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.expected, len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseExtraPackages(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMapFormToAnswers_EmptyExtraPackages(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"commas only", ",,,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := devinit.ExportParseExtraPackages(tt.input)
			if got != nil {
				t.Errorf("parseExtraPackages(%q) = %v, want nil", tt.input, got)
			}
		})
	}
}

func TestMapFormToAnswers_HookMapping(t *testing.T) {
	detected := types.DetectedProject{}

	fs := devinit.NewExportFormState(
		devinit.WithQuickChoice("customize"),
		devinit.WithConfirmed(true),
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithAutoFormat(true),
		devinit.WithSafetyBlock(true),
		devinit.WithClaudeCode(true),
		devinit.WithPermissionLevel("standard"),
	)

	answers := devinit.ExportMapFormToAnswers(fs, "/tmp/project", "myproject", detected)

	if !answers.Hooks.AutoFormat {
		t.Error("expected Hooks.AutoFormat=true")
	}
	if !answers.Hooks.SafetyBlock {
		t.Error("expected Hooks.SafetyBlock=true")
	}
}

func TestMapFormToAnswers_ClaudeDisabled(t *testing.T) {
	detected := types.DetectedProject{}

	fs := devinit.NewExportFormState(
		devinit.WithQuickChoice("customize"),
		devinit.WithConfirmed(true),
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithClaudeCode(false),
		devinit.WithSkills([]string{"deploy"}),
		devinit.WithMCPServers([]string{"github"}),
	)

	answers := devinit.ExportMapFormToAnswers(fs, "/tmp/project", "myproject", detected)

	if answers.ClaudeCode {
		t.Error("expected ClaudeCode=false")
	}
	if len(answers.Skills) != 0 {
		t.Errorf("expected empty Skills when Claude disabled, got %v", answers.Skills)
	}
	if len(answers.MCPServers) != 0 {
		t.Errorf("expected empty MCPServers when Claude disabled, got %v", answers.MCPServers)
	}
}

func TestIsAccessible_NoEnv(t *testing.T) {
	origAccessible := os.Getenv("ACCESSIBLE")
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("ACCESSIBLE")
	os.Unsetenv("NO_COLOR")
	t.Cleanup(func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		}
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		}
	})

	if devinit.ExportIsAccessible() {
		t.Error("isAccessible() = true, want false when no env vars set")
	}
}

func TestIsAccessible_AccessibleEnv(t *testing.T) {
	origAccessible := os.Getenv("ACCESSIBLE")
	os.Setenv("ACCESSIBLE", "1")
	t.Cleanup(func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	})

	if !devinit.ExportIsAccessible() {
		t.Error("isAccessible() = false, want true when ACCESSIBLE is set")
	}
}

func TestIsAccessible_NoColorEnv(t *testing.T) {
	origAccessible := os.Getenv("ACCESSIBLE")
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("ACCESSIBLE")
	os.Setenv("NO_COLOR", "1")
	t.Cleanup(func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		}
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
	})

	if !devinit.ExportIsAccessible() {
		t.Error("isAccessible() = false, want true when NO_COLOR is set")
	}
}

func TestBuildWizardForm_ReturnsNonNil(t *testing.T) {
	detected := types.DetectedProject{
		HasGoMod:  true,
		GoVersion: "1.24",
	}

	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithDirenv(true),
		devinit.WithClaudeCode(true),
		devinit.WithPermissionLevel("standard"),
	)

	cmd := &cobra.Command{}
	var opts devinit.ExportInitOptions
	devinit.ExportRegisterInitFlags(cmd, &opts)
	_ = cmd.ParseFlags(nil)
	flagSet := devinit.ExportNewFlagSet(cmd)

	form := devinit.ExportBuildWizardForm(detected, fs, flagSet, "dracula")
	if form == nil {
		t.Fatal("buildWizardForm returned nil")
	}
}

func TestResolveTheme_AllNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{"charm"},
		{"dracula"},
		{"catppuccin"},
		{"base16"},
		{"default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			theme := devinit.ExportResolveTheme(tt.name)
			if theme == nil {
				t.Errorf("resolveTheme(%q) returned nil", tt.name)
			}
		})
	}
}

func TestResolveTheme_UnknownFallback(t *testing.T) {
	t.Parallel()

	theme := devinit.ExportResolveTheme("foobar")
	if theme == nil {
		t.Error("resolveTheme(\"foobar\") returned nil, expected Dracula fallback")
	}
}

func TestMapFormToAnswers_NixHardeningGuide(t *testing.T) {
	t.Parallel()

	detected := types.DetectedProject{}

	fs := devinit.NewExportFormState(
		devinit.WithQuickChoice("customize"),
		devinit.WithConfirmed(true),
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithNixHardeningGuide(true),
	)

	answers := devinit.ExportMapFormToAnswers(fs, "/tmp/project", "myproject", detected)

	if !answers.NixHardeningGuide {
		t.Error("expected NixHardeningGuide=true")
	}
}

func TestIsAccessible_TermDumb(t *testing.T) {
	origTerm := os.Getenv("TERM")
	origAccessible := os.Getenv("ACCESSIBLE")
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("ACCESSIBLE")
	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "dumb")
	t.Cleanup(func() {
		if origTerm != "" {
			os.Setenv("TERM", origTerm)
		} else {
			os.Unsetenv("TERM")
		}
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		}
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		}
	})

	if !devinit.ExportIsAccessible() {
		t.Error("isAccessible() = false, want true when TERM=dumb")
	}
}
