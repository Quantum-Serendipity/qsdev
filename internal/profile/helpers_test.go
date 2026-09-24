package profile

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func mustWorkflow(t *testing.T, p *InfraProfile) types.GeneratedFile {
	t.Helper()
	f, err := p.generateSecurityScanWorkflow(ProjectInputs{})
	if err != nil {
		t.Fatalf("generateSecurityScanWorkflow(%s): %v", p.Name, err)
	}
	return f
}

func mustSecurityDoc(t *testing.T, p *InfraProfile) types.GeneratedFile {
	t.Helper()
	f, err := p.generateSecurityDoc(ProjectInputs{})
	if err != nil {
		t.Fatalf("generateSecurityDoc(%s): %v", p.Name, err)
	}
	return f
}

func mustConfigFiles(t *testing.T, p *InfraProfile, in ProjectInputs) []types.GeneratedFile {
	t.Helper()
	files, err := p.ConfigFiles(in)
	if err != nil {
		t.Fatalf("ConfigFiles(%s): %v", p.Name, err)
	}
	return files
}

func findFile(files []types.GeneratedFile, path string) (types.GeneratedFile, bool) {
	for _, f := range files {
		if f.Path == path {
			return f, true
		}
	}
	return types.GeneratedFile{}, false
}
