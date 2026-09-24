package python_test

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*python.Module)(nil)

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, &python.Module{}, "python", "Python", 1)
}

// --- Detection tests ---

func TestDetect_PyprojectTomlOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when pyproject.toml is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "pyproject.toml")
	if result.SuggestedConfig.PackageManager != "pip" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "pip")
	}
}

func TestDetect_RequirementsTxtOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0.0\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when requirements.txt is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "requirements.txt")
}

func TestDetect_SetupPyOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "setup.py", "from setuptools import setup\nsetup(name='myapp')\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when setup.py is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "setup.py")
}

func TestDetect_PipfileOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Pipfile", "[packages]\nflask = \"*\"\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when Pipfile is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "Pipfile")
}

func TestDetect_UvLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, "uv.lock", "# uv lockfile\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "uv" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "uv")
	}
	assertEvidenceContains(t, result.Evidence, "uv.lock")
}

func TestDetect_PoetryLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, "poetry.lock", "# poetry lockfile\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "poetry" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "poetry")
	}
	assertEvidenceContains(t, result.Evidence, "poetry.lock")
}

func TestDetect_UvLockTakesPriorityOverPoetryLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, "uv.lock", "# uv lockfile\n")
	writeFile(t, dir, "poetry.lock", "# poetry lockfile\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if result.SuggestedConfig.PackageManager != "uv" {
		t.Errorf("PackageManager = %q, want %q (uv.lock should take priority over poetry.lock)",
			result.SuggestedConfig.PackageManager, "uv")
	}
}

func TestDetect_PythonVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")
	writeFile(t, dir, ".python-version", "3.11.5\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != "3.11.5" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "3.11.5")
	}
	assertEvidenceContains(t, result.Evidence, ".python-version")
}

func TestDetect_PyprojectRequiresPython(t *testing.T) {
	dir := t.TempDir()
	pyproject := `[project]
name = "myapp"
requires-python = ">=3.10"
`
	writeFile(t, dir, "pyproject.toml", pyproject)

	m := &python.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != "3.10" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "3.10")
	}
	assertEvidenceContains(t, result.Evidence, "requires-python")
}

func TestDetect_PythonVersionFileTakesPriorityOverRequiresPython(t *testing.T) {
	dir := t.TempDir()
	pyproject := `[project]
name = "myapp"
requires-python = ">=3.10"
`
	writeFile(t, dir, "pyproject.toml", pyproject)
	writeFile(t, dir, ".python-version", "3.13.0\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if result.SuggestedConfig.Version != "3.13.0" {
		t.Errorf("Version = %q, want %q (.python-version should take priority)",
			result.SuggestedConfig.Version, "3.13.0")
	}
}

func TestDetect_RequiresPythonVariants(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		version string
	}{
		{"ge_only", `requires-python = ">=3.11"`, "3.11"},
		{"tilde_eq", `requires-python = "~=3.9"`, "3.9"},
		{"exact", `requires-python = "==3.12"`, "3.12"},
		// An upper bound alone must never pin the excluded version itself.
		{"lt", `requires-python = "<3.13"`, "3.12"},
		{"no_specifier", `requires-python = "3.10"`, "3.10"},
		{"with_spaces", `  requires-python = ">=3.11"`, "3.11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "pyproject.toml", "[project]\n"+tt.line+"\n")

			m := &python.Module{}
			result := m.Detect(dir)

			if result.SuggestedConfig.Version != tt.version {
				t.Errorf("Version = %q, want %q for line %q",
					result.SuggestedConfig.Version, tt.version, tt.line)
			}
		})
	}
}

// TestDetect_PythonVersionConstraints guards W073: the pinned version must
// satisfy the project's constraint (never an excluded bound), PEP 440
// whitespace and TOML literal strings parse, and Poetry's python constraint is
// read when requires-python is absent.
func TestDetect_PythonVersionConstraints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		pyproject string
		want      string
	}{
		{"upper bound first", "[project]\nrequires-python = \"<3.13,>=3.10\"\n", "3.10"},
		{"exclusion first", "[project]\nrequires-python = \"!=3.9.*,>=3.8\"\n", "3.8"},
		{"exclusion of lower bound", "[project]\nrequires-python = \">=3.9,!=3.9.*\"\n", "3.10"},
		{"space after operator", "[project]\nrequires-python = \">= 3.10\"\n", "3.10"},
		{"single quoted", "[project]\nrequires-python = '>=3.11'\n", "3.11"},
		{"upper bound only", "[project]\nrequires-python = \"<3.11\"\n", "3.10"},
		{"upper bound above default", "[project]\nrequires-python = \"<4\"\n", "3.12"},
		{"patch upper bound excludes minor", "[project]\nrequires-python = \">=3.9,<3.11.2\"\n", "3.9"},
		{"exact patch", "[project]\nrequires-python = \"==3.11.4\"\n", "3.11.4"},
		{"wildcard", "[project]\nrequires-python = \"==3.11.*\"\n", "3.11"},
		{"compatible release", "[project]\nrequires-python = \"~=3.10\"\n", "3.10"},
		{"poetry caret", "[tool.poetry.dependencies]\npython = \"^3.9\"\n", "3.9"},
		{"poetry classic range", "[tool.poetry.dependencies]\npython = \">=3.8,<3.11\"\n", "3.8"},
		{"poetry space-separated", "[tool.poetry.dependencies]\npython = \">=3.9 <3.12\"\n", "3.9"},
		{"poetry tilde", "[tool.poetry.dependencies]\npython = \"~3.11\"\n", "3.11"},
		{"poetry or", "[tool.poetry.dependencies]\npython = \"~3.10 || ~3.12\"\n", "3.10"},
		{"poetry any", "[tool.poetry.dependencies]\npython = \"*\"\n", ""},
		{"major-only lower bound", "[project]\nrequires-python = \">=3\"\n", "3.12"},
		{"poetry major-only caret", "[tool.poetry.dependencies]\npython = \"^3\"\n", "3.12"},
		{"python 2 compatible", "[project]\nrequires-python = \">=2.7,!=3.0.*,!=3.1.*\"\n", "3.8"},
		{"poetry python 2 or 3", "[tool.poetry.dependencies]\npython = \"^2.7 || ^3.9\"\n", "3.9"},
		{"requires-python wins over poetry", "[project]\nrequires-python = \">=3.12\"\n[tool.poetry.dependencies]\npython = \"^3.9\"\n", "3.12"},
		{"unsatisfiable", "[project]\nrequires-python = \">=3.12,<3.10\"\n", ""},
		{"garbage", "[project]\nrequires-python = \"python3\"\n", ""},
		{"invalid toml", "[project\nrequires-python = \">=3.11\"\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, dir, "pyproject.toml", tt.pyproject)
			if got := (&python.Module{}).Detect(dir).SuggestedConfig.Version; got != tt.want {
				t.Errorf("Version = %q, want %q for:\n%s", got, tt.want, tt.pyproject)
			}
		})
	}
}

// TestDetect_PythonVersionFileForms guards W073: .python-version comments are
// skipped, uv's cpython-/platform forms are reduced to the version, and values
// nixpkgs-python cannot provide fall back to pyproject instead of being pinned.
func TestDetect_PythonVersionFileForms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"comment first", "# pinned for CI\n\n3.11\n", "3.11"},
		{"uv platform request", "cpython-3.12.4-linux-x86_64-gnu\n", "3.12.4"},
		{"uv at request", "cpython@3.13\n", "3.13"},
		{"pypy", "pypy3.10\n", "3.9"},
		{"free-threaded", "3.13t\n", "3.9"},
		{"system", "system\n", "3.9"},
		{"only comments", "# nothing\n", "3.9"},
		{"zero-width space", "3.11\u200b\n", "3.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, dir, "pyproject.toml", "[project]\nrequires-python = \">=3.9\"\n")
			writeFile(t, dir, ".python-version", tt.content)
			if got := (&python.Module{}).Detect(dir).SuggestedConfig.Version; got != tt.want {
				t.Errorf("Version = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDetect_ToolConfig guards W074 and W071: the project's own mypy, bandit
// and uv exclude-newer settings are recorded for the generators.
func TestDetect_ToolConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  map[string]string
	}{
		{"none", map[string]string{"pyproject.toml": "[project]\nname = \"x\"\n"}, nil},
		{"tool.mypy", map[string]string{"pyproject.toml": "[tool.mypy]\n"}, map[string]string{"mypy": "true"}},
		{"mypy.ini", map[string]string{"requirements.txt": "", "mypy.ini": "[mypy]\n"}, map[string]string{"mypy": "true"}},
		{"setup.cfg mypy", map[string]string{"setup.cfg": "[metadata]\nname = x\n[mypy]\nstrict = True\n"}, map[string]string{"mypy": "true"}},
		{"setup.cfg without mypy", map[string]string{"setup.cfg": "[metadata]\nname = x\n"}, nil},
		{"tool.bandit", map[string]string{"pyproject.toml": "[tool.bandit]\nskips = [\"B101\"]\n"}, map[string]string{"bandit_config": "pyproject.toml"}},
		{"tool.uv exclude-newer", map[string]string{"pyproject.toml": "[tool.uv]\nexclude-newer = \"2025-01-01T00:00:00Z\"\n", "uv.lock": ""}, map[string]string{"uv_exclude_newer": "project"}},
		{"uv.toml exclude-newer", map[string]string{"pyproject.toml": "[project]\n", "uv.toml": "exclude-newer = \"14 days\"\n"}, map[string]string{"uv_exclude_newer": "project"}},
		{"tool.uv without exclude-newer", map[string]string{"pyproject.toml": "[tool.uv]\ndev-dependencies = []\n"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			got := (&python.Module{}).Detect(dir).SuggestedConfig.Extras
			if !maps.Equal(got, tt.want) {
				t.Errorf("Extras = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetect_ProjectManagers guards W077: projects managed by tools qsdev
// does not support are not reported as pip, and further Python layouts are
// detected.
func TestDetect_ProjectManagers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		files       map[string]string
		wantPM      string
		wantManager string
		wantEvid    string
	}{
		{"pdm lock", map[string]string{"pyproject.toml": "[project]\n", "pdm.lock": ""}, "", "pdm", "pdm.lock"},
		{"pdm dev-dependencies", map[string]string{"pyproject.toml": "[tool.pdm.dev-dependencies]\ntest = []\n"}, "", "pdm", "tool.pdm.dev-dependencies"},
		{"pdm build backend only", map[string]string{"pyproject.toml": "[tool.pdm.build]\nincludes = []\n"}, "pip", "", ""},
		{"pipenv", map[string]string{"Pipfile": "[packages]\n", "Pipfile.lock": "{}"}, "", "pipenv", "Pipfile.lock"},
		{"hatch envs", map[string]string{"pyproject.toml": "[tool.hatch.envs.default]\ndependencies = []\n"}, "", "hatch", "tool.hatch.envs"},
		{"hatch build only", map[string]string{"pyproject.toml": "[tool.hatch.version]\npath = \"x.py\"\n"}, "pip", "", ""},
		{"conda", map[string]string{"environment.yml": "name: x\n"}, "", "conda", "environment.yml"},
		{"uv lock wins", map[string]string{"pyproject.toml": "[tool.pdm.dev-dependencies]\n", "uv.lock": ""}, "uv", "", ""},
		{"setup.cfg only", map[string]string{"setup.cfg": "[metadata]\n"}, "pip", "", "setup.cfg"},
		{"requirements.in only", map[string]string{"requirements.in": "flask\n"}, "pip", "", "requirements.in"},
		{"requirements-dev.txt only", map[string]string{"requirements-dev.txt": "pytest\n"}, "pip", "", "requirements-dev.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			result := (&python.Module{}).Detect(dir)
			if !result.Detected {
				t.Fatal("expected Detected=true")
			}
			if got := result.SuggestedConfig.PackageManager; got != tt.wantPM {
				t.Errorf("PackageManager = %q, want %q", got, tt.wantPM)
			}
			if got := result.SuggestedConfig.Extras["project_manager"]; got != tt.wantManager {
				t.Errorf("Extras[project_manager] = %q, want %q", got, tt.wantManager)
			}
			if tt.wantEvid != "" {
				assertEvidenceContains(t, result.Evidence, tt.wantEvid)
			}
		})
	}
}

// TestManifestFiles_ProjectManagers guards W077: an unsupported manager's
// real manifest and lockfile are reported instead of requirements.txt.
func TestManifestFiles_ProjectManagers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		manager    string
		wantPath   string
		wantLock   string
		wantPolicy ecosystem.LockFilePolicy
	}{
		{"", "requirements.txt", "", ecosystem.LockFilePolicyNone},
		{"pdm", "pyproject.toml", "pdm.lock", ecosystem.LockFilePolicyRequired},
		{"pipenv", "Pipfile", "Pipfile.lock", ecosystem.LockFilePolicyRequired},
		{"hatch", "pyproject.toml", "", ecosystem.LockFilePolicyNone},
		{"conda", "environment.yml", "", ecosystem.LockFilePolicyNone},
	}
	for _, tt := range tests {
		t.Run("manager="+tt.manager, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Extras: map[string]string{"project_manager": tt.manager}}
			got := (&python.Module{}).ManifestFiles(cfg)
			if len(got) != 1 || got[0].Path != tt.wantPath || got[0].LockFile != tt.wantLock || got[0].LockFilePolicy != tt.wantPolicy {
				t.Errorf("ManifestFiles() = %+v, want %s/%q/%v", got, tt.wantPath, tt.wantLock, tt.wantPolicy)
			}
		})
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &python.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

func TestDetect_HighestConfidenceWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==3.0.0\n")
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\n")

	m := &python.Module{}
	result := m.Detect(dir)

	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain (pyproject.toml should elevate to Certain)",
			result.Confidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_Pip(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		PackageManager: "pip",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "languages.python")
	assertContains(t, fragment, "enable = true")
	assertContains(t, fragment, `version = "3.12"`)
	assertContains(t, fragment, "venv.enable = true")
	assertNotContains(t, fragment, "uv.enable")
	assertNotContains(t, fragment, "poetry.enable")
}

func TestDevenvNixFragment_Uv(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		PackageManager: "uv",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "uv.enable = true")
	assertNotContains(t, fragment, "poetry.enable")
}

func TestDevenvNixFragment_Poetry(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		PackageManager: "poetry",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	assertContains(t, fragment, "poetry.enable = true")
	assertNotContains(t, fragment, "uv.enable")
}

// TestDevenvNixFragment_PoetryOwnsVenv guards W072: devenv's virtualenv task
// runs after the poetry task and would activate a second, empty venv over
// poetry's .venv, and poetry.install must never resolve without a lockfile
// or run against one `poetry check --lock` rejects.
func TestDevenvNixFragment_PoetryOwnsVenv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pm      string
		want    []string
		notWant []string
	}{
		{
			pm: "poetry",
			want: []string{
				"poetry.enable = true;",
				"poetry.install.enable = true;",
				"poetry.activate.enable = true;",
				`tasks."qsdev:python:poetry-check-lock" = lib.mkIf config.languages.python.poetry.install.enable {`,
				"if [ ! -f pyproject.toml ]; then",
				"if [ ! -f poetry.lock ]; then",
				"if ! ${config.languages.python.poetry.package}/bin/poetry check --lock --no-interaction; then",
				"cwd = config.devenv.root;",
				`before = [ "devenv:python:poetry" ];`,
			},
			// pathExists results stick in devenv's evaluation cache, so a
			// lockfile created later would never enable the install.
			notWant: []string{"venv.enable", "pathExists"},
		},
		{pm: "pip", want: []string{"venv.enable = true;"}, notWant: []string{"poetry.", "tasks."}},
		{pm: "uv", want: []string{"venv.enable = true;"}, notWant: []string{"poetry.", "tasks."}},
	}
	for _, tt := range tests {
		t.Run(tt.pm, func(t *testing.T) {
			t.Parallel()
			fragment, err := (&python.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: tt.pm})
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			for _, w := range tt.want {
				assertContains(t, fragment, w)
			}
			for _, nw := range tt.notWant {
				assertNotContains(t, fragment, nw)
			}
		})
	}
}

// TestSetupWarnings guards W072: init warns when a poetry-mode project lacks
// the pyproject.toml devenv's poetry task needs, or the poetry.lock the shell
// needs before it installs and activates poetry's .venv.
func TestSetupWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		pm    string
		files []string
		want  string // substring of the only warning; "" for no warnings
	}{
		{name: "poetry without pyproject.toml", pm: "poetry", files: []string{"requirements.txt"}, want: "pyproject.toml is missing"},
		{name: "poetry lockfile without pyproject.toml", pm: "poetry", files: []string{"poetry.lock"}, want: "pyproject.toml is missing"},
		{name: "poetry without poetry.lock", pm: "poetry", files: []string{"pyproject.toml"}, want: "poetry.lock is missing"},
		{name: "poetry project", pm: "poetry", files: []string{"pyproject.toml", "poetry.lock"}},
		{name: "pip without pyproject.toml", pm: "pip", files: []string{"requirements.txt"}},
		{name: "default package manager", files: []string{"requirements.txt"}},
		{name: "uv without uv.lock", pm: "uv", files: []string{"pyproject.toml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				writeFile(t, dir, f, "")
			}
			got := (&python.Module{}).SetupWarnings(dir, ecosystem.ModuleConfig{PackageManager: tt.pm})
			if tt.want == "" {
				if len(got) != 0 {
					t.Errorf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Errorf("SetupWarnings() = %q, want one warning containing %q", got, tt.want)
			}
		})
	}
}

// TestSetupWarnings_UnsupportedManager guards W077: a pip-mode project that is
// really managed by pdm, pipenv, hatch or conda gets a visible warning naming
// that tool, instead of being silently configured as a pip project.
func TestSetupWarnings_UnsupportedManager(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		pm    string
		files map[string]string
		want  []string // substrings of the only warning; nil for no warnings
	}{
		{name: "pdm lock", files: map[string]string{"pyproject.toml": "[project]\n", "pdm.lock": ""}, want: []string{"pdm", "pdm.lock found", "pip, uv, poetry"}},
		{name: "pdm dev-dependencies", pm: "pip", files: map[string]string{"pyproject.toml": "[tool.pdm.dev-dependencies]\ntest = []\n"}, want: []string{"pdm", "[tool.pdm.dev-dependencies]"}},
		{name: "pdm scripts", files: map[string]string{"pyproject.toml": "[tool.pdm.scripts]\ntest = \"pytest\"\n"}, want: []string{"pdm", "[tool.pdm.scripts]"}},
		{name: "pdm source array", files: map[string]string{"pyproject.toml": "[[tool.pdm.source]]\nname = \"internal\"\nurl = \"https://pypi.example/simple\"\n"}, want: []string{"pdm", "[tool.pdm.source]"}},
		{name: "pdm build backend only", files: map[string]string{"pyproject.toml": "[tool.pdm.build]\nincludes = []\n"}},
		{name: "poetry chosen for pdm project", pm: "poetry", files: map[string]string{"pyproject.toml": "[project]\n", "pdm.lock": "", "poetry.lock": ""}},
		{name: "pipenv", files: map[string]string{"Pipfile": "[packages]\n", "Pipfile.lock": "{}"}, want: []string{"pipenv", "Pipfile.lock found"}},
		{name: "hatch envs", files: map[string]string{"pyproject.toml": "[tool.hatch.envs.default]\ndependencies = []\n"}, want: []string{"hatch", "[tool.hatch.envs]"}},
		{name: "conda", files: map[string]string{"environment.yaml": "name: x\n"}, want: []string{"conda", "environment.yaml found"}},
		{name: "hatch build backend only", files: map[string]string{"pyproject.toml": "[tool.hatch.version]\npath = \"x.py\"\n"}},
		{name: "plain pip project", files: map[string]string{"requirements.txt": "requests==2.32.3\n"}},
		{name: "uv chosen for pdm project", pm: "uv", files: map[string]string{"pyproject.toml": "[project]\n", "pdm.lock": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			got := (&python.Module{}).SetupWarnings(dir, ecosystem.ModuleConfig{PackageManager: tt.pm})
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("SetupWarnings() = %q, want exactly one warning", got)
			}
			for _, w := range tt.want {
				if !strings.Contains(got[0], w) {
					t.Errorf("SetupWarnings()[0] = %q, want it to contain %q", got[0], w)
				}
			}
		})
	}
}

// TestDevenvNixFragment_SupplyChainEnv guards W071: uv and poetry projects
// get their hardening from devenv env vars, and a project's own uv
// exclude-newer is never overridden.
func TestDevenvNixFragment_SupplyChainEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     ecosystem.ModuleConfig
		want    []string
		notWant []string
	}{
		{
			name:    "uv",
			cfg:     ecosystem.ModuleConfig{PackageManager: "uv"},
			want:    []string{`env.UV_EXCLUDE_NEWER = "P7D";`},
			notWant: []string{"PIP_CONFIG_FILE", "POETRY_"},
		},
		{
			name:    "uv with project exclude-newer",
			cfg:     ecosystem.ModuleConfig{PackageManager: "uv", Extras: map[string]string{"uv_exclude_newer": "project"}},
			notWant: []string{"UV_EXCLUDE_NEWER"},
		},
		{
			name:    "poetry",
			cfg:     ecosystem.ModuleConfig{PackageManager: "poetry"},
			want:    []string{`env.POETRY_INSTALLER_ONLY_BINARY = ":all:";`},
			notWant: []string{"PIP_CONFIG_FILE", "UV_EXCLUDE_NEWER"},
		},
		{
			name:    "pip",
			cfg:     ecosystem.ModuleConfig{PackageManager: "pip"},
			notWant: []string{"UV_EXCLUDE_NEWER", "POETRY_"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fragment, err := (&python.Module{}).DevenvNixFragment(tt.cfg)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			for _, w := range tt.want {
				assertContains(t, fragment, w)
			}
			for _, nw := range tt.notWant {
				assertNotContains(t, fragment, nw)
			}
		})
	}
}

// TestDevenvNixFragment_NixParses parses every package-manager variant with
// nix-instantiate (when available) inside a devenv-style module.
func TestDevenvNixFragment_NixParses(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	for _, pm := range []string{"pip", "uv", "poetry"} {
		t.Run(pm, func(t *testing.T) {
			t.Parallel()
			fragment, err := (&python.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: pm})
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			expr := "{ pkgs, lib, config, ... }:\n{\n" + fragment + "}\n"
			out, err := exec.CommandContext(t.Context(), nixInstantiate, "--parse", "-E", expr).CombinedOutput()
			if err != nil {
				t.Fatalf("nix-instantiate --parse failed: %v\n%s\nfragment:\n%s", err, out, fragment)
			}
		})
	}
}

func TestDevenvNixFragment_MutualExclusion(t *testing.T) {
	m := &python.Module{}

	// uv fragment should not contain poetry
	uvFrag, _ := m.DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: "uv"})
	if strings.Contains(uvFrag, "poetry.enable") {
		t.Error("uv fragment should not contain poetry.enable")
	}

	// poetry fragment should not contain uv
	poetryFrag, _ := m.DevenvNixFragment(ecosystem.ModuleConfig{PackageManager: "poetry"})
	if strings.Contains(poetryFrag, "uv.enable") {
		t.Error("poetry fragment should not contain uv.enable")
	}
}

func TestDevenvNixFragment_DefaultVersion(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	assertContains(t, fragment, `version = "3.12"`)
}

func TestDevenvNixFragment_CustomVersion(t *testing.T) {
	m := &python.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		Version: "3.11",
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	assertContains(t, fragment, `version = "3.11"`)
	assertNotContains(t, fragment, `version = "3.12"`)
}

// TestDetect_PythonVersionFileRejectsNonVersions guards against Nix
// antiquotation injection: .python-version is repo-controlled and its value is
// rendered into devenv.nix, so anything but a plain version must be dropped.
func TestDetect_PythonVersionFileRejectsNonVersions(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		"3.12${builtins.readFile /etc/hostname}",
		`3.12"; imports = [ ./evil.nix ]; x = "`,
		"3.12\u0000",
		"pypy3.10",
		"system",
	} {
		t.Run(content, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"\nrequires-python = \">=3.11\"\n")
			writeFile(t, dir, ".python-version", content+"\n")

			result := (&python.Module{}).Detect(dir)
			if result.SuggestedConfig.Version != "3.11" {
				t.Errorf("Version = %q, want fallback to requires-python %q", result.SuggestedConfig.Version, "3.11")
			}
		})
	}
}

func TestDevenvNixFragment_RejectsInvalidVersion(t *testing.T) {
	t.Parallel()
	for _, version := range []string{
		"3.12${builtins.readFile /etc/hostname}",
		`3.12"; x = "`,
		"3.12\u0000",
		`3.12\`,
		"latest",
		"3",
	} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			fragment, err := (&python.Module{}).DevenvNixFragment(ecosystem.ModuleConfig{Version: version})
			if err == nil {
				t.Errorf("DevenvNixFragment(%q) error = nil, want error\ngot:\n%s", version, fragment)
			}
		})
	}
}

// TestDevenvNixFragment_PipConfigWired asserts the generated pip.conf is
// actually applied: pip never reads a project-root pip.conf on its own.
func TestDevenvNixFragment_PipConfigWired(t *testing.T) {
	t.Parallel()
	const wiring = `env.PIP_CONFIG_FILE = "${config.devenv.root}/pip.conf";`
	tests := []struct {
		pm   string
		want bool
	}{
		{pm: "", want: true},
		{pm: "pip", want: true},
		{pm: "uv", want: false},
		{pm: "poetry", want: false},
	}
	for _, tt := range tests {
		t.Run("pm="+tt.pm, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{PackageManager: tt.pm}
			fragment, err := (&python.Module{}).DevenvNixFragment(cfg)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if got := strings.Contains(fragment, wiring); got != tt.want {
				t.Errorf("PIP_CONFIG_FILE wiring present = %v, want %v\ngot:\n%s", got, tt.want, fragment)
			}
			// Wiring must point at the file SecurityConfigs generates.
			files := (&python.Module{}).SecurityConfigs(cfg)
			if tt.want && (len(files) != 1 || files[0].Path != "pip.conf") {
				t.Errorf("SecurityConfigs() = %v, want a single pip.conf", files)
			}
		})
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_Pip(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pip"})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs(pip) returned %d files, want 1", len(configs))
	}

	cfg := configs[0]
	if cfg.Path != "pip.conf" {
		t.Errorf("Path = %q, want %q", cfg.Path, "pip.conf")
	}

	content := string(cfg.Content)
	assertContains(t, content, "[global]")
	assertContains(t, content, "only-binary = :all:")
	assertContains(t, content, "Security-hardened pip configuration")
	assertContains(t, content, branding.GeneratedBy())
	assertContains(t, content, "pip >= 26.0")
}

func TestSecurityConfigs_PipDefault(t *testing.T) {
	m := &python.Module{}
	// Empty PackageManager should default to pip.
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs(default) returned %d files, want 1", len(configs))
	}
	if configs[0].Path != "pip.conf" {
		t.Errorf("Path = %q, want %q", configs[0].Path, "pip.conf")
	}
}

func TestSecurityConfigs_Uv(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "uv"})

	if configs != nil {
		t.Errorf("SecurityConfigs(uv) = %v, want nil", configs)
	}
}

func TestSecurityConfigs_Poetry(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "poetry"})

	if configs != nil {
		t.Errorf("SecurityConfigs(poetry) = %v, want nil", configs)
	}
}

// --- Registry proxy tests ---

func TestSecurityConfigs_Pip_RegistryProxy(t *testing.T) {
	m := &python.Module{}
	proxy := "https://pypi.corp.example.com/simple/"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "pip",
		RegistryProxy:  proxy,
	})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs(pip+proxy) returned %d files, want 1", len(configs))
	}

	content := string(configs[0].Content)
	assertContains(t, content, "index-url = "+proxy)
	// Existing security settings must be preserved.
	assertContains(t, content, "only-binary = :all:")
	assertContains(t, content, "[global]")
}

func TestSecurityConfigs_Pip_NoRegistryProxy(t *testing.T) {
	m := &python.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pip"})

	content := string(configs[0].Content)
	assertNotContains(t, content, "index-url")
}

func TestSecurityConfigs_Pip_RegistryProxyPreservesExisting(t *testing.T) {
	m := &python.Module{}
	proxy := "https://pypi.corp.example.com/simple/"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "pip",
		RegistryProxy:  proxy,
	})

	content := string(configs[0].Content)
	assertContains(t, content, "Security-hardened pip configuration")
	assertContains(t, content, branding.GeneratedBy())
	assertContains(t, content, "only-binary = :all:")
}

// TestSecurityConfigs_PipHashCheckingScopedToLockedInstall asserts that hash
// checking is required only by the locked install, never by the pip.conf the
// devenv shell exports through PIP_CONFIG_FILE: a global require-hashes makes
// devenv's `venv --upgrade-deps` (an unpinned `pip install --upgrade pip`) and
// every `pip install -e .` fail. only-binary stays global because pip applies
// it only to index packages, not to the local project directory.
func TestSecurityConfigs_PipHashCheckingScopedToLockedInstall(t *testing.T) {
	t.Parallel()
	const locked = "pip install --require-hashes --only-binary :all: -r requirements.txt"
	tests := []struct {
		name string
		cfg  ecosystem.ModuleConfig
	}{
		{name: "default", cfg: ecosystem.ModuleConfig{}},
		{name: "pip", cfg: ecosystem.ModuleConfig{PackageManager: "pip"}},
		{name: "pip with registry proxy", cfg: ecosystem.ModuleConfig{
			PackageManager: "pip", RegistryProxy: "https://pypi.corp.example.com/simple/",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &python.Module{}
			files := m.SecurityConfigs(tt.cfg)
			if len(files) != 1 {
				t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
			}
			settings := pipConfSettings(string(files[0].Content))
			if v, ok := settings["require-hashes"]; ok {
				t.Errorf("pip.conf sets require-hashes = %q; hash checking must be scoped to the locked install", v)
			}
			if got := settings["only-binary"]; got != ":all:" {
				t.Errorf("pip.conf only-binary = %q, want %q", got, ":all:")
			}
			// The file tells developers how to get the hash-checked install.
			assertContains(t, string(files[0].Content), "#   "+locked+"\n")

			var ciInstall string
			for _, c := range m.CICommands(tt.cfg) {
				if c.Phase == ecosystem.CIPhaseInstall {
					ciInstall = c.Command
				}
			}
			if ciInstall != locked {
				t.Errorf("CI install command = %q, want %q", ciInstall, locked)
			}
		})
	}
}

// pipConfSettings returns the key = value settings of an INI-style pip.conf,
// ignoring comments and section headers.
func pipConfSettings(content string) map[string]string {
	settings := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			settings[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return settings
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := &python.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{Extras: map[string]string{"mypy": "true"}})

	// bandit is NOT a git-hooks.nix built-in, so it is rendered as a custom
	// hook (BuiltIn:false) with a NixPackage so an `entry` is always emitted.
	want := []struct {
		id         string
		builtIn    bool
		nixPackage string
		language   string
	}{
		{id: "ruff", builtIn: true, language: "python"},
		{id: "mypy", builtIn: true, language: "python"},
		// bandit runs the nix-provided binary directly, so Language is "system"
		// (not "python", which would build a redundant venv).
		{id: "bandit", builtIn: false, nixPackage: "bandit", language: "system"},
	}

	if len(hooks) != len(want) {
		t.Fatalf("PreCommitHooks() returned %d hooks, want %d", len(hooks), len(want))
	}

	for i, hook := range hooks {
		w := want[i]
		if hook.ID != w.id {
			t.Errorf("hooks[%d].ID = %q, want %q", i, hook.ID, w.id)
		}
		if hook.BuiltIn != w.builtIn {
			t.Errorf("hooks[%d].BuiltIn = %v, want %v", i, hook.BuiltIn, w.builtIn)
		}
		if hook.NixPackage != w.nixPackage {
			t.Errorf("hooks[%d].NixPackage = %q, want %q", i, hook.NixPackage, w.nixPackage)
		}
		if hook.Language != w.language {
			t.Errorf("hooks[%d].Language = %q, want %q", i, hook.Language, w.language)
		}
		if len(hook.Types) != 1 || hook.Types[0] != "python" {
			t.Errorf("hooks[%d].Types = %v, want [\"python\"]", i, hook.Types)
		}
		if hook.Name == "" {
			t.Errorf("hooks[%d].Name should not be empty", i)
		}
		if hook.Description == "" {
			t.Errorf("hooks[%d].Description should not be empty", i)
		}
	}
}

// TestPreCommitHooks_ProjectConfig guards W074: mypy only runs for projects
// that configure it, and bandit either reads the project's [tool.bandit] or
// skips low-severity findings such as B101 assert_used in pytest tests.
func TestPreCommitHooks_ProjectConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		extras      map[string]string
		wantIDs     []string
		wantBandit  string
		wantTypeChk []string
	}{
		{
			name:       "unconfigured",
			wantIDs:    []string{"ruff", "bandit"},
			wantBandit: "bandit -ll --skip B101 -r",
		},
		{
			name:        "mypy configured",
			extras:      map[string]string{"mypy": "true"},
			wantIDs:     []string{"ruff", "mypy", "bandit"},
			wantBandit:  "bandit -ll --skip B101 -r",
			wantTypeChk: []string{"mypy ."},
		},
		{
			name:       "bandit configured",
			extras:     map[string]string{"bandit_config": "pyproject.toml"},
			wantIDs:    []string{"ruff", "bandit"},
			wantBandit: "bandit -c pyproject.toml -r",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Extras: tt.extras}
			var ids []string
			for _, h := range (&python.Module{}).PreCommitHooks(cfg) {
				ids = append(ids, h.ID)
				if h.ID == "bandit" && h.Entry != tt.wantBandit {
					t.Errorf("bandit Entry = %q, want %q", h.Entry, tt.wantBandit)
				}
			}
			if !slices.Equal(ids, tt.wantIDs) {
				t.Errorf("hook IDs = %v, want %v", ids, tt.wantIDs)
			}
			if got := (&python.Module{}).VerificationCommands(cfg).TypeCheck; !slices.Equal(got, tt.wantTypeChk) {
				t.Errorf("TypeCheck = %v, want %v", got, tt.wantTypeChk)
			}
		})
	}
}

// --- DevenvYamlInputs tests ---

// TestDevenvYamlInputs locks in the unconditional-return decision: because
// DevenvNixFragment always emits languages.python.version (defaulting to 3.12),
// the nixpkgs-python input must be returned regardless of whether the user
// pinned a version. Guarding on config.Version would re-break unpinned projects.
func TestDevenvYamlInputs(t *testing.T) {
	m := &python.Module{}
	cases := map[string]ecosystem.ModuleConfig{
		"empty config":   {},
		"pinned version": {Version: "3.13"},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			inputs := m.DevenvYamlInputs(cfg)
			if len(inputs) != 1 {
				t.Fatalf("DevenvYamlInputs(%+v) returned %d inputs, want 1", cfg, len(inputs))
			}
			got := inputs[0]
			if got.URL != "github:cachix/nixpkgs-python" {
				t.Errorf("URL = %q, want github:cachix/nixpkgs-python", got.URL)
			}
			if got.Follows != "nixpkgs" {
				t.Errorf("Follows = %q, want nixpkgs", got.Follows)
			}
		})
	}
}

// --- CICommands tests ---

// TestCICommands guards W071: the uv install uses --locked (which checks the
// lockfile and applies the cooldown, unlike --frozen) with a documented
// duration, poetry checks its lockfile first, and the deprecated Safety 2
// `safety check` command is gone.
func TestCICommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  ecosystem.ModuleConfig
		want []string
	}{
		{name: "default", want: []string{"pip install --require-hashes --only-binary :all: -r requirements.txt", "pip-audit"}},
		{name: "pip", cfg: ecosystem.ModuleConfig{PackageManager: "pip"}, want: []string{"pip install --require-hashes --only-binary :all: -r requirements.txt", "pip-audit"}},
		{name: "uv", cfg: ecosystem.ModuleConfig{PackageManager: "uv"}, want: []string{"uv sync --locked --exclude-newer P7D", "pip-audit"}},
		{
			name: "uv with project exclude-newer",
			cfg:  ecosystem.ModuleConfig{PackageManager: "uv", Extras: map[string]string{"uv_exclude_newer": "project"}},
			want: []string{"uv sync --locked", "pip-audit"},
		},
		{name: "poetry", cfg: ecosystem.ModuleConfig{PackageManager: "poetry"}, want: []string{"poetry check --lock", "poetry install --no-interaction", "pip-audit"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, c := range (&python.Module{}).CICommands(tt.cfg) {
				got = append(got, c.Command)
				wantPhase := ecosystem.CIPhaseInstall
				if c.Command == "pip-audit" {
					wantPhase = ecosystem.CIPhaseScan
				}
				if c.Phase != wantPhase {
					t.Errorf("%q Phase = %v, want %v", c.Command, c.Phase, wantPhase)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CICommands() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := &python.Module{}
	pms := m.PackageManagers()

	if len(pms) != 3 {
		t.Fatalf("PackageManagers() returned %d entries, want 3", len(pms))
	}

	expectedNames := []string{"pip", "uv", "poetry"}
	for i, pm := range pms {
		if pm.Name != expectedNames[i] {
			t.Errorf("pms[%d].Name = %q, want %q", i, pm.Name, expectedNames[i])
		}
		if pm.LockFile == "" {
			t.Errorf("pms[%d].LockFile should not be empty", i)
		}
	}
}

func TestPackageManagers_LockFiles(t *testing.T) {
	m := &python.Module{}
	pms := m.PackageManagers()

	expectedLockFiles := map[string]string{
		"pip":    "requirements.txt",
		"uv":     "uv.lock",
		"poetry": "poetry.lock",
	}

	for _, pm := range pms {
		expected, ok := expectedLockFiles[pm.Name]
		if !ok {
			t.Errorf("unexpected package manager %q", pm.Name)
			continue
		}
		if pm.LockFile != expected {
			t.Errorf("pms[%s].LockFile = %q, want %q", pm.Name, pm.LockFile, expected)
		}
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := &python.Module{}
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("WizardFields() returned %d fields, want 2", len(fields))
	}

	// First field: Python version input
	if fields[0].Key != types.SettingVersion || fields[0].Type != ecosystem.FieldTypeInput {
		t.Errorf("fields[0] = %q (%v), want the %q input", fields[0].Key, fields[0].Type, types.SettingVersion)
	}

	// Second field: package manager select
	pmField := fields[1]
	if pmField.Key != types.SettingPackageManager {
		t.Errorf("fields[1].Key = %q, want %q", pmField.Key, types.SettingPackageManager)
	}
	if pmField.Type != ecosystem.FieldTypeSelect {
		t.Errorf("fields[1].Type = %v, want FieldTypeSelect", pmField.Type)
	}
	if len(pmField.Options) != 3 {
		t.Fatalf("fields[1].Options has %d entries, want 3", len(pmField.Options))
	}
	optionValues := make([]string, len(pmField.Options))
	for i, opt := range pmField.Options {
		optionValues[i] = opt.Value
	}
	expectedOptions := []string{"pip", "uv", "poetry"}
	for i, expected := range expectedOptions {
		if optionValues[i] != expected {
			t.Errorf("fields[1].Options[%d].Value = %q, want %q", i, optionValues[i], expected)
		}
	}
}

// --- Registration test ---

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("python")
	if !ok {
		t.Fatal("expected module 'python' to be registered in DefaultRegistry")
	}
	if mod.Name() != "python" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "python")
	}
}

// --- Test helpers ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected string to contain %q\ngot:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected string NOT to contain %q\ngot:\n%s", substr, s)
	}
}

func assertEvidenceContains(t *testing.T, evidence []string, substr string) {
	t.Helper()
	for _, e := range evidence {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("evidence %v should contain an entry mentioning %q", evidence, substr)
}
