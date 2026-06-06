package devinit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestStageStatus_String(t *testing.T) {
	tests := []struct {
		status   StageStatus
		expected string
	}{
		{StageSuccess, "updated"},
		{StageSkipped, "skipped"},
		{StageFailed, "failed"},
		{StageUpToDate, "up-to-date"},
		{StageStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.expected {
				t.Errorf("StageStatus(%d).String() = %q, want %q", int(tt.status), got, tt.expected)
			}
		})
	}
}

func newTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

func TestRunSelfUpdateStage_DevBuild(t *testing.T) {
	cmd, _ := newTestCmd()
	result := runSelfUpdateStage(cmd, FullUpdateOptions{})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped for dev build, got %v", result.Status)
	}
	if result.Name != "Self-update" {
		t.Errorf("expected name 'Self-update', got %q", result.Name)
	}
	if !strings.Contains(result.Message, "dev build") {
		t.Errorf("expected message to mention 'dev build', got %q", result.Message)
	}
}

func TestRunDevenvInputStage_NotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cmd, _ := newTestCmd()
	result := runDevenvInputStage(cmd, FullUpdateOptions{})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped when devenv not on PATH, got %v", result.Status)
	}
	if result.Name != "Devenv inputs" {
		t.Errorf("expected name 'Devenv inputs', got %q", result.Name)
	}
	if !strings.Contains(result.Message, "devenv not installed") {
		t.Errorf("expected message about devenv not installed, got %q", result.Message)
	}
}

func TestRunDevenvInputStage_DryRun(t *testing.T) {
	cmd, _ := newTestCmd()
	result := runDevenvInputStage(cmd, FullUpdateOptions{DryRun: true})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped for dry-run, got %v", result.Status)
	}
	if strings.Contains(result.Message, "devenv not installed") {
		return
	}
	if !strings.Contains(result.Message, "dry-run") {
		t.Errorf("expected message to mention 'dry-run', got %q", result.Message)
	}
	if !strings.Contains(result.Message, "devenv update") {
		t.Errorf("expected message to mention 'devenv update', got %q", result.Message)
	}
}

func TestRunFullUpdate_SelectiveStages(t *testing.T) {
	tests := []struct {
		name          string
		opts          FullUpdateOptions
		expectStages  []string
		excludeStages []string
	}{
		{
			name:          "self-only runs only self-update",
			opts:          FullUpdateOptions{SelfOnly: true},
			expectStages:  []string{"Self-update"},
			excludeStages: []string{"Config regeneration", "Devenv inputs"},
		},
		{
			name:          "configs-only runs only config regeneration",
			opts:          FullUpdateOptions{ConfigsOnly: true},
			expectStages:  []string{"Config regeneration"},
			excludeStages: []string{"Self-update", "Devenv inputs"},
		},
		{
			name:          "deps-only runs only devenv inputs",
			opts:          FullUpdateOptions{DepsOnly: true},
			expectStages:  []string{"Devenv inputs"},
			excludeStages: []string{"Self-update", "Config regeneration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())

			cmd, buf := newTestCmd()
			_ = runFullUpdate(cmd, tt.opts)

			output := buf.String()

			for _, stage := range tt.expectStages {
				if !strings.Contains(output, stage) {
					t.Errorf("expected output to contain stage %q, got:\n%s", stage, output)
				}
			}

			for _, stage := range tt.excludeStages {
				summaryIdx := strings.Index(output, "Update Summary:")
				if summaryIdx >= 0 {
					summary := output[summaryIdx:]
					if strings.Contains(summary, stage) {
						t.Errorf("expected summary NOT to contain stage %q, got:\n%s", stage, summary)
					}
				}
			}
		})
	}
}

func TestRunFullUpdate_AllStages(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cmd, buf := newTestCmd()
	err := runFullUpdate(cmd, FullUpdateOptions{})

	output := buf.String()

	allStages := []string{"Self-update", "Config regeneration", "Devenv inputs"}
	for _, stage := range allStages {
		if !strings.Contains(output, stage) {
			t.Errorf("expected output to contain stage %q, got:\n%s", stage, output)
		}
	}

	if err == nil {
		t.Error("expected error from runFullUpdate (config stage should fail), got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "one or more update stages failed") {
		t.Errorf("unexpected error message: %v", err)
	}

	if !strings.Contains(output, "[1/3]") {
		t.Errorf("expected progress indicator [1/3] in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[2/3]") {
		t.Errorf("expected progress indicator [2/3] in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[3/3]") {
		t.Errorf("expected progress indicator [3/3] in output, got:\n%s", output)
	}

	if !strings.Contains(output, "Update Summary:") {
		t.Errorf("expected 'Update Summary:' in output, got:\n%s", output)
	}
}

func TestIsMinorBump(t *testing.T) {
	t.Parallel()

	tests := []struct {
		oldVer string
		newVer string
		want   bool
	}{
		{"0.7.0", "0.7.1", false},
		{"0.7.2", "0.7.3", false},
		{"0.7.0", "0.8.0", true},
		{"0.7.3", "0.8.0", true},
		{"1.0.0", "2.0.0", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "1.0.1", false},
		{"v0.7.0", "v0.7.1", false},
		{"v0.7.0", "v0.8.0", true},
		{"0.7.0", "0.7.0", false},
		{"1.0", "1.1", true},
		{"1.0", "1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.oldVer+"→"+tt.newVer, func(t *testing.T) {
			t.Parallel()
			got := isMinorBump(tt.oldVer, tt.newVer)
			if got != tt.want {
				t.Errorf("isMinorBump(%q, %q) = %v, want %v", tt.oldVer, tt.newVer, got, tt.want)
			}
		})
	}
}

func TestParseMajorMinor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		wantMajor int
		wantMinor int
	}{
		{"0.7.3", 0, 7},
		{"1.2.3", 1, 2},
		{"v1.2.3", 1, 2},
		{"10.20.30", 10, 20},
		{"1.0", 1, 0},
		{"1", 1, 0},
		{"", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			major, minor := parseMajorMinor(tt.input)
			if major != tt.wantMajor || minor != tt.wantMinor {
				t.Errorf("parseMajorMinor(%q) = (%d, %d), want (%d, %d)",
					tt.input, major, minor, tt.wantMajor, tt.wantMinor)
			}
		})
	}
}

func TestUpdateCmd_CheckFlag(t *testing.T) {
	cmd := updateCmd()
	if err := cmd.Flags().Set("check", "true"); err != nil {
		t.Fatalf("failed to set --check flag: %v", err)
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// In test context, version is "dev", so check-only prints skip message.
	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Dev build") {
		t.Errorf("expected dev build skip message, got: %s", output)
	}
}

func TestUpdateCmd_HasExpectedFlags(t *testing.T) {
	cmd := updateCmd()

	expectedFlags := []string{
		"dry-run", "force", "self-only", "configs-only",
		"deps-only", "check", "changelog",
	}

	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s to be registered", name)
		}
	}
}
