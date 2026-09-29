package ecosystem

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestResolveLanguageModules(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	goMod := &MockModule{NameVal: "go", ManifestFilesVal: []ManifestFileInfo{{Path: "go.mod"}}}
	jsMod := &MockModule{NameVal: "javascript", ManifestFilesVal: []ManifestFileInfo{{Path: "package.json", VSSupported: true}}}
	for _, m := range []EcosystemModule{goMod, jsMod} {
		if err := reg.Register(m); err != nil {
			t.Fatal(err)
		}
	}
	langs := []types.LanguageChoice{
		{Name: "javascript", Version: "22"},
		{Name: "unknown"},
		{Name: "go", Version: "1.25"},
	}

	modules, configFor := ResolveLanguageModules(langs, reg)
	if len(modules) != 2 || modules[0].Name() != "javascript" || modules[1].Name() != "go" {
		t.Fatalf("modules = %v, want [javascript go] in language order, unknown skipped", modules)
	}
	for mod, want := range map[EcosystemModule]string{goMod: "1.25", jsMod: "22"} {
		if got := configFor(mod).Version; got != want {
			t.Errorf("configFor(%s).Version = %q, want %q", mod.Name(), got, want)
		}
	}

	report := LanguageManifestCoverage(langs, reg)
	if len(report.Covered) != 1 || report.Covered[0].Path != "package.json" {
		t.Errorf("Covered = %v, want [package.json]", report.Covered)
	}
	if len(report.Uncovered) != 1 || report.Uncovered[0].Path != "go.mod" {
		t.Errorf("Uncovered = %v, want [go.mod]", report.Uncovered)
	}
}
