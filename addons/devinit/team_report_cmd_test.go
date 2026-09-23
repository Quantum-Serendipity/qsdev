package devinit

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/teamreport"
)

// TestCreateTeamIssues is the regression test for team-report printing
// "Created N issue(s)" while issues with no repository were silently skipped.
func TestCreateTeamIssues(t *testing.T) {
	t.Parallel()

	withRepo := teamreport.IssueSpec{Title: "a degraded", Repo: "org/a"}
	noRepo := teamreport.IssueSpec{Title: "b degraded"}
	createErr := errors.New("gh failed")

	tests := []struct {
		name        string
		issues      []teamreport.IssueSpec
		createErr   error
		wantCreated []string
		wantOutput  string
		wantErr     string
	}{
		{
			name:       "no issues",
			wantOutput: "No issues to create",
		},
		{
			name:        "all routable",
			issues:      []teamreport.IssueSpec{withRepo},
			wantCreated: []string{"org/a"},
			wantOutput:  "Created 1 of 1 issue(s)",
		},
		{
			name:        "issue without repo is not counted and fails the command",
			issues:      []teamreport.IssueSpec{withRepo, noRepo},
			wantCreated: []string{"org/a"},
			wantOutput:  "Created 1 of 2 issue(s)",
			wantErr:     "b degraded",
		},
		{
			name:       "only unroutable issues",
			issues:     []teamreport.IssueSpec{noRepo},
			wantOutput: "Created 0 of 1 issue(s)",
			wantErr:    "repository is unknown",
		},
		{
			name:        "creation failure is returned",
			issues:      []teamreport.IssueSpec{withRepo},
			createErr:   createErr,
			wantCreated: []string{"org/a"},
			wantErr:     "gh failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var created []string
			create := func(issues []teamreport.IssueSpec) (int, error) {
				for _, i := range issues {
					created = append(created, i.Repo)
				}
				if tt.createErr != nil {
					return 0, tt.createErr
				}
				return len(issues), nil
			}
			var out bytes.Buffer
			err := createTeamIssues(&out, tt.issues, create)

			if strings.Join(created, ",") != strings.Join(tt.wantCreated, ",") {
				t.Errorf("created %v, want %v", created, tt.wantCreated)
			}
			if tt.wantOutput != "" && !strings.Contains(out.String(), tt.wantOutput) {
				t.Errorf("output %q does not contain %q", out.String(), tt.wantOutput)
			}
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Errorf("error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}
