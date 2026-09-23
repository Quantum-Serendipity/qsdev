package claudecode_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

func TestCompareVersions_NoChange(t *testing.T) {
	tv := claudecode.ExportComputeTemplateVersion()
	sv := claudecode.ExportComputeSkillLibraryVersion()

	diff := claudecode.ExportCompareVersions(tv, sv)

	if diff.NeedsUpdate() {
		t.Error("expected NeedsUpdate() == false when versions match")
	}
	if diff.TemplateChanged {
		t.Error("expected TemplateChanged == false")
	}
	if diff.SkillLibraryChanged {
		t.Error("expected SkillLibraryChanged == false")
	}
}

func TestCompareVersions_TemplateChanged(t *testing.T) {
	sv := claudecode.ExportComputeSkillLibraryVersion()

	diff := claudecode.ExportCompareVersions("sha256:0000000000000000000000000000000000000000000000000000000000000000", sv)

	if !diff.TemplateChanged {
		t.Error("expected TemplateChanged == true")
	}
	if diff.SkillLibraryChanged {
		t.Error("expected SkillLibraryChanged == false")
	}
	if !diff.NeedsUpdate() {
		t.Error("expected NeedsUpdate() == true")
	}
}

func TestCompareVersions_SkillLibraryChanged(t *testing.T) {
	tv := claudecode.ExportComputeTemplateVersion()

	diff := claudecode.ExportCompareVersions(tv, "sha256:0000000000000000000000000000000000000000000000000000000000000000")

	if diff.TemplateChanged {
		t.Error("expected TemplateChanged == false")
	}
	if !diff.SkillLibraryChanged {
		t.Error("expected SkillLibraryChanged == true")
	}
	if !diff.NeedsUpdate() {
		t.Error("expected NeedsUpdate() == true")
	}
}

func TestCompareVersions_EmptyStored(t *testing.T) {
	diff := claudecode.ExportCompareVersions("", "")

	if !diff.TemplateChanged {
		t.Error("expected TemplateChanged == true for empty stored version")
	}
	if !diff.SkillLibraryChanged {
		t.Error("expected SkillLibraryChanged == true for empty stored version")
	}
	if !diff.NeedsUpdate() {
		t.Error("expected NeedsUpdate() == true for empty stored versions")
	}
}

// TestTemplateFS_OnlyShippedFiles verifies the embedded template tree holds no
// build-tree artifacts: dot/underscore entries (.gitkeep, __pycache__) and
// Python bytecode would make the binary and its template-version hash depend
// on whether tests ran before the build.
func TestTemplateFS_OnlyShippedFiles(t *testing.T) {
	t.Parallel()
	err := fs.WalkDir(claudecode.ExportTemplateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		base := d.Name()
		if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") || strings.HasSuffix(base, ".pyc") {
			t.Errorf("embedded template tree contains build artifact %q", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIsTemplateTestFixture(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"templates/hooks/lsp-first-guard_test.sh": true,
		"templates/hooks/lsp-first-guard.sh":      false,
		"templates/hooks/package-guard.py":        false,
		"templates/claude-md.tmpl":                false,
	}
	for path, want := range cases {
		if got := claudecode.ExportIsTemplateTestFixture(path); got != want {
			t.Errorf("isTemplateTestFixture(%q) = %v, want %v", path, got, want)
		}
	}
}
