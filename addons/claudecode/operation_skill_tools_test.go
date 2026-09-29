package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestOperationSkillsRegisteredFromManifest is the regression guard for the
// operation-skill list being defined twice: every manifest skill must be an
// always-on registry tool, so after the enabled-tool merge every skill is
// both deployed and advertised in CLAUDE.md (qsdev-add-dep used to be
// advertised but never deployed).
func TestOperationSkillsRegisteredFromManifest(t *testing.T) {
	names := claudecode.AvailableQsdevOpsSkillNames()
	if len(names) == 0 {
		t.Fatal("no operation skills in manifest")
	}
	reg := toolreg.DefaultRegistry()

	for _, name := range names {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("manifest skill %q is not a registered tool", name)
			continue
		}
		if tool.Default != toolreg.AlwaysOn {
			t.Errorf("tool %q Default = %v, want AlwaysOn", name, tool.Default)
		}
	}

	// Operation skills are Full-tier artifacts (claudecode-generators gates
	// both deployment and the CLAUDE.md listing on it).
	answers := types.WizardAnswers{Tier: "full", EnabledTools: map[string]bool{}}
	toolreg.MergeInferredTools(&answers, reg)

	files, err := claudecode.ExportDeployOperationSkills(answers)
	if err != nil {
		t.Fatalf("deployOperationSkills: %v", err)
	}
	deployed := make(map[string]bool)
	for _, f := range files {
		deployed[f.Path] = true
	}
	advertised := make(map[string]bool)
	for _, s := range claudecode.BuildClaudeMdData(answers, ecosystem.DefaultRegistry()).AvailableSkills {
		advertised[s.Name] = true
	}
	for _, name := range names {
		if !deployed[".claude/skills/"+name+"/SKILL.md"] {
			t.Errorf("operation skill %q not deployed after MergeInferredTools", name)
		}
		if !advertised["/"+name] {
			t.Errorf("operation skill %q not advertised in CLAUDE.md", name)
		}
	}
}

// TestClaudeMdOmitsDisabledOperationSkills checks CLAUDE.md only advertises
// the operation skills that deployOperationSkills actually writes.
func TestClaudeMdOmitsDisabledOperationSkills(t *testing.T) {
	answers := types.WizardAnswers{Tier: "full", EnabledTools: map[string]bool{"qsdev-doctor": true}}
	data := claudecode.BuildClaudeMdData(answers, ecosystem.DefaultRegistry())

	found := false
	for _, s := range data.AvailableSkills {
		if !strings.HasPrefix(s.Name, "/qsdev-") {
			continue
		}
		if s.Name == "/qsdev-doctor" {
			found = true
			continue
		}
		t.Errorf("CLAUDE.md advertises %s, which is disabled and not deployed", s.Name)
	}
	if !found {
		t.Error("CLAUDE.md does not advertise the enabled /qsdev-doctor skill")
	}
}
