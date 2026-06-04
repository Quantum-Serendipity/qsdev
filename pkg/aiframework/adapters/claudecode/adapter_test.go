package claudecode_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	claudecodeaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	claudecodeadapter "github.com/Quantum-Serendipity/qsdev/pkg/aiframework/adapters/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework/contracttest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func newTestAdapter() *claudecodeadapter.Adapter {
	cfg := claudecodeaddon.Config{
		DefaultPermissions: claudecodeaddon.PermissionPresetStandard,
	}
	return claudecodeadapter.New(cfg, nil)
}

func TestDetect_PresentRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := newTestAdapter()
	det, err := a.Detect(root)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if !det.Detected {
		t.Fatal("expected Detected=true with .claude/ and CLAUDE.md present")
	}
	if det.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", det.Confidence)
	}
	if len(det.Evidence) < 2 {
		t.Errorf("expected at least 2 evidence entries, got %d", len(det.Evidence))
	}
	if len(det.ConfigPaths) < 2 {
		t.Errorf("expected at least 2 config paths, got %d", len(det.ConfigPaths))
	}
}

func TestDetect_AbsentRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	a := newTestAdapter()
	det, err := a.Detect(root)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if det.Detected {
		t.Fatal("expected Detected=false for empty directory")
	}
}

func TestRender_ProducesSettingsJSON(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	input := &aiframework.PolicyInput{
		ProjectRoot: t.TempDir(),
		Permissions: &aiframework.PermissionPolicy{
			Preset: "standard",
		},
	}

	files, err := a.Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one generated file")
	}

	found := false
	for _, f := range files {
		if filepath.Base(f.Path) == "settings.json" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a file with path ending in settings.json, got paths: %v", filePaths(files))
	}
}

func TestValidate_ValidJSON(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	files := []types.GeneratedFile{
		{Path: "test.json", Content: []byte(`{"key": "value"}`)},
	}

	issues := a.Validate(context.Background(), files)
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid JSON, got %d: %v", len(issues), issues)
	}
}

func TestValidate_InvalidJSON(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	files := []types.GeneratedFile{
		{Path: "broken.json", Content: []byte(`{invalid`)},
	}

	issues := a.Validate(context.Background(), files)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for invalid JSON, got %d", len(issues))
	}
	if issues[0].Severity != aiframework.SeverityError {
		t.Errorf("expected SeverityError, got %v", issues[0].Severity)
	}
}

func TestDeploy_PackageGuard(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	hooks := []aiframework.HookPolicy{
		{
			Event: aiframework.EventPreToolUse,
			Logic: aiframework.LogicPackageGuard,
		},
	}

	files, err := a.Deploy(context.Background(), hooks)
	if err != nil {
		t.Fatalf("Deploy returned error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one hook file for package guard")
	}

	found := false
	for _, f := range files {
		if filepath.Base(f.Path) == "package-guard.py" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected package-guard.py in generated files, got: %v", filePaths(files))
	}
}

func TestFilterServers_StdioOnly(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	servers := []aiframework.MCPServerSpec{
		{Name: "stdio-server", Transport: aiframework.TransportStdio},
		{Name: "http-server", Transport: aiframework.TransportStreamableHTTP},
		{Name: "sse-server", Transport: aiframework.TransportSSE},
	}

	filtered := a.FilterServers(servers)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered server, got %d", len(filtered))
	}
	if filtered[0].Name != "stdio-server" {
		t.Errorf("expected stdio-server, got %s", filtered[0].Name)
	}
}

func TestEnforcementTier_IsHook(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	if tier := a.EnforcementTier(); tier != aiframework.TierHook {
		t.Errorf("expected TierHook, got %v", tier)
	}
}

func TestReportGaps_DenyRules(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()
	policy := &aiframework.PermissionPolicy{
		DenyRules: []aiframework.PermissionRule{
			{Pattern: "Bash(rm -rf /)", Reason: "prevent system wipe"},
			{Pattern: "Bash(dd if=*)", Reason: "prevent disk overwrite"},
		},
	}

	gaps := a.ReportGaps(context.Background(), policy)
	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(gaps))
	}
	for _, g := range gaps {
		if g.RequiredTier != aiframework.TierKernel {
			t.Errorf("expected RequiredTier=TierKernel, got %v", g.RequiredTier)
		}
		if g.ActualTier != aiframework.TierHook {
			t.Errorf("expected ActualTier=TierHook, got %v", g.ActualTier)
		}
	}
}

func TestContractSuite(t *testing.T) {
	t.Parallel()

	a := newTestAdapter()

	presentRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(presentRoot, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(presentRoot, "CLAUDE.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	contracttest.RunAllContractTests(t, contracttest.ContractAdapters{
		Detection: a,
		Config:    a,
		Hooks:     a,
		Registry:  a,
		Metrics:   a,
		State:     a,
		Tools:     a,
	}, contracttest.ContractFixtures{
		PresentRoot: presentRoot,
		AbsentRoot:  t.TempDir(),
		PolicyInput: &aiframework.PolicyInput{
			ProjectRoot: t.TempDir(),
			Permissions: &aiframework.PermissionPolicy{
				Preset:    "standard",
				DenyRules: []aiframework.PermissionRule{{Pattern: "Bash(rm -rf *)", Reason: "destructive"}},
			},
		},
	})
}

func filePaths(files []types.GeneratedFile) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	return paths
}
