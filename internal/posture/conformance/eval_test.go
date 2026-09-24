package conformance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

func makeReport() *posture.PostureReport {
	return &posture.PostureReport{
		Score: posture.AggregateScore{
			Total:     85.5,
			Grade:     "B",
			Defense:   90.0,
			Config:    80.0,
			DepHealth: new(82.0),
		},
		Defense: posture.DefenseCoverage{
			Layers: []posture.DefenseLayer{
				{Name: "pretooluse-hooks", Weight: posture.WeightCritical, Status: posture.LayerEnabled},
				{Name: "age-gating", Weight: posture.WeightHigh, Status: posture.LayerEnabled},
				{Name: "install-script-blocking", Weight: posture.WeightHigh, Status: posture.LayerEnabled},
				{Name: "lock-file-enforcement", Weight: posture.WeightHigh, Status: posture.LayerEnabled},
				{Name: "vulnerability-scanning", Weight: posture.WeightHigh, Status: posture.LayerEnabled},
				{Name: "nix-hardening", Weight: posture.WeightMedium, Status: posture.LayerDisabled},
				{Name: "sast", Weight: posture.WeightMedium, Status: posture.LayerPartial, Score: 5},
				{Name: "secrets-scanning", Weight: posture.WeightMedium, Status: posture.LayerEnabled},
				{Name: "container-security", Weight: posture.WeightMedium, Status: posture.LayerNotApplicable},
				{Name: "license-compliance", Weight: posture.WeightLow, Status: posture.LayerDisabled},
			},
			Score: 90.0,
		},
		Config: posture.ConfigHealth{Score: 80.0},
		Dependencies: posture.DependencyHealth{
			Totals: posture.VulnSeverityCounts{
				Critical: 0,
				High:     2,
				Moderate: 5,
				Low:      10,
				Info:     3,
			},
			Score:   new(82.0),
			Scanned: true,
		},
		Tools: []posture.ToolStatus{
			{Name: "attach-guard", Enabled: true},
			{Name: "semgrep", Enabled: true},
			{Name: "gitleaks", Enabled: true},
		},
	}
}

func TestEvalCheckExpression_DefenseStatus(t *testing.T) {
	report := makeReport()

	tests := []struct {
		expr string
		want bool
	}{
		{"defense.pretooluse-hooks.status == enabled", true},
		{"defense.pretooluse-hooks.status == disabled", false},
		{"defense.nix-hardening.status == disabled", true},
		{"defense.nix-hardening.status == enabled", false},
		{"defense.sast.status == partial", true},
		{"defense.container-security.status == not-applicable", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalCheckExpression(tt.expr, report)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalCheckExpression_DependenciesTotals(t *testing.T) {
	report := makeReport()

	tests := []struct {
		expr string
		want bool
	}{
		{"dependencies.totals.critical == 0", true},
		{"dependencies.totals.critical == 1", false},
		{"dependencies.totals.high == 2", true},
		{"dependencies.totals.high <= 5", true},
		{"dependencies.totals.high <= 1", false},
		{"dependencies.totals.moderate == 5", true},
		{"dependencies.totals.low == 10", true},
		{"dependencies.totals.info == 3", true},
		{"dependencies.totals.info <= 2", false},
		{"dependencies.totals.unknown == 0", true},
		{"dependencies.totals.low >= 10", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalCheckExpression(tt.expr, report)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalCheckExpression_ConfigScore(t *testing.T) {
	report := makeReport()

	tests := []struct {
		expr string
		want bool
	}{
		{"config.score >= 80.0", true},
		{"config.score >= 80", true},
		{"config.score >= 90.0", false},
		{"config.score >= 79.9", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalCheckExpression(tt.expr, report)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalCheckExpression_ScoreTotal(t *testing.T) {
	report := makeReport()

	tests := []struct {
		expr string
		want bool
	}{
		{"score.total >= 85.0", true},
		{"score.total >= 85.5", true},
		{"score.total >= 86.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalCheckExpression(tt.expr, report)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalCheckExpression_ToolsEnabled(t *testing.T) {
	report := makeReport()

	tests := []struct {
		expr string
		want bool
	}{
		{"tools.attach-guard.enabled == true", true},
		{"tools.attach-guard.enabled == false", false},
		{"tools.semgrep.enabled == true", true},
		{"tools.license-compliance.enabled == false", true},
		{"tools.license-compliance.enabled == true", false},
		{"tools.nonexistent.enabled == false", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalCheckExpression(tt.expr, report)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalCheckExpression_Errors(t *testing.T) {
	report := makeReport()

	tests := []struct {
		name string
		expr string
	}{
		{"empty", ""},
		{"no operator", "defense.pretooluse-hooks.status enabled"},
		{"unknown domain", "unknown.field == 1"},
		{"bad defense path", "defense.pretooluse-hooks == enabled"},
		{"unknown layer", "defense.nonexistent-layer.status == enabled"},
		{"bad dep severity", "dependencies.totals.bogus == 0"},
		{"bad dep value", "dependencies.totals.critical == abc"},
		{"bad config path", "config.unknown >= 80"},
		{"bad config value", "config.score >= abc"},
		{"bad score path", "score.unknown >= 80"},
		{"bad score value", "score.total >= abc"},
		{"bad tools path", "tools.semgrep == true"},
		{"bad tools value", "tools.semgrep.enabled == maybe"},
		{"defense wrong op", "defense.sast.status >= enabled"},
		{"tools wrong op", "tools.semgrep.enabled >= true"},
		{"bad dep path", "dependencies.unknown.critical == 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EvalCheckExpression(tt.expr, report)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestEvalCheckExpression_Whitespace(t *testing.T) {
	report := makeReport()

	// Extra whitespace should be handled
	got, err := EvalCheckExpression("  defense.pretooluse-hooks.status  ==  enabled  ", report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true with whitespace")
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	pf, err := LoadFile("/nonexistent/path/.qsdev-policy.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pf != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestLoadFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev-policy.yaml")

	content := `conformance:
  custom:
    name: "strict"
    requirements:
      - name: "no critical vulns"
        check: "dependencies.totals.critical == 0"
      - name: "sast enabled"
        check: "tools.semgrep.enabled == true"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	pf, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pf == nil {
		t.Fatal("expected non-nil policy file")
		return
	}
	if pf.Conformance.Custom == nil {
		t.Fatal("expected custom conformance")
		return
	}
	if pf.Conformance.Custom.Name != "strict" {
		t.Errorf("name: got %q, want %q", pf.Conformance.Custom.Name, "strict")
	}
	if len(pf.Conformance.Custom.Requirements) != 2 {
		t.Errorf("requirements count: got %d, want 2", len(pf.Conformance.Custom.Requirements))
	}
}

func TestLoadFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev-policy.yaml")

	if err := os.WriteFile(path, []byte("conformance:\n  custom:\n    - :\n  ][broken"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev-policy.yaml")

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	pf, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pf == nil {
		t.Fatal("expected non-nil policy file")
		return
	}
	if pf.Conformance.Custom != nil {
		t.Error("expected nil custom conformance for empty file")
	}
}

func TestEvalCheckExpression_IntegerComparisons(t *testing.T) {
	report := makeReport()

	// Test >= for integers
	got, err := EvalCheckExpression("dependencies.totals.high >= 2", report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected high >= 2 to be true")
	}

	got, err = EvalCheckExpression("dependencies.totals.high >= 3", report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected high >= 3 to be false")
	}
}

// TestEvalCheckExpression_DependenciesInconclusive is the regression test for
// "dependencies.totals.critical == 0" passing on a report whose totals are
// zero only because no conclusive scan ran.
func TestEvalCheckExpression_DependenciesInconclusive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		deps posture.DependencyHealth
	}{
		{"not scanned", posture.DependencyHealth{}},
		{"scan failed", posture.DependencyHealth{ScanFailed: true}},
		{"unresolved severity", posture.DependencyHealth{
			Scanned: true,
			Totals:  posture.VulnSeverityCounts{Unknown: 1},
		}},
	}
	exprs := []string{
		"dependencies.totals.critical == 0",
		"dependencies.totals.high <= 5",
		"dependencies.totals.unknown <= 5",
	}
	for _, tt := range tests {
		for _, expr := range exprs {
			t.Run(tt.name+"/"+expr, func(t *testing.T) {
				t.Parallel()
				report := makeReport()
				report.Dependencies = tt.deps
				got, err := EvalCheckExpression(expr, report)
				if got {
					t.Error("inconclusive dependency results passed")
				}
				if !errors.Is(err, ErrDependenciesInconclusive) {
					t.Errorf("err = %v, want ErrDependenciesInconclusive", err)
				}
			})
		}
	}
}

// TestEvalCheckExpression_InvalidBeforeInconclusive: a malformed dependencies
// expression reports the syntax error, not the scan state, so a typo is
// visible even on an unscanned report.
func TestEvalCheckExpression_InvalidBeforeInconclusive(t *testing.T) {
	t.Parallel()
	report := makeReport()
	report.Dependencies = posture.DependencyHealth{}
	for _, expr := range []string{
		"dependencies.totals.bogus == 0",
		"dependencies.totals.critical == abc",
	} {
		_, err := EvalCheckExpression(expr, report)
		if err == nil || errors.Is(err, ErrDependenciesInconclusive) {
			t.Errorf("%s: err = %v, want a syntax error", expr, err)
		}
	}
}

func TestLoadFile_Invalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "misspelt key",
			content: "conformance:\n  custom:\n    name: x\n    requirement:\n" +
				"      - name: a\n        check: score.total >= 1\n",
			wantErr: "requirement",
		},
		{
			name:    "no requirements",
			content: "conformance:\n  custom:\n    name: x\n",
			wantErr: "no requirements",
		},
		{
			name:    "requirement without name",
			content: "conformance:\n  custom:\n    requirements:\n      - check: score.total >= 1\n",
			wantErr: "has no name",
		},
		{
			name:    "requirement without check",
			content: "conformance:\n  custom:\n    requirements:\n      - name: a\n",
			wantErr: "no check expression",
		},
		{
			name: "duplicate names",
			content: "conformance:\n  custom:\n    requirements:\n" +
				"      - name: a\n        check: score.total >= 1\n" +
				"      - name: a\n        check: config.score >= 1\n",
			wantErr: "duplicate requirement name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), ".qsdev-policy.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadFile(path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadFile error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
