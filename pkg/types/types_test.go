package types_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
	"gopkg.in/yaml.v3"
)

func fullWizardAnswers() types.WizardAnswers {
	return types.WizardAnswers{
		ProjectName: "my-project",
		ProjectRoot: "/home/user/projects/my-project",
		Detected: types.DetectedProject{
			HasGoMod:    true,
			GoVersion:   "1.22.5",
			IsGitRepo:   true,
			RemoteURL:   "git@github.com:org/repo.git",
			Ecosystems:  map[string]bool{"go": true, "docker": true},
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
			"devenv.yaml": {Hash: "sha256:abc123", Strategy: types.Overwrite, Mode: 0644},
			"devenv.nix":  {Hash: "sha256:def456", Strategy: types.SectionMarker, Mode: 0644},
			".envrc":      {Hash: "sha256:789abc", Strategy: types.Append, Mode: 0755},
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
			"devenv.yaml": {Hash: "sha256:abc123", Strategy: types.Overwrite, Mode: 0644},
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
		Mode:     os.FileMode(0644),
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
}

func TestEnvVarsMapWithSpecialCharacters(t *testing.T) {
	original := types.WizardAnswers{
		EnvVars: map[string]string{
			"NORMAL":    "value",
			"WITH_EQUAL": "key=value&other=thing",
			"WITH_QUOTE": `she said "hello"`,
			"WITH_NEWLINE": "line1\nline2",
			"EMPTY":     "",
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
