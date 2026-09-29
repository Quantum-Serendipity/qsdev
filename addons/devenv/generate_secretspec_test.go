package devenv_test

import (
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// secretSpecFile mirrors secretspec's configuration schema
// ([project] + [profiles.<name>] inline secret tables).
type secretSpecFile struct {
	Project struct {
		Name     string `toml:"name"`
		Revision string `toml:"revision"`
	} `toml:"project"`
	Profiles map[string]map[string]secretSpecSecret `toml:"profiles"`
}

type secretSpecSecret struct {
	Description string `toml:"description"`
	Required    *bool  `toml:"required"`
	Type        string `toml:"type"`
	Generate    any    `toml:"generate"`
}

// parseSecretSpec decodes content against the secretspec schema and fails on
// any key the schema does not define (e.g. the old invented [providers] or
// [secrets.*] tables).
func parseSecretSpec(t *testing.T, content []byte) secretSpecFile {
	t.Helper()
	var f secretSpecFile
	md, err := toml.Decode(string(content), &f)
	if err != nil {
		t.Fatalf("secretspec.toml is not valid TOML: %v\n%s", err, content)
	}
	for _, key := range md.Undecoded() {
		// generate's options table decodes into `any`, which BurntSushi
		// reports as undecoded; everything else must be in the schema.
		if len(key) == 5 && key[3] == "generate" {
			continue
		}
		t.Fatalf("secretspec.toml has key %v outside the secretspec schema\n%s", key, content)
	}
	if f.Project.Name == "" || f.Project.Revision != "1.0" {
		t.Fatalf("[project] = %+v, want a name and revision \"1.0\"", f.Project)
	}
	return f
}

func generateSecretSpec(t *testing.T, answers types.WizardAnswers, reg *ecosystem.Registry) secretSpecFile {
	t.Helper()
	gf, err := devenv.GenerateSecretSpecToml(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}
	return parseSecretSpec(t, gf.Content)
}

func TestGenerateSecretSpecToml_WithServices(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{
		ProjectName: "my-app",
		Services:    []types.ServiceChoice{{Name: "postgres"}, {Name: "redis"}},
	}

	gf, err := devenv.GenerateSecretSpecToml(answers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}
	f := parseSecretSpec(t, gf.Content)
	if f.Project.Name != "my-app" {
		t.Errorf("project name = %q, want my-app", f.Project.Name)
	}

	secrets := f.Profiles["default"]
	for _, name := range []string{"DATABASE_URL", "POSTGRES_PASSWORD", "REDIS_URL"} {
		s, ok := secrets[name]
		if !ok {
			t.Errorf("missing %s in [profiles.default]", name)
			continue
		}
		if s.Description == "" {
			t.Errorf("%s has no description (required by secretspec)", name)
		}
	}

	pw := secrets["POSTGRES_PASSWORD"]
	if pw.Type != "password" {
		t.Errorf("POSTGRES_PASSWORD type = %q, want password (type is required with generate)", pw.Type)
	}
	gen, ok := pw.Generate.(map[string]any)
	if !ok || gen["length"] != int64(32) || gen["charset"] != "alphanumeric" {
		t.Errorf("POSTGRES_PASSWORD generate = %#v, want { length = 32, charset = \"alphanumeric\" }", pw.Generate)
	}
	if url := secrets["DATABASE_URL"]; url.Generate != nil || url.Type != "" {
		t.Errorf("DATABASE_URL must not be auto-generated: %+v", url)
	}

	if gf.Path != "secretspec.toml" {
		t.Errorf("expected path secretspec.toml, got %q", gf.Path)
	}
	if gf.Mode != 0o644 {
		t.Errorf("expected mode 0o644, got %04o", gf.Mode)
	}
	if gf.Strategy != types.Overwrite {
		t.Errorf("expected strategy Overwrite, got %v", gf.Strategy)
	}
	if gf.Owner != "secretspec" {
		t.Errorf("expected owner secretspec, got %q", gf.Owner)
	}
}

func TestGenerateSecretSpecToml_GenerationTypes(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&mockSecretDeclarerModule{
		MockModule: ecosystem.MockModule{NameVal: "gen", DisplayNameVal: "Gen", TierVal: 2},
		secrets: []ecosystem.SecretDecl{
			{Name: "HEX_KEY", Description: "hex", AutoGenerate: true, GenerateSpec: &ecosystem.GenerateSpec{Length: 64, Charset: "hex"}},
			{Name: "B64_KEY", Description: "b64", AutoGenerate: true, GenerateSpec: &ecosystem.GenerateSpec{Length: 44, Charset: "base64"}},
			{Name: "ID", Description: "uuid", AutoGenerate: true, GenerateSpec: &ecosystem.GenerateSpec{Charset: "uuid"}},
			{Name: "PLAIN", Description: "default", AutoGenerate: true},
		},
	})
	f := generateSecretSpec(t, types.WizardAnswers{ProjectRoot: "/tmp/proj", Languages: []types.LanguageChoice{{Name: "gen"}}}, reg)
	if f.Project.Name != "proj" {
		t.Errorf("project name = %q, want directory name proj", f.Project.Name)
	}

	tests := []struct {
		name     string
		wantType string
		wantGen  any
	}{
		{"HEX_KEY", "hex", map[string]any{"bytes": int64(32)}},
		{"B64_KEY", "base64", map[string]any{"bytes": int64(33)}},
		{"ID", "uuid", true},
		{"PLAIN", "password", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := f.Profiles["default"][tt.name]
			if s.Type != tt.wantType {
				t.Errorf("type = %q, want %q", s.Type, tt.wantType)
			}
			switch want := tt.wantGen.(type) {
			case bool:
				if s.Generate != want {
					t.Errorf("generate = %#v, want %v", s.Generate, want)
				}
			case map[string]any:
				got, ok := s.Generate.(map[string]any)
				if !ok || got["bytes"] != want["bytes"] {
					t.Errorf("generate = %#v, want %#v", s.Generate, want)
				}
			}
		})
	}
}

func TestGenerateSecretSpecToml_EscapesProjectName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		projectName string
	}{
		{"quote and newline injection", "evil\"\n[providers]\nx = \"y"},
		{"backslash and tab", "a\\b\tc"},
		{"other control characters", "a\u0001b\u001fc\u007fd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				ProjectName: tt.projectName,
				Services:    []types.ServiceChoice{{Name: "redis"}},
			}
			f := generateSecretSpec(t, answers, nil)
			if f.Project.Name != answers.ProjectName {
				t.Errorf("project name = %q, want %q", f.Project.Name, answers.ProjectName)
			}
		})
	}
}

func TestGenerateSecretSpecToml_NoDeclarations(t *testing.T) {
	t.Parallel()
	gf, err := devenv.GenerateSecretSpecToml(types.WizardAnswers{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf != nil {
		t.Errorf("expected nil when no declarations, got %+v", gf)
	}
}

func TestGenerateSecretSpecToml_Dedup(t *testing.T) {
	t.Parallel()
	// Duplicate declarations must not produce duplicate TOML keys, which
	// would make the file unparseable.
	answers := types.WizardAnswers{
		ProjectName: "p",
		Services:    []types.ServiceChoice{{Name: "postgres"}, {Name: "postgres"}},
	}
	f := generateSecretSpec(t, answers, nil)
	if len(f.Profiles["default"]) != 2 {
		t.Errorf("secrets = %v, want DATABASE_URL and POSTGRES_PASSWORD once each", f.Profiles["default"])
	}
}

func TestGenerateSecretSpecToml_ServicesAndModulesCombined(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&mockSecretDeclarerModule{
		MockModule: ecosystem.MockModule{NameVal: "container", DisplayNameVal: "Containers", TierVal: 1},
		secrets: []ecosystem.SecretDecl{
			{Name: "DOCKER_REGISTRY_TOKEN", Description: "Docker registry token", Required: true, Source: "container"},
		},
	})
	answers := types.WizardAnswers{
		ProjectName: "p",
		Services:    []types.ServiceChoice{{Name: "redis"}},
		Languages:   []types.LanguageChoice{{Name: "container"}},
	}
	f := generateSecretSpec(t, answers, reg)
	secrets := f.Profiles["default"]
	if _, ok := secrets["REDIS_URL"]; !ok {
		t.Error("expected REDIS_URL from service")
	}
	tok, ok := secrets["DOCKER_REGISTRY_TOKEN"]
	if !ok {
		t.Fatal("expected DOCKER_REGISTRY_TOKEN from module")
	}
	if tok.Required == nil || !*tok.Required {
		t.Errorf("DOCKER_REGISTRY_TOKEN required = %v, want true", tok.Required)
	}
}

func TestGenerateSecretSpecToml_UnknownServiceReturnsNil(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{{Name: "mongodb"}}, // no secrets declared for mongodb
	}
	gf, err := devenv.GenerateSecretSpecToml(answers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf != nil {
		t.Errorf("expected nil for service with no secrets, got %+v", gf)
	}
}

// mockSecretDeclarerModule embeds MockModule and adds SecretDeclarer.
type mockSecretDeclarerModule struct {
	ecosystem.MockModule
	secrets []ecosystem.SecretDecl
}

func (m *mockSecretDeclarerModule) SecretDeclarations(_ ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	return m.secrets
}
