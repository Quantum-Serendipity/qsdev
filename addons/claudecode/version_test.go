package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
)

func TestComputeTemplateVersion_Deterministic(t *testing.T) {
	v1 := claudecode.ExportComputeTemplateVersion()
	v2 := claudecode.ExportComputeTemplateVersion()

	if v1 != v2 {
		t.Errorf("expected deterministic output, got %q and %q", v1, v2)
	}
}

func TestComputeTemplateVersion_HasSha256Prefix(t *testing.T) {
	v := claudecode.ExportComputeTemplateVersion()

	if !strings.HasPrefix(v, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", v)
	}

	// sha256: prefix + 64 hex chars = 71 total.
	if len(v) != 71 {
		t.Errorf("expected length 71, got %d for %q", len(v), v)
	}
}

func TestComputeSkillLibraryVersion_Deterministic(t *testing.T) {
	v1 := claudecode.ExportComputeSkillLibraryVersion()
	v2 := claudecode.ExportComputeSkillLibraryVersion()

	if v1 != v2 {
		t.Errorf("expected deterministic output, got %q and %q", v1, v2)
	}
}

func TestComputeSkillLibraryVersion_DiffersFromTemplate(t *testing.T) {
	tv := claudecode.ExportComputeTemplateVersion()
	sv := claudecode.ExportComputeSkillLibraryVersion()

	if tv == sv {
		t.Errorf("template version and skill library version should differ, both are %q", tv)
	}
}
