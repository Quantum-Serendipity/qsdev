package ecosystem

import (
	"testing"
)

func staticConfig(_ EcosystemModule) ModuleConfig {
	return ModuleConfig{}
}

func TestAggregateVerificationCommands_SingleModule(t *testing.T) {
	mod := &MockModule{
		NameVal: "go",
		VerificationCommandsVal: VerificationCommands{
			Build:  []string{"go build ./..."},
			Test:   []string{"go test ./..."},
			Lint:   []string{"go vet ./...", "golangci-lint run"},
			Format: []string{"gofmt -l ."},
		},
	}

	result := AggregateVerificationCommands([]EcosystemModule{mod}, staticConfig)

	if len(result.Build) != 1 || result.Build[0] != "go build ./..." {
		t.Errorf("Build = %v, want [go build ./...]", result.Build)
	}
	if len(result.Test) != 1 || result.Test[0] != "go test ./..." {
		t.Errorf("Test = %v, want [go test ./...]", result.Test)
	}
	if len(result.Lint) != 2 {
		t.Errorf("Lint = %v, want 2 entries", result.Lint)
	}
	if len(result.Format) != 1 {
		t.Errorf("Format = %v, want 1 entry", result.Format)
	}
}

func TestAggregateVerificationCommands_MultiModule(t *testing.T) {
	goMod := &MockModule{
		NameVal: "go",
		VerificationCommandsVal: VerificationCommands{
			Build: []string{"go build ./..."},
			Test:  []string{"go test ./..."},
			Lint:  []string{"golangci-lint run"},
		},
	}
	jsMod := &MockModule{
		NameVal: "javascript",
		VerificationCommandsVal: VerificationCommands{
			Build:  []string{"npm run build"},
			Test:   []string{"npm test"},
			Lint:   []string{"npm run lint"},
			Format: []string{"prettier --check ."},
		},
	}

	result := AggregateVerificationCommands([]EcosystemModule{goMod, jsMod}, staticConfig)

	if len(result.Build) != 2 {
		t.Errorf("Build = %v, want 2 entries", result.Build)
	}
	if len(result.Test) != 2 {
		t.Errorf("Test = %v, want 2 entries", result.Test)
	}
	if len(result.Lint) != 2 {
		t.Errorf("Lint = %v, want 2 entries", result.Lint)
	}
}

func TestAggregateVerificationCommands_Dedup(t *testing.T) {
	mod1 := &MockModule{
		NameVal:                 "a",
		VerificationCommandsVal: VerificationCommands{Test: []string{"test-cmd"}},
	}
	mod2 := &MockModule{
		NameVal:                 "b",
		VerificationCommandsVal: VerificationCommands{Test: []string{"test-cmd"}},
	}

	result := AggregateVerificationCommands([]EcosystemModule{mod1, mod2}, staticConfig)

	if len(result.Test) != 1 {
		t.Errorf("Test = %v, want 1 entry (deduped)", result.Test)
	}
}

func TestAggregateVerificationCommands_Empty(t *testing.T) {
	result := AggregateVerificationCommands(nil, staticConfig)

	if !result.IsEmpty() {
		t.Error("expected empty result for nil modules")
	}
}

func TestAggregateManifestCoverage_Partitioning(t *testing.T) {
	goMod := &MockModule{
		NameVal: "go",
		ManifestFilesVal: []ManifestFileInfo{
			{Path: "go.mod", Ecosystem: "go", VSSupported: false, LockFile: "go.sum"},
		},
	}
	jsMod := &MockModule{
		NameVal: "javascript",
		ManifestFilesVal: []ManifestFileInfo{
			{Path: "package.json", Ecosystem: "npm", VSSupported: true, LockFile: "package-lock.json"},
		},
	}

	report := AggregateManifestCoverage([]EcosystemModule{goMod, jsMod}, staticConfig)

	if len(report.AllManifests) != 2 {
		t.Errorf("AllManifests = %d, want 2", len(report.AllManifests))
	}
	if len(report.Covered) != 1 {
		t.Errorf("Covered = %d, want 1", len(report.Covered))
	}
	if report.Covered[0].Path != "package.json" {
		t.Errorf("Covered[0].Path = %q, want package.json", report.Covered[0].Path)
	}
	if len(report.Uncovered) != 1 {
		t.Errorf("Uncovered = %d, want 1", len(report.Uncovered))
	}
	if report.Uncovered[0].Path != "go.mod" {
		t.Errorf("Uncovered[0].Path = %q, want go.mod", report.Uncovered[0].Path)
	}
	if !report.HasUncovered() {
		t.Error("HasUncovered should be true")
	}
}

func TestAggregateManifestCoverage_AllCovered(t *testing.T) {
	mod := &MockModule{
		NameVal: "python",
		ManifestFilesVal: []ManifestFileInfo{
			{Path: "requirements.txt", Ecosystem: "pip", VSSupported: true},
		},
	}

	report := AggregateManifestCoverage([]EcosystemModule{mod}, staticConfig)

	if report.HasUncovered() {
		t.Error("HasUncovered should be false when all manifests are covered")
	}
	if len(report.Covered) != 1 {
		t.Errorf("Covered = %d, want 1", len(report.Covered))
	}
}

func TestAggregateManifestCoverage_Empty(t *testing.T) {
	report := AggregateManifestCoverage(nil, staticConfig)

	if len(report.AllManifests) != 0 {
		t.Errorf("AllManifests = %d, want 0", len(report.AllManifests))
	}
	if report.HasUncovered() {
		t.Error("HasUncovered should be false for empty report")
	}
}

func TestVerificationCommands_All(t *testing.T) {
	vc := VerificationCommands{
		Build:     []string{"build"},
		Test:      []string{"test1", "test2"},
		Lint:      []string{"lint"},
		TypeCheck: []string{"typecheck"},
		Format:    []string{"format"},
	}

	all := vc.All()
	if len(all) != 6 {
		t.Errorf("All() = %v, want 6 entries", all)
	}
	if all[0] != "build" || all[1] != "test1" || all[5] != "format" {
		t.Errorf("All() order = %v, want build, test1, test2, lint, typecheck, format", all)
	}
}

func TestVerificationCommands_IsEmpty(t *testing.T) {
	empty := VerificationCommands{}
	if !empty.IsEmpty() {
		t.Error("expected IsEmpty() = true for zero-value")
	}

	nonEmpty := VerificationCommands{Test: []string{"test"}}
	if nonEmpty.IsEmpty() {
		t.Error("expected IsEmpty() = false when Test has entries")
	}
}
