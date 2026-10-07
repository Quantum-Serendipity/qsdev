package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateLocalConfigTemplate_GoProject(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "go") {
		t.Error("Go project template should mention go")
	}
	if !strings.Contains(s, "1.24") {
		t.Error("Go project template should include go version")
	}
}

func TestGenerateLocalConfigTemplate_MultiLanguage(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "javascript", Version: "22"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "go") {
		t.Error("Multi-lang template should mention go")
	}
	if !strings.Contains(s, "javascript") {
		t.Error("Multi-lang template should mention javascript")
	}
}

func TestGenerateLocalConfigTemplate_ClaudeEnabled(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "claude_code:") {
		t.Error("Claude-enabled template should contain claude_code section")
	}
	if !strings.Contains(s, "permission_level") {
		t.Error("Claude-enabled template should contain permission_level")
	}
}

func TestGenerateLocalConfigTemplate_ClaudeDisabled(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
		ClaudeCode: false,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if strings.Contains(s, "claude_code:") {
		t.Error("Claude-disabled template should not contain claude_code section")
	}
}

func TestGenerateLocalConfigTemplate_AllLinesCommented(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "python", Version: "3.12"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		if line == "" {
			continue // empty lines are fine
		}
		if !strings.HasPrefix(line, "#") {
			t.Errorf("line %d is not commented: %q", i+1, line)
		}
	}
}

func TestGenerateLocalConfigTemplate_DefaultVersions(t *testing.T) {
	// Test that languages without explicit versions get default examples.
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "python"}, // no version set
		},
		ClaudeCode: false,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "python") {
		t.Error("template should mention python")
	}
	if !strings.Contains(s, "3.12") {
		t.Error("template should include default python version example")
	}
}

// The generated template must parse under the strict local-config decoder
// both as written and with its example settings uncommented, so a developer
// who enables an example never hits an unknown-key error.
func TestGenerateLocalConfigTemplate_ParsesStrictly(t *testing.T) {
	t.Parallel()
	content := string(GenerateLocalConfigTemplate(types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go", Version: "1.24"}},
		ClaudeCode: true,
	}, types.DetectedProject{}))

	start := strings.Index(content, "# extra_packages:")
	if start < 0 {
		t.Fatal("template has no extra_packages example")
	}
	var uncommented strings.Builder
	for line := range strings.Lines(content[start:]) {
		uncommented.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "#"), " "))
	}

	tests := []struct {
		name    string
		content string
	}{
		{"as written", content},
		{"examples uncommented", uncommented.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), ".qsdev.local.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := qsdevconfig.ParseLocalConfig(path); err != nil {
				t.Fatalf("template does not parse strictly: %v\n%s", err, tt.content)
			}
		})
	}
}

func TestWriteLocalConfigTemplate(t *testing.T) {
	t.Parallel()

	localCfg := branding.Get().LocalConfig
	answers := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go"}}}
	template := string(GenerateLocalConfigTemplate(answers, answers.Detected))
	const mine = "# mine\n"
	announce := "+ " + localCfg + " (local, untracked)\n"

	writeMine := func(t *testing.T, path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(mine), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// linkTo points the local config at target, skipping where symlinks
	// need a privilege the test lacks (Windows by default).
	linkTo := func(t *testing.T, dir, target string) {
		t.Helper()
		if err := os.Symlink(target, filepath.Join(dir, localCfg)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}

	tests := []struct {
		name string
		// setup prepares dir and returns the file a symlink points at, if any.
		setup      func(t *testing.T, dir string) string
		dryRun     bool
		wantErr    bool
		wantOut    string
		wantOnDisk string // "" means the file must not exist
	}{
		{name: "absent is created and announced", wantOut: announce, wantOnDisk: template},
		{
			name:       "existing is kept",
			setup:      func(t *testing.T, dir string) string { writeMine(t, filepath.Join(dir, localCfg)); return "" },
			wantOnDisk: mine,
		},
		{
			name: "symlink to existing file in root is kept",
			setup: func(t *testing.T, dir string) string {
				target := filepath.Join(dir, "my-local.yaml")
				writeMine(t, target)
				linkTo(t, dir, "my-local.yaml")
				return target
			},
			wantOnDisk: mine,
		},
		{
			name: "symlink to existing file outside root is kept",
			setup: func(t *testing.T, dir string) string {
				target := filepath.Join(t.TempDir(), "dot-local.yaml")
				writeMine(t, target)
				linkTo(t, dir, target)
				return target
			},
			wantOnDisk: mine,
		},
		{
			name: "dangling symlink refused",
			setup: func(t *testing.T, dir string) string {
				linkTo(t, dir, filepath.Join(t.TempDir(), "elsewhere"))
				return ""
			},
			wantErr: true,
		},
		{name: "dry run previews only", dryRun: true, wantOut: announce},
		{
			name:       "dry run with existing is silent",
			setup:      func(t *testing.T, dir string) string { writeMine(t, filepath.Join(dir, localCfg)); return "" },
			dryRun:     true,
			wantOnDisk: mine,
		},
		{
			name: "dry run with symlink to existing file is silent",
			setup: func(t *testing.T, dir string) string {
				target := filepath.Join(t.TempDir(), "dot-local.yaml")
				writeMine(t, target)
				linkTo(t, dir, target)
				return target
			},
			dryRun:     true,
			wantOnDisk: mine,
		},
		{
			name: "dry run with dangling symlink fails like the real run",
			setup: func(t *testing.T, dir string) string {
				linkTo(t, dir, filepath.Join(t.TempDir(), "elsewhere"))
				return ""
			},
			dryRun:  true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			var target string
			if tt.setup != nil {
				target = tt.setup(t, dir)
			}
			var out bytes.Buffer
			err := writeLocalConfigTemplate(dir, answers, tt.dryRun, &out)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if out.String() != tt.wantOut {
				t.Errorf("output = %q, want %q", out.String(), tt.wantOut)
			}
			if tt.wantErr {
				return
			}
			data, err := os.ReadFile(filepath.Join(dir, localCfg))
			switch {
			case tt.wantOnDisk == "" && !os.IsNotExist(err):
				t.Errorf("%s exists after a dry run (err %v)", localCfg, err)
			case tt.wantOnDisk != "" && string(data) != tt.wantOnDisk:
				t.Errorf("%s = %q, want %q (err %v)", localCfg, data, tt.wantOnDisk, err)
			}
			if target != "" {
				assertSymlinkKept(t, filepath.Join(dir, localCfg), target, mine)
			}
		})
	}
}

// TestWriteLocalConfigTemplate_DryRunReportsLstatError verifies a dry run
// surfaces an error other than "absent" (here ENOTDIR) instead of reporting
// the file as present, since the real run would fail with it.
func TestWriteLocalConfigTemplate_DryRunReportsLstatError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows reports a file used as a directory as not-exist")
	}
	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := writeLocalConfigTemplate(notDir, types.WizardAnswers{}, true, &out)
	if err == nil {
		t.Fatalf("dry run under a regular file succeeded with output %q", out.String())
	}
}

// assertSymlinkKept fails unless link is still a symlink that resolves to
// target and target still holds want.
func assertSymlinkKept(t *testing.T, link, target, want string) {
	t.Helper()
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is no longer a symlink (mode %v, err %v)", link, info, err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != want {
		t.Errorf("symlink target %s = %q (err %v), want %q", target, got, err, want)
	}
}
