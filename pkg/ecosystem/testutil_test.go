package ecosystem

import (
	"fmt"
	"go/build"
	"slices"
	"testing"
)

// TestProductionFilesDoNotImportTesting guards against the package's non-test
// files importing "testing", which would link the testing package (and
// register its flags) in every binary that imports pkg/ecosystem.
func TestProductionFilesDoNotImportTesting(t *testing.T) {
	t.Parallel()

	pkg, err := build.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("build.ImportDir: %v", err)
	}
	if slices.Contains(pkg.Imports, "testing") {
		t.Error(`non-test files of pkg/ecosystem import "testing"`)
	}
}

// recordingReporter captures AssertModuleIdentity failures.
type recordingReporter struct{ errs []string }

func (r *recordingReporter) Helper() {}

func (r *recordingReporter) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

func TestAssertModuleIdentity(t *testing.T) {
	t.Parallel()

	m := &MockModule{NameVal: "go", DisplayNameVal: "Go", TierVal: 1}
	tests := []struct {
		name        string
		wantName    string
		wantDisplay string
		wantTier    int
		wantErrs    int
	}{
		{"all match", "go", "Go", 1, 0},
		{"name mismatch", "rust", "Go", 1, 1},
		{"all mismatch", "rust", "Rust", 2, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &recordingReporter{}
			AssertModuleIdentity(r, m, tt.wantName, tt.wantDisplay, tt.wantTier)
			if len(r.errs) != tt.wantErrs {
				t.Errorf("reported %d errors %q, want %d", len(r.errs), r.errs, tt.wantErrs)
			}
		})
	}
}
