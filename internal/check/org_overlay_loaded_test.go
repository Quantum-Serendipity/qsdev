package check

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckOrgOverlayLoaded(t *testing.T) {
	t.Parallel()
	if got := CheckOrgOverlayLoaded(CheckContext{}); len(got) != 0 {
		t.Errorf("no skipped overlay: got %+v, want no result", got)
	}

	skipped := errors.New("user defaults /o/defaults.yaml: defaults file loosens the built-in security floor")
	got := CheckOrgOverlayLoaded(CheckContext{OrgOverlayErr: skipped})
	if len(got) != 1 {
		t.Fatalf("skipped overlay: got %d results, want 1", len(got))
	}
	r := got[0]
	if r.Name != "config_catalog" || r.Status != StatusFail || r.Severity != SeverityHigh {
		t.Errorf("result %+v, want config_catalog FAIL high", r)
	}
	if !strings.Contains(r.Message, "/o/defaults.yaml") || !strings.Contains(r.Remediation, "defaults validate") {
		t.Errorf("result %+v does not name the file and 'defaults validate'", r)
	}

	report := RunAllChecks(CheckContext{OrgOverlayErr: skipped})
	if report.Summary.Fail < 1 {
		t.Errorf("RunAllChecks does not fail a skipped org overlay: %+v", report.Summary)
	}
}
