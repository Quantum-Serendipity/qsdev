package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestCheckSecurityHardening_LockFilePresent(t *testing.T) {
	dir := t.TempDir()

	// Create go.sum.
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("hash"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		GdevConfig: &types.GdevConfig{
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
		GdevConfig: &types.GdevConfig{
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

func TestCheckSecurityHardening_NpmrcPresent(t *testing.T) {
	dir := t.TempDir()

	// Create lock file and .npmrc.
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".npmrc"), []byte("package-lock=true"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		GdevConfig: &types.GdevConfig{
			Languages: []types.LanguageConfig{
				{Name: "javascript"},
			},
		},
	}

	results := CheckSecurityHardening(ctx)

	var npmrcResult *CheckResult
	for i := range results {
		if results[i].Name == "npmrc_exists" {
			npmrcResult = &results[i]
			break
		}
	}

	if npmrcResult == nil {
		t.Fatal("expected npmrc_exists result")
	}
	if npmrcResult.Status != StatusPass {
		t.Errorf("npmrc_exists.Status = %s, want %s", npmrcResult.Status, StatusPass)
	}
}

func TestCheckSecurityHardening_NpmrcMissing(t *testing.T) {
	dir := t.TempDir()

	// Create lock file but no .npmrc.
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		GdevConfig: &types.GdevConfig{
			Languages: []types.LanguageConfig{
				{Name: "javascript"},
			},
		},
	}

	results := CheckSecurityHardening(ctx)

	var npmrcResult *CheckResult
	for i := range results {
		if results[i].Name == "npmrc_exists" {
			npmrcResult = &results[i]
			break
		}
	}

	if npmrcResult == nil {
		t.Fatal("expected npmrc_exists result")
	}
	if npmrcResult.Status != StatusFail {
		t.Errorf("npmrc_exists.Status = %s, want %s", npmrcResult.Status, StatusFail)
	}
}

func TestCheckSecurityHardening_PythonConfigPresent(t *testing.T) {
	dir := t.TempDir()

	// Create pyproject.toml and lock file.
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "uv.lock"), []byte("locked"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		GdevConfig: &types.GdevConfig{
			Languages: []types.LanguageConfig{
				{Name: "python"},
			},
		},
	}

	results := CheckSecurityHardening(ctx)

	var pyResult *CheckResult
	for i := range results {
		if results[i].Name == "python_config_exists" {
			pyResult = &results[i]
			break
		}
	}

	if pyResult == nil {
		t.Fatal("expected python_config_exists result")
	}
	if pyResult.Status != StatusPass {
		t.Errorf("python_config_exists.Status = %s, want %s", pyResult.Status, StatusPass)
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
		GdevConfig: &types.GdevConfig{},
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
		GdevConfig: &types.GdevConfig{
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
