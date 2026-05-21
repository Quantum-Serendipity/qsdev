package types_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"gopkg.in/yaml.v3"
)

func fullWizardAnswers() types.WizardAnswers {
	return types.WizardAnswers{
		ProjectName: "my-project",
		ProjectRoot: "/home/user/projects/my-project",
		Detected: types.DetectedProject{
			HasGoMod:     true,
			GoVersion:    "1.22.5",
			IsGitRepo:    true,
			RemoteURL:    "git@github.com:org/repo.git",
			Ecosystems:   map[string]bool{"go": true, "docker": true},
			HasClaudeDir: true,
		},
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.22", PackageManager: "", Extras: []string{"delve"}},
			{Name: "typescript", Version: "22", PackageManager: "pnpm", Extras: []string{}},
		},
		Services: []types.ServiceChoice{
			{Name: "postgres", Version: "16", Settings: map[string]string{"database": "mydb"}},
			{Name: "redis", Version: "", Settings: map[string]string{}},
		},
		Direnv:          true,
		GitHooks:        []string{"pre-commit", "pre-push"},
		ExtraPackages:   []string{"jq", "ripgrep"},
		EnvVars:         map[string]string{"DATABASE_URL": "postgres://localhost/mydb", "KEY_WITH_SPECIAL": "val=ue&foo"},
		ClaudeCode:      true,
		PermissionLevel: "standard",
		Skills:          []string{"deploy", "security-review"},
		Hooks:           types.HookChoices{AutoFormat: true, SafetyBlock: true, PreCommit: false, AuditLog: false},
		MCPServers:      []string{"github"},
		QuickChoice:     "customize",
		Confirmed:       true,
	}
}

func TestWizardAnswersJSONRoundTrip(t *testing.T) {
	original := fullWizardAnswers()
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.WizardAnswers
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, got) {
		t.Errorf("JSON round-trip mismatch.\nOriginal: %+v\nGot:      %+v", original, got)
	}
}

func TestWizardAnswersYAMLRoundTrip(t *testing.T) {
	original := fullWizardAnswers()
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var got types.WizardAnswers
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	// Re-marshal both sides to JSON for comparison, which normalizes
	// nil vs empty slice/map differences from YAML round-trip.
	origJSON, _ := json.Marshal(original)
	gotJSON, _ := json.Marshal(got)
	if string(origJSON) != string(gotJSON) {
		t.Errorf("YAML round-trip mismatch.\nOriginal JSON: %s\nGot JSON:      %s", origJSON, gotJSON)
	}
}

func TestWizardAnswersZeroValueRoundTrip(t *testing.T) {
	var original types.WizardAnswers
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.WizardAnswers
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, got) {
		t.Errorf("zero-value JSON round-trip mismatch.\nOriginal: %+v\nGot:      %+v", original, got)
	}
}

func TestGeneratedStateYAMLRoundTrip(t *testing.T) {
	original := types.GeneratedState{
		LastRun:             time.Date(2026, 5, 12, 14, 30, 0, 0, time.UTC),
		TemplateVersion:     "1.0.0",
		SkillLibraryVersion: "2.1.0",
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc123", Strategy: types.Overwrite, Mode: 0o644},
			"devenv.nix":  {Hash: "sha256:def456", Strategy: types.SectionMarker, Mode: 0o644},
			".envrc":      {Hash: "sha256:789abc", Strategy: types.Append, Mode: 0o755},
		},
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var got types.GeneratedState
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if !original.LastRun.Equal(got.LastRun) {
		t.Errorf("LastRun mismatch: got %v, want %v", got.LastRun, original.LastRun)
	}
	if got.TemplateVersion != original.TemplateVersion {
		t.Errorf("TemplateVersion: got %q, want %q", got.TemplateVersion, original.TemplateVersion)
	}
	if got.SkillLibraryVersion != original.SkillLibraryVersion {
		t.Errorf("SkillLibraryVersion: got %q, want %q", got.SkillLibraryVersion, original.SkillLibraryVersion)
	}
	if len(got.Files) != len(original.Files) {
		t.Fatalf("Files count: got %d, want %d", len(got.Files), len(original.Files))
	}
	for path, origFS := range original.Files {
		gotFS, ok := got.Files[path]
		if !ok {
			t.Errorf("missing file state for %q", path)
			continue
		}
		if gotFS.Hash != origFS.Hash {
			t.Errorf("%s.Hash: got %q, want %q", path, gotFS.Hash, origFS.Hash)
		}
		if gotFS.Strategy != origFS.Strategy {
			t.Errorf("%s.Strategy: got %v, want %v", path, gotFS.Strategy, origFS.Strategy)
		}
		if gotFS.Mode != origFS.Mode {
			t.Errorf("%s.Mode: got %o, want %o", path, gotFS.Mode, origFS.Mode)
		}
	}
}

func TestGeneratedStateJSONRoundTrip(t *testing.T) {
	original := types.GeneratedState{
		LastRun:         time.Date(2026, 5, 12, 14, 30, 0, 0, time.UTC),
		TemplateVersion: "1.0.0",
		Files: map[string]types.FileState{
			"devenv.yaml": {Hash: "sha256:abc123", Strategy: types.Overwrite, Mode: 0o644},
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.GeneratedState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !original.LastRun.Equal(got.LastRun) {
		t.Errorf("LastRun mismatch")
	}
	if got.Files["devenv.yaml"].Strategy != types.Overwrite {
		t.Errorf("Strategy: got %v, want overwrite", got.Files["devenv.yaml"].Strategy)
	}
}

func TestDetectedProjectRoundTrip(t *testing.T) {
	original := types.DetectedProject{
		HasGoMod:       true,
		GoVersion:      "1.22.5",
		HasPackageJSON: true,
		NodeVersion:    "22.1.0",
		PackageManager: "pnpm",
		HasCargoToml:   false,
		HasPyProject:   true,
		PythonVersion:  "3.12",
		HasDockerfile:  true,
		Ecosystems:     map[string]bool{"go": true, "python": true, "docker": true},
		HasDevenvNix:   true,
		IsGitRepo:      true,
		RemoteURL:      "git@github.com:org/repo.git",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.DetectedProject
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, got) {
		t.Errorf("JSON round-trip mismatch.\nOriginal: %+v\nGot:      %+v", original, got)
	}
}

func TestGeneratedFileContentRoundTrip(t *testing.T) {
	original := types.GeneratedFile{
		Path:     "test.nix",
		Content:  []byte("{ pkgs, ... }: { packages = [ pkgs.git ]; }"),
		Mode:     os.FileMode(0o644),
		Strategy: types.Overwrite,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.GeneratedFile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if string(got.Content) != string(original.Content) {
		t.Errorf("Content mismatch: got %q, want %q", got.Content, original.Content)
	}
	if got.Strategy != original.Strategy {
		t.Errorf("Strategy mismatch: got %v, want %v", got.Strategy, original.Strategy)
	}
}

func TestHookChoicesRoundTrip(t *testing.T) {
	original := types.HookChoices{
		AutoFormat:  true,
		SafetyBlock: false,
		PreCommit:   true,
		AuditLog:    false,
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var got types.HookChoices
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original, got) {
		t.Errorf("YAML round-trip mismatch.\nOriginal: %+v\nGot:      %+v", original, got)
	}
}

func TestWizardAnswers_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		answers  types.WizardAnswers
		complete bool
	}{
		{
			name: "complete with languages and confirmation",
			answers: types.WizardAnswers{
				Confirmed: true,
				Languages: []types.LanguageChoice{{Name: "go"}},
			},
			complete: true,
		},
		{
			name: "complete with claude code and permission level",
			answers: types.WizardAnswers{
				Confirmed:       true,
				Languages:       []types.LanguageChoice{{Name: "go"}},
				ClaudeCode:      true,
				PermissionLevel: "standard",
			},
			complete: true,
		},
		{
			name: "incomplete without confirmation",
			answers: types.WizardAnswers{
				Confirmed: false,
				Languages: []types.LanguageChoice{{Name: "go"}},
			},
			complete: false,
		},
		{
			name: "incomplete without languages",
			answers: types.WizardAnswers{
				Confirmed: true,
				Languages: nil,
			},
			complete: false,
		},
		{
			name: "incomplete with empty languages",
			answers: types.WizardAnswers{
				Confirmed: true,
				Languages: []types.LanguageChoice{},
			},
			complete: false,
		},
		{
			name: "incomplete with claude code but no permission level",
			answers: types.WizardAnswers{
				Confirmed:       true,
				Languages:       []types.LanguageChoice{{Name: "go"}},
				ClaudeCode:      true,
				PermissionLevel: "",
			},
			complete: false,
		},
		{
			name:     "incomplete zero value",
			answers:  types.WizardAnswers{},
			complete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.answers.IsComplete()
			if got != tt.complete {
				t.Errorf("IsComplete() = %v, want %v", got, tt.complete)
			}
		})
	}
}

func TestWizardAnswers_FillDefaults(t *testing.T) {
	t.Run("fills languages from detection", func(t *testing.T) {
		a := types.WizardAnswers{}
		detected := types.DetectedProject{
			HasGoMod:       true,
			GoVersion:      "1.24",
			HasPackageJSON: true,
			NodeVersion:    "22",
			PackageManager: "pnpm",
			HasPyProject:   true,
			PythonVersion:  "3.12",
		}
		a.FillDefaults(detected)

		if len(a.Languages) != 3 {
			t.Fatalf("expected 3 languages, got %d: %+v", len(a.Languages), a.Languages)
		}
		if a.Languages[0].Name != "go" || a.Languages[0].Version != "1.24" {
			t.Errorf("expected go 1.24, got %+v", a.Languages[0])
		}
		if a.Languages[1].Name != "javascript" || a.Languages[1].PackageManager != "pnpm" {
			t.Errorf("expected javascript with pnpm, got %+v", a.Languages[1])
		}
		if a.Languages[2].Name != "python" || a.Languages[2].Version != "3.12" {
			t.Errorf("expected python 3.12, got %+v", a.Languages[2])
		}
	})

	t.Run("preserves existing languages", func(t *testing.T) {
		a := types.WizardAnswers{
			Languages: []types.LanguageChoice{{Name: "rust"}},
		}
		detected := types.DetectedProject{
			HasGoMod:  true,
			GoVersion: "1.24",
		}
		a.FillDefaults(detected)

		if len(a.Languages) != 1 {
			t.Fatalf("expected 1 language (preserved), got %d", len(a.Languages))
		}
		if a.Languages[0].Name != "rust" {
			t.Errorf("expected rust (preserved), got %s", a.Languages[0].Name)
		}
	})

	t.Run("fills default permission level for claude code", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true}
		a.FillDefaults(types.DetectedProject{})

		if a.PermissionLevel != "standard" {
			t.Errorf("expected permission level 'standard', got %q", a.PermissionLevel)
		}
	})

	t.Run("preserves existing permission level", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, PermissionLevel: "minimal"}
		a.FillDefaults(types.DetectedProject{})

		if a.PermissionLevel != "minimal" {
			t.Errorf("expected permission level 'minimal', got %q", a.PermissionLevel)
		}
	})

	t.Run("fills default hooks for claude code", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true}
		a.FillDefaults(types.DetectedProject{})

		if !a.Hooks.SafetyBlock {
			t.Error("expected SafetyBlock to be true")
		}
	})

	t.Run("preserves existing hooks", func(t *testing.T) {
		a := types.WizardAnswers{
			ClaudeCode: true,
			Hooks:      types.HookChoices{AutoFormat: true},
		}
		a.FillDefaults(types.DetectedProject{})

		if a.Hooks.SafetyBlock {
			t.Error("expected SafetyBlock to remain false when other hooks are set")
		}
		if !a.Hooks.AutoFormat {
			t.Error("expected AutoFormat to remain true")
		}
	})

	t.Run("detects all ecosystem types", func(t *testing.T) {
		a := types.WizardAnswers{}
		detected := types.DetectedProject{
			HasCargoToml:  true,
			HasPomXML:     true,
			HasCsproj:     true,
			HasDockerfile: true,
			HasTerraform:  true,
		}
		a.FillDefaults(detected)

		if len(a.Languages) != 5 {
			t.Fatalf("expected 5 languages, got %d: %+v", len(a.Languages), a.Languages)
		}
		names := make(map[string]bool)
		for _, l := range a.Languages {
			names[l.Name] = true
		}
		for _, expected := range []string{"rust", "java", "dotnet", "docker", "terraform"} {
			if !names[expected] {
				t.Errorf("missing expected language %q", expected)
			}
		}
	})

	t.Run("no default permission level when claude disabled", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: false}
		a.FillDefaults(types.DetectedProject{})

		if a.PermissionLevel != "" {
			t.Errorf("expected empty permission level when claude disabled, got %q", a.PermissionLevel)
		}
	})

	t.Run("merges Go version from detection into existing empty-version entry", func(t *testing.T) {
		a := types.WizardAnswers{
			Languages: []types.LanguageChoice{{Name: "go"}},
		}
		a.FillDefaults(types.DetectedProject{HasGoMod: true, GoVersion: "1.26.3"})

		if a.Languages[0].Version != "1.26.3" {
			t.Errorf("expected Go version 1.26.3 from detection, got %q", a.Languages[0].Version)
		}
	})

	t.Run("preserves explicit Go version over detection", func(t *testing.T) {
		a := types.WizardAnswers{
			Languages: []types.LanguageChoice{{Name: "go", Version: "1.24"}},
		}
		a.FillDefaults(types.DetectedProject{HasGoMod: true, GoVersion: "1.26.3"})

		if a.Languages[0].Version != "1.24" {
			t.Errorf("expected explicit Go version 1.24 preserved, got %q", a.Languages[0].Version)
		}
	})

	t.Run("merges JavaScript version and package manager from detection", func(t *testing.T) {
		a := types.WizardAnswers{
			Languages: []types.LanguageChoice{{Name: "javascript"}},
		}
		a.FillDefaults(types.DetectedProject{HasPackageJSON: true, NodeVersion: "22", PackageManager: "pnpm"})

		if a.Languages[0].Version != "22" {
			t.Errorf("expected JS version 22 from detection, got %q", a.Languages[0].Version)
		}
		if a.Languages[0].PackageManager != "pnpm" {
			t.Errorf("expected package manager pnpm from detection, got %q", a.Languages[0].PackageManager)
		}
	})

	t.Run("derives ComplianceLevel from Tier full", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "full"}
		a.FillDefaults(types.DetectedProject{})

		if a.ComplianceLevel != "strict" {
			t.Errorf("expected ComplianceLevel 'strict' for full tier, got %q", a.ComplianceLevel)
		}
	})

	t.Run("derives ComplianceLevel from Tier standard", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "standard"}
		a.FillDefaults(types.DetectedProject{})

		if a.ComplianceLevel != "enhanced" {
			t.Errorf("expected ComplianceLevel 'enhanced' for standard tier, got %q", a.ComplianceLevel)
		}
	})

	t.Run("preserves explicit ComplianceLevel over Tier", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "full", ComplianceLevel: "strict"}
		a.FillDefaults(types.DetectedProject{})

		if a.ComplianceLevel != "strict" {
			t.Errorf("expected explicit ComplianceLevel 'strict' preserved, got %q", a.ComplianceLevel)
		}
	})

	t.Run("derives EnabledTools from Tier full", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "full"}
		a.FillDefaults(types.DetectedProject{})

		for _, tool := range []string{"semgrep", "gitleaks", "secretspec"} {
			if !a.EnabledTools[tool] {
				t.Errorf("expected EnabledTools[%q] = true for full tier", tool)
			}
		}
	})

	t.Run("preserves existing EnabledTools over Tier", func(t *testing.T) {
		a := types.WizardAnswers{
			ClaudeCode:   true,
			Tier:         "full",
			EnabledTools: map[string]bool{"custom": true},
		}
		a.FillDefaults(types.DetectedProject{})

		if !a.EnabledTools["custom"] {
			t.Error("expected existing EnabledTools[\"custom\"] preserved")
		}
		if a.EnabledTools["semgrep"] {
			t.Error("expected tier-derived tools NOT added when EnabledTools already set")
		}
	})

	t.Run("catalog-backed agent tool defaults", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "full"}
		a.FillDefaults(types.DetectedProject{})

		cat := catalog.Default()
		defaults := cat.DefaultAgentToolConfig()

		if a.AgentTools.PostmortemEnabled != defaults.PostmortemEnabled {
			t.Errorf("PostmortemEnabled = %v, want %v", a.AgentTools.PostmortemEnabled, defaults.PostmortemEnabled)
		}
		if a.AgentTools.VersionSentinel != defaults.VersionSentinel {
			t.Errorf("VersionSentinel = %v, want %v", a.AgentTools.VersionSentinel, defaults.VersionSentinel)
		}
		if a.AgentTools.VersionSentinelHours != defaults.VersionSentinelHours {
			t.Errorf("VersionSentinelHours = %d, want %d", a.AgentTools.VersionSentinelHours, defaults.VersionSentinelHours)
		}
		if a.AgentTools.SembleEnabled != defaults.SembleEnabled {
			t.Errorf("SembleEnabled = %v, want %v", a.AgentTools.SembleEnabled, defaults.SembleEnabled)
		}
		if a.AgentTools.SembleMode != defaults.SembleMode {
			t.Errorf("SembleMode = %q, want %q", a.AgentTools.SembleMode, defaults.SembleMode)
		}
	})

	t.Run("catalog-backed MCP server defaults", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "full"}
		a.FillDefaults(types.DetectedProject{})

		want := catalog.Default().DefaultMCPServers()
		if !reflect.DeepEqual(a.MCPServers, want) {
			t.Errorf("MCPServers = %v, want %v", a.MCPServers, want)
		}
	})

	t.Run("catalog-backed tier-to-compliance derivation", func(t *testing.T) {
		tierMap := catalog.Default().TierToCompliance()
		for tier, wantLevel := range tierMap {
			if tier == "supply-chain-only" {
				continue // early return path, tested separately
			}
			a := types.WizardAnswers{ClaudeCode: true, Tier: tier}
			a.FillDefaults(types.DetectedProject{})
			if a.ComplianceLevel != wantLevel {
				t.Errorf("Tier %q: ComplianceLevel = %q, want %q", tier, a.ComplianceLevel, wantLevel)
			}
		}
	})

	t.Run("catalog-backed tier-to-enabled-tools derivation", func(t *testing.T) {
		tierTools := catalog.Default().TierToEnabledTools()
		for tier, wantTools := range tierTools {
			if tier == "supply-chain-only" {
				continue // early return path, tested separately
			}
			a := types.WizardAnswers{ClaudeCode: true, Tier: tier}
			a.FillDefaults(types.DetectedProject{})
			for _, tool := range wantTools {
				if !a.EnabledTools[tool] {
					t.Errorf("Tier %q: EnabledTools missing %q", tier, tool)
				}
			}
			if len(wantTools) > 0 && len(a.EnabledTools) != len(wantTools) {
				t.Errorf("Tier %q: EnabledTools count = %d, want %d", tier, len(a.EnabledTools), len(wantTools))
			}
		}
	})

	t.Run("supply-chain-only early return skips catalog agent defaults", func(t *testing.T) {
		a := types.WizardAnswers{ClaudeCode: true, Tier: "supply-chain-only"}
		a.FillDefaults(types.DetectedProject{})

		if len(a.MCPServers) != 0 {
			t.Errorf("supply-chain-only should skip MCP defaults, got %v", a.MCPServers)
		}
		if a.AgentTools.PostmortemEnabled {
			t.Error("supply-chain-only should skip agent tool defaults")
		}
		if a.ComplianceLevel != "" {
			t.Errorf("supply-chain-only early return should leave ComplianceLevel empty, got %q", a.ComplianceLevel)
		}
		if a.EnabledTools != nil {
			t.Errorf("supply-chain-only should leave EnabledTools nil, got %v", a.EnabledTools)
		}
	})
}

func TestGeneratedStateZeroValueRoundTrip(t *testing.T) {
	var original types.GeneratedState // zero value: nil Files map

	// YAML round-trip
	yamlData, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var gotYAML types.GeneratedState
	if err := yaml.Unmarshal(yamlData, &gotYAML); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if len(gotYAML.Files) != 0 {
		t.Errorf("YAML round-trip: Files should be nil or empty, got %v", gotYAML.Files)
	}

	// JSON round-trip
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var gotJSON types.GeneratedState
	if err := json.Unmarshal(jsonData, &gotJSON); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(gotJSON.Files) != 0 {
		t.Errorf("JSON round-trip: Files should be nil or empty, got %v", gotJSON.Files)
	}
}

func TestFileStateWithBaseContentRoundTrip(t *testing.T) {
	original := types.GeneratedState{
		TemplateVersion: "1.0.0",
		Files: map[string]types.FileState{
			"settings.json": {
				Hash:        "sha256:abc123",
				Strategy:    types.ThreeWayMerge,
				Mode:        0o644,
				BaseContent: []byte(`{"permissions": {"allow": ["Read"]}}`),
			},
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	var got types.GeneratedState
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	gotFS, ok := got.Files["settings.json"]
	if !ok {
		t.Fatal("missing file state for settings.json")
	}
	if string(gotFS.BaseContent) != string(original.Files["settings.json"].BaseContent) {
		t.Errorf("BaseContent mismatch: got %q, want %q", gotFS.BaseContent, original.Files["settings.json"].BaseContent)
	}
	if gotFS.Hash != "sha256:abc123" {
		t.Errorf("Hash mismatch: got %q, want %q", gotFS.Hash, "sha256:abc123")
	}
	if gotFS.Strategy != types.ThreeWayMerge {
		t.Errorf("Strategy mismatch: got %v, want %v", gotFS.Strategy, types.ThreeWayMerge)
	}
}

func TestServiceChoiceRoundTrip(t *testing.T) {
	original := types.ServiceChoice{
		Name:    "postgres",
		Version: "16",
		Settings: map[string]string{
			"database": "mydb",
			"port":     "5432",
			"user":     "admin",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.ServiceChoice
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name: got %q, want %q", got.Name, original.Name)
	}
	if got.Version != original.Version {
		t.Errorf("Version: got %q, want %q", got.Version, original.Version)
	}
	if len(got.Settings) != len(original.Settings) {
		t.Fatalf("Settings count: got %d, want %d", len(got.Settings), len(original.Settings))
	}
	for k, v := range original.Settings {
		if got.Settings[k] != v {
			t.Errorf("Settings[%q]: got %q, want %q", k, got.Settings[k], v)
		}
	}
}

func TestNewDetectedProject(t *testing.T) {
	dp := types.NewDetectedProject()

	if dp.Ecosystems == nil {
		t.Fatal("NewDetectedProject().Ecosystems should be non-nil")
	}

	// Writing to the map should not panic.
	dp.Ecosystems["go"] = true
	dp.Ecosystems["python"] = true

	if !dp.Ecosystems["go"] {
		t.Error("expected Ecosystems[\"go\"] to be true after assignment")
	}
	if len(dp.Ecosystems) != 2 {
		t.Errorf("expected 2 entries in Ecosystems, got %d", len(dp.Ecosystems))
	}
}

func TestEnvVarsMapWithSpecialCharacters(t *testing.T) {
	original := types.WizardAnswers{
		EnvVars: map[string]string{
			"NORMAL":       "value",
			"WITH_EQUAL":   "key=value&other=thing",
			"WITH_QUOTE":   `she said "hello"`,
			"WITH_NEWLINE": "line1\nline2",
			"EMPTY":        "",
		},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got types.WizardAnswers
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(original.EnvVars, got.EnvVars) {
		t.Errorf("EnvVars mismatch.\nOriginal: %+v\nGot:      %+v", original.EnvVars, got.EnvVars)
	}
}
