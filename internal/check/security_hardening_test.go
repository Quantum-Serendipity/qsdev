package check

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	// Register the ecosystem modules whose generated security configs the
	// hardening check derives its expected settings from.
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/helm"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/php"
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
			files:      map[string]string{"pip.conf": "[global]\nonly-binary = :all:\n"},
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

// TestCheckSecurityHardening_JavaScriptSubproject verifies that the lock file
// and security config checks look where generation writes for a JavaScript
// project in a subdirectory (a Go service with its UI in frontend/), and that
// they honour detected extras such as Yarn Classic.
func TestCheckSecurityHardening_JavaScriptSubproject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		files        map[string]string
		wantLock     CheckStatus
		wantLockPath string
		wantConfig   CheckStatus
		wantConfPath string
	}{
		{
			name: "hardened subproject",
			files: map[string]string{
				"go.mod":                     "module x\n",
				"frontend/package.json":      `{"dependencies":{"react":"19.0.0"}}`,
				"frontend/package-lock.json": "{}",
				"frontend/.npmrc":            hardenedNpmrc,
			},
			wantLock:     StatusPass,
			wantLockPath: "frontend/package-lock.json",
			wantConfig:   StatusPass,
			wantConfPath: "frontend/.npmrc",
		},
		{
			name: "subproject without lock file or npmrc",
			files: map[string]string{
				"frontend/package.json": `{"dependencies":{"react":"19.0.0"}}`,
			},
			wantLock:     StatusFail,
			wantLockPath: "",
			wantConfig:   StatusFail,
			wantConfPath: "frontend/.npmrc",
		},
		{
			name: "root npmrc does not harden the subproject",
			files: map[string]string{
				"web/package.json":   `{}`,
				"web/pnpm-lock.yaml": "lockfileVersion: '9.0'\n",
				".npmrc":             hardenedNpmrc,
			},
			wantLock:     StatusPass,
			wantLockPath: "web/pnpm-lock.yaml",
			wantConfig:   StatusFail,
			wantConfPath: "web/pnpm-workspace.yaml",
		},
		{
			name: "yarn classic subproject checks .yarnrc",
			files: map[string]string{
				"ui/package.json": `{"packageManager":"yarn@1.22.22"}`,
				"ui/yarn.lock":    "# yarn lockfile v1\n",
				"ui/.yarnrc":      "ignore-scripts true\n",
			},
			wantLock:     StatusPass,
			wantLockPath: "ui/yarn.lock",
			wantConfig:   StatusPass,
			wantConfPath: "ui/.yarnrc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				p := filepath.Join(dir, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			results := CheckSecurityHardening(CheckContext{
				ProjectRoot: dir,
				QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{{Name: "javascript"}}},
			})

			for _, want := range []struct {
				name   string
				status CheckStatus
				path   string
			}{
				{"lockfile_javascript", tt.wantLock, tt.wantLockPath},
				{"security_config_javascript", tt.wantConfig, tt.wantConfPath},
			} {
				got := findResult(results, want.name)
				if got == nil {
					t.Fatalf("no %s result in %+v", want.name, results)
				}
				if got.Status != want.status || got.FilePath != want.path {
					t.Errorf("%s = %s %q (%s), want %s %q", want.name, got.Status, got.FilePath, got.Message, want.status, want.path)
				}
			}
		})
	}
}

// TestCheckSecurityHardening_AgeGateBelowComplianceLevel verifies that the
// generated release-age gate the check compares against follows the project's
// compliance level: a strict project's .npmrc must wait at least 14 days.
func TestCheckSecurityHardening_AgeGateBelowComplianceLevel(t *testing.T) {
	t.Parallel()
	npmrc := func(days string) string {
		return strings.Replace(hardenedNpmrc, "min-release-age=3", "min-release-age="+days, 1)
	}
	tests := []struct {
		name        string
		level       string
		client      string
		npmrc       string
		wantStatus  CheckStatus
		wantMessage string
	}{
		{name: "strict below level", level: "strict", npmrc: npmrc("3"), wantStatus: StatusFail, wantMessage: "min-release-age=14"},
		{name: "strict at level", level: "strict", npmrc: npmrc("14"), wantStatus: StatusPass},
		{name: "strict above level", level: "strict", npmrc: npmrc("21"), wantStatus: StatusPass},
		{name: "enhanced below level", level: "enhanced", npmrc: npmrc("3"), wantStatus: StatusFail, wantMessage: "min-release-age=7"},
		{name: "client raises level", level: "baseline", client: "strict", npmrc: npmrc("7"), wantStatus: StatusFail, wantMessage: "min-release-age=14"},
		{name: "baseline", level: "baseline", npmrc: npmrc("3"), wantStatus: StatusPass},
		{name: "unset level is baseline", npmrc: npmrc("3"), wantStatus: StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".npmrc"), []byte(tt.npmrc), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg := &types.QsdevConfig{
				Languages: []types.LanguageConfig{{Name: "javascript"}},
				Security:  types.SecurityConfig{Level: tt.level},
			}
			if tt.client != "" {
				cfg.Client = &types.ClientConfig{SecurityLevel: tt.client}
			}
			got := findResult(CheckSecurityHardening(CheckContext{ProjectRoot: dir, QsdevConfig: cfg}), "security_config_javascript")
			if got == nil {
				t.Fatal("no security_config_javascript result")
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("Status = %s, want %s (%s)", got.Status, tt.wantStatus, got.Message)
			}
			if tt.wantMessage == "" {
				return
			}
			if !strings.Contains(got.Message, tt.wantMessage) {
				t.Errorf("Message = %q, want it to name %s", got.Message, tt.wantMessage)
			}
			if !strings.Contains(got.Remediation, tt.wantMessage) {
				t.Errorf("Remediation = %q, want it to name %s", got.Remediation, tt.wantMessage)
			}
		})
	}
}

func TestFlattenJSONSettings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "scalar leaves as dotted keys",
			in:   `{"a": "x", "b": {"c": true, "d": null}}`,
			want: map[string]string{"(root)": "{2}", "a": "x", "b": "{2}", "b.c": "true", "b.d": "null"},
		},
		{
			name: "array indices are segments",
			in:   `{"repositories": [{"type": "composer"}, {"packagist.org": false}]}`,
			want: map[string]string{
				"(root)":                       "{1}",
				"repositories":                 "[2]",
				"repositories.0":               "{1}",
				"repositories.0.type":          "composer",
				"repositories.1":               "{1}",
				"repositories.1.packagist.org": "false",
			},
		},
		{
			name: "empty containers record a zero count",
			in:   `{"allow-plugins": {}, "list": []}`,
			want: map[string]string{"(root)": "{2}", "allow-plugins": "{0}", "list": "[0]"},
		},
		{
			name: "underscore keys are skipped at any depth and not counted",
			in:   `{"_comment": "x", "_nested": {"k": 1}, "c": {"_note": "y", "k": 2}}`,
			want: map[string]string{"(root)": "{1}", "c": "{1}", "c.k": "2"},
		},
		{
			name: "numbers stay exact",
			in:   `{"big": 12345678901234567890, "frac": 0.10, "exp": 1e3}`,
			want: map[string]string{"(root)": "{3}", "big": "12345678901234567890", "frac": "0.10", "exp": "1e3"},
		},
		{
			name: "top-level array",
			in:   `[true]`,
			want: map[string]string{"(root)": "[1]", "0": "true"},
		},
		{name: "malformed", in: `{"a": `, wantErr: true},
		{name: "trailing data", in: `{"a": 1} {"b": 2}`, wantErr: true},
		{name: "empty input", in: ``, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := flattenJSONSettings([]byte(tt.in))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("flattenJSONSettings(%q) = %v, want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("flattenJSONSettings(%q): %v", tt.in, err)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("flattenJSONSettings(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// testProxyInfra is a registry proxy configuration for check tests.
var testProxyInfra = types.InfraConfig{RegistryProxy: "https://nexus.example.test"}

// writeGeneratedSecurityConfigs writes the security configs generation
// produces for lang under qcfg into dir and returns them: the same
// ecosystem.ToGenerationConfig input the generator uses, with the
// infrastructure .qsdev.yaml carries and its compliance release-age window.
func writeGeneratedSecurityConfigs(t *testing.T, dir string, qcfg *types.QsdevConfig, lang types.LanguageConfig) []types.GeneratedFile {
	t.Helper()
	mod, ok := ecosystem.DefaultRegistry().ByName(lang.Name)
	if !ok {
		t.Fatalf("module %q not registered", lang.Name)
	}
	cfg := ecosystem.ToGenerationConfig(
		types.LanguageChoice{Name: lang.Name, PackageManager: lang.PackageManager},
		types.WizardAnswers{Infrastructure: qcfg.Infrastructure},
		catalog.EffectiveAgeGate(config.EffectiveSecurityLevel(qcfg)),
	)
	files := mod.SecurityConfigs(cfg)
	for _, gf := range files {
		p := filepath.Join(dir, filepath.FromSlash(gf.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, gf.Content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return files
}

// resultsNamed returns every result with the given name.
func resultsNamed(results []CheckResult, name string) []CheckResult {
	var out []CheckResult
	for _, r := range results {
		if r.Name == name {
			out = append(out, r)
		}
	}
	return out
}

// TestCheckSecurityConfig_PHP verifies qsdev check reads the tracked
// Composer home config.json as JSON: the generated files pass, and removing
// or weakening any hardening setting fails with the setting named.
func TestCheckSecurityConfig_PHP(t *testing.T) {
	t.Parallel()
	home := branding.Get().ComposerHomeDir()
	configPath := home + "/config.json"
	gitignorePath := home + "/.gitignore"
	settings := func(m map[string]any) map[string]any { return m["config"].(map[string]any) }

	tests := []struct {
		name        string
		genInfra    types.InfraConfig // infrastructure the files were generated with
		checkInfra  types.InfraConfig // infrastructure .qsdev.yaml holds at check time
		mutate      func(m map[string]any)
		raw         string // replaces config.json verbatim when set
		gitignore   string // replaces the nested .gitignore verbatim when set
		remove      string // generated file to delete
		wantStatus  CheckStatus
		wantPath    string // result to inspect; "" means every result
		wantMessage string
	}{
		{name: "generated with proxy", genInfra: testProxyInfra, checkInfra: testProxyInfra, wantStatus: StatusPass},
		{name: "generated without proxy", wantStatus: StatusPass},
		{
			name: "repositories deleted", genInfra: testProxyInfra, checkInfra: testProxyInfra,
			mutate:     func(m map[string]any) { delete(m, "repositories") },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "repositories.1.packagist.org=false",
		},
		{
			name: "packagist.org re-enabled", genInfra: testProxyInfra, checkInfra: testProxyInfra,
			mutate:     func(m map[string]any) { m["repositories"].([]any)[1].(map[string]any)["packagist.org"] = true },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "repositories.1.packagist.org=false",
		},
		{
			name: "proxy added after generation", checkInfra: testProxyInfra,
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "repositories.0.url=https://nexus.example.test/repository/composer-proxy/",
		},
		{
			name:       "secure-http disabled",
			mutate:     func(m map[string]any) { settings(m)["secure-http"] = false },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "config.secure-http=true",
		},
		{
			name:       "abandoned packages only warn",
			mutate:     func(m map[string]any) { settings(m)["audit"] = map[string]any{"abandoned": "warn"} },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "config.audit.abandoned=fail",
		},
		{
			name:       "every plugin allowed",
			mutate:     func(m map[string]any) { settings(m)["allow-plugins"] = true },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "config.allow-plugins must be an empty object, not true",
		},
		{
			// Composer merges allow-plugins, so a wildcard here allows every
			// plugin as surely as true.
			name:       "wildcard plugin allowed",
			mutate:     func(m map[string]any) { settings(m)["allow-plugins"] = map[string]any{"*": true} },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "config.allow-plugins has unexpected entries: *",
		},
		{
			name:       "named plugins allowed",
			mutate:     func(m map[string]any) { settings(m)["allow-plugins"] = map[string]any{"vendor/plugin": true} },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "config.allow-plugins has unexpected entries: vendor/plugin",
		},
		{
			name: "unproxied repository appended", genInfra: testProxyInfra, checkInfra: testProxyInfra,
			mutate: func(m map[string]any) {
				m["repositories"] = append(m["repositories"].([]any), map[string]any{"type": "composer", "url": "http://evil.example.test/"})
			},
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "repositories must have exactly 2 entries (found 3)",
		},
		{
			name: "repository added without proxy",
			mutate: func(m map[string]any) {
				m["repositories"] = []any{map[string]any{"type": "composer", "url": "http://evil.example.test/"}}
			},
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "the top level has unexpected entries: repositories",
		},
		{
			name:       "audit advisories ignored",
			mutate:     func(m map[string]any) { settings(m)["audit"].(map[string]any)["ignore"] = []any{"GHSA-xxxx"} },
			wantStatus: StatusFail, wantPath: configPath, wantMessage: "config.audit has unexpected entries: ignore",
		},
		{
			name:       "comment edited",
			mutate:     func(m map[string]any) { m["_comment"] = "edited" },
			wantStatus: StatusPass,
		},
		{name: "malformed JSON", raw: `{"config": {`, wantStatus: StatusFail, wantPath: configPath, wantMessage: "not valid JSON"},
		{name: "config.json missing", remove: configPath, wantStatus: StatusFail, wantPath: configPath, wantMessage: "could not be read"},
		{name: ".gitignore missing", remove: gitignorePath, wantStatus: StatusFail, wantPath: gitignorePath, wantMessage: "could not be read"},
		{
			name: ".gitignore emptied", gitignore: "\n",
			wantStatus: StatusFail, wantPath: gitignorePath, wantMessage: "patterns differ: want `*`, `!.gitignore`, `!config.json`; found no patterns",
		},
		{
			name: ".gitignore re-includes auth.json", gitignore: "*\n!.gitignore\n!config.json\n!auth.json\n",
			wantStatus: StatusFail, wantPath: gitignorePath, wantMessage: "found `*`, `!.gitignore`, `!config.json`, `!auth.json`",
		},
		{
			// A later pattern overrides an earlier one: * after !config.json
			// ignores the policy file again.
			name: ".gitignore reordered", gitignore: "!config.json\n*\n!.gitignore\n",
			wantStatus: StatusFail, wantPath: gitignorePath, wantMessage: "found `!config.json`, `*`, `!.gitignore`",
		},
		{
			name: ".gitignore with comments and blank lines", gitignore: "# composer home\n*\n\n!.gitignore\r\n!config.json  \n",
			wantStatus: StatusPass, wantPath: gitignorePath,
		},
		{
			name: ".gitignore negation indented", gitignore: "*\n!.gitignore\n  !config.json\n",
			wantStatus: StatusFail, wantPath: gitignorePath, wantMessage: "found `*`, `!.gitignore`, `  !config.json`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			lang := types.LanguageConfig{Name: ecosystem.NamePHP}
			writeGeneratedSecurityConfigs(t, dir, &types.QsdevConfig{Infrastructure: tt.genInfra}, lang)
			cfgFile := filepath.Join(dir, filepath.FromSlash(configPath))
			if tt.mutate != nil {
				data, err := os.ReadFile(cfgFile)
				if err != nil {
					t.Fatal(err)
				}
				var m map[string]any
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatal(err)
				}
				tt.mutate(m)
				if data, err = json.Marshal(m); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(cfgFile, data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.raw != "" {
				if err := os.WriteFile(cfgFile, []byte(tt.raw), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.gitignore != "" {
				if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(gitignorePath)), []byte(tt.gitignore), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.remove != "" {
				if err := os.Remove(filepath.Join(dir, filepath.FromSlash(tt.remove))); err != nil {
					t.Fatal(err)
				}
			}

			results := resultsNamed(CheckSecurityHardening(CheckContext{
				ProjectRoot: dir,
				QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{lang}, Infrastructure: tt.checkInfra},
			}), "security_config_php")
			if len(results) != 2 {
				t.Fatalf("got %d security_config_php results, want 2 (config.json and .gitignore): %+v", len(results), results)
			}
			for _, r := range results {
				if tt.wantPath != "" && r.FilePath != tt.wantPath {
					continue
				}
				if r.Status != tt.wantStatus {
					t.Errorf("%s: Status = %s, want %s (%s)", r.FilePath, r.Status, tt.wantStatus, r.Message)
				}
				if !strings.Contains(r.Message, tt.wantMessage) {
					t.Errorf("%s: Message = %q, want it to name %q", r.FilePath, r.Message, tt.wantMessage)
				}
				// Container markers are internal: users must never see them.
				for _, text := range []string{r.Message, r.Remediation} {
					if markerText.MatchString(text) || strings.Contains(text, rootKey) || strings.Contains(text, "pattern.") {
						t.Errorf("%s: %q shows an internal marker", r.FilePath, text)
					}
				}
			}
		})
	}
}

// markerText matches a container marker (jsonObjectMarker, jsonArrayMarker)
// rendered as a setting value.
var markerText = regexp.MustCompile(`=[{\[][0-9]+[}\]]`)

// TestCheckSecurityConfig_ProxyStillPasses pins that resolving the registry
// proxy at check time does not break ecosystems that already write it: the
// configs generation produces for each JavaScript package manager and for pip,
// with and without a proxy, still pass.
func TestCheckSecurityConfig_ProxyStillPasses(t *testing.T) {
	t.Parallel()
	langs := []types.LanguageConfig{
		{Name: ecosystem.NameJavaScript, PackageManager: "npm"},
		{Name: ecosystem.NameJavaScript, PackageManager: "pnpm"},
		{Name: ecosystem.NameJavaScript, PackageManager: "yarn"},
		{Name: ecosystem.NameJavaScript, PackageManager: "bun"},
		{Name: ecosystem.NamePython, PackageManager: "pip"},
		{Name: ecosystem.NamePython},
		{Name: ecosystem.NamePHP},
	}
	for _, lang := range langs {
		for _, infra := range []types.InfraConfig{{}, testProxyInfra} {
			t.Run(lang.Name+"/"+lang.PackageManager+"/proxy="+infra.RegistryProxy, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				qcfg := &types.QsdevConfig{Languages: []types.LanguageConfig{lang}, Infrastructure: infra}
				files := writeGeneratedSecurityConfigs(t, dir, qcfg, lang)
				if len(files) == 0 {
					t.Fatal("generation produced no security configs")
				}
				results := resultsNamed(CheckSecurityHardening(CheckContext{ProjectRoot: dir, QsdevConfig: qcfg}), "security_config_"+lang.Name)
				if len(results) != len(files) {
					t.Fatalf("got %d results for %d generated files: %+v", len(results), len(files), results)
				}
				for _, r := range results {
					if r.Status != StatusPass {
						t.Errorf("%s: Status = %s, want PASS (%s)", r.FilePath, r.Status, r.Message)
					}
				}
			})
		}
	}
}
