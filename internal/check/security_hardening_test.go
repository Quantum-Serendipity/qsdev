package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	// Register the ecosystem modules whose generated security configs the
	// hardening check derives its expected settings from.
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/helm"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/terraform"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckSecurityHardening_LockFilePresent(t *testing.T) {
	dir := t.TempDir()

	// Create go.sum.
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("hash"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		QsdevConfig: &types.QsdevConfig{
			Languages: []types.LanguageConfig{
				{Name: "go"},
			},
		},
	}

	results := CheckSecurityHardening(ctx)

	hasPass := false
	for _, r := range results {
		if r.Name == "lockfile_go" && r.Status == StatusPass {
			hasPass = true
			break
		}
	}
	if !hasPass {
		t.Error("expected lockfile_go to pass when go.sum exists")
	}
}

func TestCheckSecurityHardening_LockFileMissing(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot: dir,
		QsdevConfig: &types.QsdevConfig{
			Languages: []types.LanguageConfig{
				{Name: "go"},
			},
		},
	}

	results := CheckSecurityHardening(ctx)

	hasFail := false
	for _, r := range results {
		if r.Name == "lockfile_go" && r.Status == StatusFail {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected lockfile_go to fail when go.sum is missing")
	}
}

// hardenedNpmrc mirrors the settings the javascript module generates.
const hardenedNpmrc = "save-exact=true\nignore-scripts=true\nmin-release-age=3\naudit=true\naudit-level=moderate\n"

// findResult returns the result with the given name, or nil.
func findResult(results []CheckResult, name string) *CheckResult {
	for i := range results {
		if results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}

func TestCheckSecurityHardening_SecurityConfigSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lang       types.LanguageConfig
		files      map[string]string
		wantStatus CheckStatus // "" means no security_config result expected
	}{
		{
			name:       "npmrc with generated settings",
			lang:       types.LanguageConfig{Name: "javascript"},
			files:      map[string]string{".npmrc": hardenedNpmrc},
			wantStatus: StatusPass,
		},
		{
			name:       "npmrc with stricter release age and spacing",
			lang:       types.LanguageConfig{Name: "javascript"},
			files:      map[string]string{".npmrc": "# custom\nsave-exact = true\nignore-scripts = true\nmin-release-age = 7\naudit=true\naudit-level=moderate\nregistry=https://example.test/\n"},
			wantStatus: StatusPass,
		},
		{
			name:       "npmrc re-enables install scripts",
			lang:       types.LanguageConfig{Name: "javascript"},
			files:      map[string]string{".npmrc": strings.Replace(hardenedNpmrc, "ignore-scripts=true", "ignore-scripts=false", 1)},
			wantStatus: StatusFail,
		},
		{
			name:       "npmrc present but empty of hardening",
			lang:       types.LanguageConfig{Name: "javascript"},
			files:      map[string]string{".npmrc": "package-lock=true\n"},
			wantStatus: StatusFail,
		},
		{
			name:       "npmrc missing",
			lang:       types.LanguageConfig{Name: "javascript"},
			wantStatus: StatusFail,
		},
		{
			name:       "pnpm workspace with hardening",
			lang:       types.LanguageConfig{Name: "javascript", PackageManager: "pnpm"},
			files:      map[string]string{"pnpm-workspace.yaml": "strictDepBuilds: true # comment\nminimumReleaseAge: 10080\ntrustPolicy: no-downgrade\nblockExoticSubdeps: true\npackages:\n  - app\n"},
			wantStatus: StatusPass,
		},
		{
			name:       "pip.conf with hardening",
			lang:       types.LanguageConfig{Name: "python"},
			files:      map[string]string{"pip.conf": "[global]\nrequire-hashes = true\nonly-binary = :all:\n"},
			wantStatus: StatusPass,
		},
		{
			name:       "pyproject.toml alone is not hardening",
			lang:       types.LanguageConfig{Name: "python"},
			files:      map[string]string{"pyproject.toml": "[project]\n"},
			wantStatus: StatusFail,
		},
		{
			name: "uv projects have no generated pip config",
			lang: types.LanguageConfig{Name: "python", PackageManager: "uv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			results := CheckSecurityHardening(CheckContext{
				ProjectRoot: dir,
				QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{tt.lang}},
			})

			got := findResult(results, "security_config_"+tt.lang.Name)
			if tt.wantStatus == "" {
				if got != nil {
					t.Fatalf("unexpected result: %+v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected security_config_%s result, got %+v", tt.lang.Name, results)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s (%s)", got.Status, tt.wantStatus, got.Message)
			}
		})
	}
}

// TestCheckSecurityHardening_ManifestIsNotALockFile verifies that a file that
// is both a manifest and a listed lock file never passes the lock-file check
// on its own.
func TestCheckSecurityHardening_ManifestIsNotALockFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang       string
		files      []string
		wantStatus CheckStatus
	}{
		{lang: "java", files: []string{"pom.xml"}, wantStatus: StatusWarn},
		{lang: "java", files: []string{"pom.xml", "gradle.lockfile"}, wantStatus: StatusPass},
		{lang: "python", files: []string{"requirements.txt"}, wantStatus: StatusWarn},
		{lang: "python", files: []string{"requirements.txt", "uv.lock"}, wantStatus: StatusPass},
		{lang: "cpp", files: []string{"vcpkg.json"}, wantStatus: StatusWarn},
		{lang: "java", wantStatus: StatusFail},
	}

	for _, tt := range tests {
		t.Run(tt.lang+"/"+strings.Join(tt.files, "+"), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			results := CheckSecurityHardening(CheckContext{
				ProjectRoot: dir,
				QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{{Name: tt.lang}}},
			})
			got := findResult(results, "lockfile_"+tt.lang)
			if got == nil {
				t.Fatalf("expected lockfile_%s result", tt.lang)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s (%s)", got.Status, tt.wantStatus, got.Message)
			}
		})
	}
}

func TestSettingSatisfied(t *testing.T) {
	t.Parallel()

	tests := []struct {
		want, got string
		ok        bool
	}{
		{"true", "true", true},
		{"true", "TRUE", true},
		{"true", "false", false},
		{"3", "7", true},
		{"3", "1", false},
		{"7d", "14d", true},
		{"7d", "3d", false},
		{"7d", "14h", false},
		{"moderate", "high", false},
	}
	for _, tt := range tests {
		t.Run(tt.want+"_"+tt.got, func(t *testing.T) {
			t.Parallel()
			if got := settingSatisfied(tt.want, tt.got); got != tt.ok {
				t.Errorf("settingSatisfied(%q, %q) = %v, want %v", tt.want, tt.got, got, tt.ok)
			}
		})
	}
}

func TestCheckSecurityHardening_NoConfig(t *testing.T) {
	ctx := CheckContext{}

	results := CheckSecurityHardening(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckSecurityHardening_NoLanguages(t *testing.T) {
	ctx := CheckContext{
		QsdevConfig: &types.QsdevConfig{},
	}

	results := CheckSecurityHardening(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckSecurityHardening_MultipleLanguages(t *testing.T) {
	dir := t.TempDir()

	// Create go.sum but not package-lock.json.
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("hash"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		QsdevConfig: &types.QsdevConfig{
			Languages: []types.LanguageConfig{
				{Name: "go"},
				{Name: "javascript"},
			},
		},
	}

	results := CheckSecurityHardening(ctx)

	goPass := false
	jsFail := false
	for _, r := range results {
		if r.Name == "lockfile_go" && r.Status == StatusPass {
			goPass = true
		}
		if r.Name == "lockfile_javascript" && r.Status == StatusFail {
			jsFail = true
		}
	}

	if !goPass {
		t.Error("expected lockfile_go to pass")
	}
	if !jsFail {
		t.Error("expected lockfile_javascript to fail")
	}
}

// TestCheckSecurityHardening_LockFileRequirement covers lock file enforcement
// that depends on the ecosystem module: manifests without dependencies need
// no lock file (W163), ecosystems outside the shared lockfile catalog use the
// lock files their module declares (W124), and npm's publishable lockfile is
// accepted (W062).
func TestCheckSecurityHardening_LockFileRequirement(t *testing.T) {
	t.Parallel()

	const goModNoDeps = "module example.com/m\n\ngo 1.24\n"
	const goModWithDeps = goModNoDeps + "\nrequire golang.org/x/mod v0.40.0\n"
	const chartWithDeps = "apiVersion: v2\nname: app\nversion: 0.1.0\ndependencies:\n  - name: redis\n    version: 18.0.0\n    repository: https://charts.example.test\n"
	const chartNoDeps = "apiVersion: v2\nname: app\nversion: 0.1.0\n"

	tests := []struct {
		name       string
		lang       string
		files      map[string]string
		resultName string
		wantStatus CheckStatus
	}{
		{"go without requires needs no go.sum", "go", map[string]string{"go.mod": goModNoDeps}, "lockfile_go", StatusSkip},
		{"go with requires needs go.sum", "go", map[string]string{"go.mod": goModWithDeps}, "lockfile_go", StatusFail},
		{"go with requires and go.sum", "go", map[string]string{"go.mod": goModWithDeps, "go.sum": "h"}, "lockfile_go", StatusPass},
		{"unreadable go.mod still enforces go.sum", "go", nil, "lockfile_go", StatusFail},
		{"terraform without provider lock fails", "terraform", map[string]string{"main.tf": "provider \"google\" {}\n"}, "lockfile_terraform", StatusFail},
		{"terraform with provider lock passes", "terraform", map[string]string{"main.tf": "provider \"google\" {}\n", ".terraform.lock.hcl": "# lock\n"}, "lockfile_terraform", StatusPass},
		{"terraform with no root config is skipped not passed", "terraform", nil, "security_hardening", StatusSkip},
		{"helm chart with dependencies warns without Chart.lock", "helm", map[string]string{"Chart.yaml": chartWithDeps}, "lockfile_helm", StatusWarn},
		{"helm chart with dependencies and Chart.lock passes", "helm", map[string]string{"Chart.yaml": chartWithDeps, "Chart.lock": "digest: x\n"}, "lockfile_helm", StatusPass},
		{"helm chart without dependencies needs no Chart.lock", "helm", map[string]string{"Chart.yaml": chartNoDeps}, "lockfile_helm", StatusSkip},
		{"npm-shrinkwrap.json is a JavaScript lock file", "javascript", map[string]string{"npm-shrinkwrap.json": "{}"}, "lockfile_javascript", StatusPass},
		{"bun.lock is a JavaScript lock file", "javascript", map[string]string{"bun.lock": "{}"}, "lockfile_javascript", StatusPass},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			results := CheckSecurityHardening(CheckContext{
				ProjectRoot: dir,
				QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{{Name: tt.lang}}},
			})

			got := findResult(results, tt.resultName)
			if got == nil {
				t.Fatalf("expected %s result, got %+v", tt.resultName, results)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s (%s)", got.Status, tt.wantStatus, got.Message)
			}
		})
	}
}
