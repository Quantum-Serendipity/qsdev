package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// breakProjectCatalog points the catalog at a project whose defaults file the
// catalog rejects, and returns the load error. The previous catalog state is
// restored when the test ends.
func breakProjectCatalog(t *testing.T) (root string, loadErr error) {
	t.Helper()
	root = t.TempDir()
	path := catalog.ProjectConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tier_to_compliance:\n  standard: nonexistent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevRoot := catalog.ProjectRoot()
	t.Cleanup(func() {
		catalog.ResetDefault() // also clears the project root
		if err := catalog.SetProjectRoot(prevRoot); err != nil {
			t.Errorf("restoring the catalog project root: %v", err)
		}
	})
	catalog.ResetDefault()
	if err := catalog.SetProjectRoot(root); err != nil {
		t.Fatal(err)
	}
	_, loadErr = catalog.Default()
	if loadErr == nil {
		t.Fatal("test setup: the project defaults file should break the catalog")
	}
	return root, loadErr
}

// TestRunCatalogUnavailable pins U08-WS3 and GC-WS2: when the catalog cannot
// load, check reports the failure as a critical config_integrity/config_catalog
// result and still runs every check that needs no catalog, without a panic.
func TestRunCatalogUnavailable(t *testing.T) {
	root, loadErr := breakProjectCatalog(t)

	tests := []struct {
		name string
		cfg  *types.QsdevConfig
		// wantBinaryCompat is the binary_compatibility result name expected.
		wantBinaryCompat string
	}{
		{name: "no config", cfg: nil, wantBinaryCompat: branding.Get().AppName + "_version_constraint"},
		{name: "valid config", cfg: &types.QsdevConfig{Version: types.ConfigVersionMax}, wantBinaryCompat: "config_schema_version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := CheckContext{
				ProjectRoot:   root,
				BinaryVersion: "1.0.0",
				QsdevConfig:   tt.cfg,
				StateFile:     filepath.Join(root, "missing-state.json"),
			}
			report := RunCatalogUnavailable(ctx, CatalogLoadFailure(loadErr))

			got := findCategoryResult(report.Checks, CategoryConfigIntegrity, "config_catalog")
			if got == nil {
				t.Fatalf("no config_integrity/config_catalog result in %+v", report.Checks)
			}
			if got.Status != StatusFail || got.Severity != SeverityCritical {
				t.Errorf("config_catalog = %s/%s, want fail/critical", got.Status, got.Severity)
			}
			if want := catalog.LoadError(loadErr).Error(); got.Message != want {
				t.Errorf("message = %q, want %q", got.Message, want)
			}
			if !strings.Contains(got.Remediation, "defaults validate") {
				t.Errorf("remediation %q does not name 'defaults validate'", got.Remediation)
			}
			if !ShouldFail(report.Checks, AuditLevelCritical) {
				t.Error("the catalog failure does not fail check at audit level critical")
			}

			for _, want := range []struct {
				cat  CheckCategory
				name string
			}{
				{CategoryBinaryCompat, tt.wantBinaryCompat},
				{CategoryConfigIntegrity, "org_overlay_pinned"},
				{CategoryFileState, "generated_files"},
			} {
				if findCategoryResult(report.Checks, want.cat, want.name) == nil {
					t.Errorf("catalog-free check %s/%s did not report", want.cat, want.name)
				}
			}
			if report.Summary.Total != len(report.Checks) || report.Summary.Fail == 0 {
				t.Errorf("summary %+v does not tally %d checks", report.Summary, len(report.Checks))
			}

			var buf bytes.Buffer
			if err := FormatReport(report, FormatJSON, &buf, false); err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				Checks []CheckResult `json:"checks"`
			}
			if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
				t.Fatalf("JSON report does not decode: %v\n%s", err, buf.String())
			}
			if r := findCategoryResult(decoded.Checks, CategoryConfigIntegrity, "config_catalog"); r == nil || r.Status != StatusFail {
				t.Errorf("JSON .checks lacks the failing config_catalog result:\n%s", buf.String())
			}
		})
	}
}

// TestRunAllChecks_NoCatalogFailureRow pins that a run with a loaded catalog
// reports no config_catalog result: the row exists only for a failure.
func TestRunAllChecks_NoCatalogFailureRow(t *testing.T) {
	t.Parallel()
	report := RunAllChecks(CheckContext{ProjectRoot: t.TempDir()})
	if r := findCategoryResult(report.Checks, CategoryConfigIntegrity, "config_catalog"); r != nil {
		t.Errorf("RunAllChecks reported %+v", *r)
	}
}

// TestRunCatalogUnavailable_LeadsWithFailure pins that the given failure
// (here a registry build failure) is reported as is, first.
func TestRunCatalogUnavailable_LeadsWithFailure(t *testing.T) {
	t.Parallel()
	failure := ToolRegistryFailure(errors.New("boom"))
	report := RunCatalogUnavailable(CheckContext{ProjectRoot: t.TempDir()}, failure)
	if len(report.Checks) == 0 || report.Checks[0].Name != failure.Name || report.Checks[0].Message != failure.Message {
		t.Errorf("first result = %+v, want %+v", report.Checks, failure)
	}
}

func findCategoryResult(results []CheckResult, cat CheckCategory, name string) *CheckResult {
	for i := range results {
		if results[i].Category == cat && results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}
