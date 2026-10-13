package devenv

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// useMalformedProjectOverlay points the default catalog at a project whose
// .qsdev/defaults.yaml does not parse, and restores the previous project root
// and both cached defaults when the test ends. It changes process-wide state,
// so its callers must not run in parallel.
func useMalformedProjectOverlay(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".qsdev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "defaults.yaml"), []byte("a: b: c\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	prior := catalog.ProjectRoot()
	t.Cleanup(func() {
		catalog.ResetDefault()
		if err := catalog.SetProjectRoot(prior); err != nil {
			t.Errorf("restoring the catalog project root: %v", err)
		}
		toolreg.ResetDefaultRegistry()
	})
	catalog.ResetDefault()
	if err := catalog.SetProjectRoot(root); err != nil {
		t.Fatal(err)
	}
	toolreg.ResetDefaultRegistry()
}

// TestGenerate_MalformedOverlayReturnsError checks that every generation
// entry point fails closed, with an error naming the file, when the project
// catalog overlay cannot be loaded, rather than panicking or silently
// generating without the catalog's security defaults.
func TestGenerate_MalformedOverlayReturnsError(t *testing.T) {
	// The registry is built before the overlay is planted: only generation
	// itself is under test.
	registry := ecosystem.DefaultRegistry()
	useMalformedProjectOverlay(t)

	goAnswers := types.WizardAnswers{
		ProjectName: "overlay-test",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}
	tests := []struct {
		name string
		run  func() error
	}{
		{"Generate", func() error {
			_, err := NewDevenvGenerator(registry).Generate(goAnswers)
			return err
		}},
		{"GenerateDevenvNix", func() error {
			_, err := GenerateDevenvNix(goAnswers, registry)
			return err
		}},
		{"GenerateDevenvYaml", func() error {
			_, err := GenerateDevenvYaml(goAnswers, registry)
			return err
		}},
		{"GenerateSecretSpecToml", func() error {
			answers := goAnswers
			answers.Services = []types.ServiceChoice{{Name: "postgres"}}
			_, err := GenerateSecretSpecToml(answers, registry)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panicked: %v", r)
						t.Errorf("%s panicked: %v", tt.name, r)
					}
				}()
				return tt.run()
			}()
			if err == nil {
				t.Fatalf("%s succeeded with a malformed .qsdev/defaults.yaml; want an error", tt.name)
			}
			if !strings.Contains(err.Error(), "defaults.yaml") {
				t.Errorf("%s error = %q; want it to name defaults.yaml", tt.name, err)
			}
		})
	}
}

// recordingModule is an ecosystem module that records the ModuleConfig every
// generation consumer passes it, keyed by the consumer.
type recordingModule struct {
	*ecosystem.MockModule
	mu   sync.Mutex
	seen map[string][]ecosystem.ModuleConfig
}

func newRecordingModule(name string) *recordingModule {
	return &recordingModule{
		MockModule: &ecosystem.MockModule{NameVal: name, DisplayNameVal: name},
		seen:       make(map[string][]ecosystem.ModuleConfig),
	}
}

func (m *recordingModule) record(consumer string, cfg ecosystem.ModuleConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seen[consumer] = append(m.seen[consumer], cfg)
}

func (m *recordingModule) DevenvNixFragment(cfg ecosystem.ModuleConfig) (string, error) {
	m.record("DevenvNixFragment", cfg)
	return "", nil
}

func (m *recordingModule) SecurityConfigs(cfg ecosystem.ModuleConfig) []types.GeneratedFile {
	m.record("SecurityConfigs", cfg)
	return nil
}

func (m *recordingModule) PreCommitHooks(cfg ecosystem.ModuleConfig) []ecosystem.HookConfig {
	m.record("PreCommitHooks", cfg)
	return nil
}

func (m *recordingModule) CICommands(cfg ecosystem.ModuleConfig) []ecosystem.CICommand {
	m.record("CICommands", cfg)
	return nil
}

func (m *recordingModule) VerificationCommands(cfg ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	m.record("VerificationCommands", cfg)
	return ecosystem.VerificationCommands{}
}

func (m *recordingModule) DevenvYamlInputs(cfg ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	m.record("DevenvYamlInputs", cfg)
	return nil
}

func (m *recordingModule) SecretDeclarations(cfg ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	m.record("SecretDeclarations", cfg)
	return nil
}

func (m *recordingModule) DevenvPackages(cfg ecosystem.ModuleConfig) []string {
	m.record("DevenvPackages", cfg)
	return nil
}

func (m *recordingModule) DevenvPackageExprs(cfg ecosystem.ModuleConfig) []string {
	m.record("DevenvPackageExprs", cfg)
	return nil
}

var (
	_ ecosystem.DevenvYamlInputProvider = (*recordingModule)(nil)
	_ ecosystem.SecretDeclarer          = (*recordingModule)(nil)
	_ ecosystem.PackageProvider         = (*recordingModule)(nil)
	_ ecosystem.PackageExprProvider     = (*recordingModule)(nil)
)

// TestGenContext_ModuleConfigConsistent is the U12-12 regression test: every
// generated file is produced from one ModuleConfig per language, completed
// from detection, so devenv.nix, devenv.yaml, security configs, tasks,
// secretspec and CI never disagree about a module's package manager.
func TestGenContext_ModuleConfigConsistent(t *testing.T) {
	t.Parallel()

	t.Run("recording-module", func(t *testing.T) {
		t.Parallel()
		const level = "enhanced"
		mod := newRecordingModule("recorder")
		registry := ecosystem.NewRegistry()
		if err := registry.Register(mod); err != nil {
			t.Fatal(err)
		}

		answers := types.WizardAnswers{
			ProjectName:     "consistent",
			Tier:            "standard",
			ComplianceLevel: level,
			Languages:       []types.LanguageChoice{{Name: "recorder"}},
			Infrastructure:  types.InfraConfig{BuildCache: "sccache"},
			Detected: types.DetectedProject{Suggested: map[string]types.LanguageChoice{
				"recorder": {Name: "recorder", PackageManager: "pmx", Extras: []string{"k=v"}},
			}},
		}
		gen := NewDevenvGenerator(registry, WithProfileRegistry(profile.DefaultProfileRegistry()))
		if _, err := gen.Generate(answers); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if _, err := GenerateSecretSpecToml(answers, registry); err != nil {
			t.Fatalf("GenerateSecretSpecToml() error = %v", err)
		}

		cat, err := catalog.Default()
		if err != nil {
			t.Fatal(err)
		}
		want := ecosystem.ModuleConfig{
			PackageManager: "pmx",
			Extras:         map[string]string{"k": "v", ecosystem.ExtraBuildCache: "sccache"},
			MinReleaseAge:  cat.AgeGate(level),
		}
		if want.MinReleaseAge == 0 {
			t.Fatalf("catalog AgeGate(%q) = 0; the test needs a non-zero window", level)
		}
		// PreCommitHooks is left out: securityHookOverrides intentionally
		// asks every registered module, selected or not, with ModuleConfig{}.
		consumers := []string{
			"DevenvNixFragment", "SecurityConfigs", "CICommands",
			"VerificationCommands", "DevenvYamlInputs", "SecretDeclarations",
			"DevenvPackages", "DevenvPackageExprs",
		}
		mod.mu.Lock()
		defer mod.mu.Unlock()
		for _, consumer := range consumers {
			got := mod.seen[consumer]
			if len(got) == 0 {
				t.Errorf("%s was never called", consumer)
				continue
			}
			for _, cfg := range got {
				if !reflect.DeepEqual(cfg, want) {
					t.Errorf("%s received %+v; want %+v", consumer, cfg, want)
				}
			}
		}
	})

	t.Run("python-uv-from-detection", func(t *testing.T) {
		t.Parallel()
		answers := types.WizardAnswers{
			ProjectName: "uvproj",
			Tier:        "standard",
			Languages:   []types.LanguageChoice{{Name: "python"}},
			Detected: types.DetectedProject{Suggested: map[string]types.LanguageChoice{
				"python": {Name: "python", PackageManager: "uv"},
			}},
		}
		gen := NewDevenvGenerator(ecosystem.DefaultRegistry(), WithProfileRegistry(profile.DefaultProfileRegistry()))
		files, err := gen.Generate(answers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		byPath := make(map[string]string, len(files))
		for _, f := range files {
			byPath[f.Path] = string(f.Content)
		}
		if nix := byPath["devenv.nix"]; !strings.Contains(nix, "uv.enable = true") {
			t.Errorf("devenv.nix does not enable uv, which CI uses:\n%s", nix)
		}
		if wf := byPath[".github/workflows/security-scan.yml"]; !strings.Contains(wf, "uv sync --locked") {
			t.Errorf("security-scan workflow does not run uv sync --locked:\n%s", wf)
		}
	})
}

// TestGenContext_UnknownLanguageErrors checks that every generator that takes
// a module registry rejects a language no module implements, as Generate
// does, instead of silently generating without it.
func TestGenContext_UnknownLanguageErrors(t *testing.T) {
	t.Parallel()
	registry := ecosystem.DefaultRegistry()
	answers := types.WizardAnswers{
		ProjectName: "unknown-lang",
		Languages:   []types.LanguageChoice{{Name: "no-such-language"}},
		Services:    []types.ServiceChoice{{Name: "postgres"}},
	}
	tests := []struct {
		name string
		run  func() error
	}{
		{"GenerateDevenvYaml", func() error { _, err := GenerateDevenvYaml(answers, registry); return err }},
		{"GenerateSecretSpecToml", func() error { _, err := GenerateSecretSpecToml(answers, registry); return err }},
		{"GenerateDevenvNix", func() error { _, err := GenerateDevenvNix(answers, registry); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), "unknown language module") {
				t.Errorf("%s error = %v; want unknown language module", tt.name, err)
			}
		})
	}
}
