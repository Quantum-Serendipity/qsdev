package claudecode_test

import (
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
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

func TestIsLibrarySkill_Known(t *testing.T) {
	if !claudecode.ExportIsLibrarySkill("deploy") {
		t.Error("expected IsLibrarySkill(\"deploy\") == true")
	}
}

func TestIsLibrarySkill_Unknown(t *testing.T) {
	if claudecode.ExportIsLibrarySkill("my-custom") {
		t.Error("expected IsLibrarySkill(\"my-custom\") == false")
	}
}

func TestIsLibraryRule_Known(t *testing.T) {
	if !claudecode.ExportIsLibraryRule("go-conventions.md") {
		t.Error("expected IsLibraryRule(\"go-conventions.md\") == true")
	}
}

func TestIsLibraryRule_SecurityRules(t *testing.T) {
	if !claudecode.ExportIsLibraryRule("security-rules.md") {
		t.Error("expected IsLibraryRule(\"security-rules.md\") == true")
	}
}

func TestIsLibraryRule_Unknown(t *testing.T) {
	if claudecode.ExportIsLibraryRule("my-team-style.md") {
		t.Error("expected IsLibraryRule(\"my-team-style.md\") == false")
	}
}
