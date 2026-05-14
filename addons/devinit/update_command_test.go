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

func TestRunSelfUpdateStage_DevBuild(t *testing.T) {
	// In test context, version.Info().Version returns "dev",
	// so runSelfUpdateStage should skip.
	result := runSelfUpdateStage(FullUpdateOptions{})

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
	// Set PATH to an empty directory so devenv is not found.
	t.Setenv("PATH", t.TempDir())

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

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
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	result := runDevenvInputStage(cmd, FullUpdateOptions{DryRun: true})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped for dry-run, got %v", result.Status)
	}
	// If devenv is not on PATH, the "not installed" check fires before dry-run.
	// Either way the status should be StageSkipped. We check the appropriate message.
	if strings.Contains(result.Message, "devenv not installed") {
		// devenv not on PATH — that check fires first; still StageSkipped, which is fine.
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

	// Set PATH to empty dir so devenv is not found (avoids running real devenv).
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())

			cmd := &cobra.Command{}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// We don't check the error here because the config stage will fail
			// (no saved state file) — we just want to verify which stages ran.
			_ = runFullUpdate(cmd, tt.opts)

			output := buf.String()

			// Verify expected stages appear in the summary.
			for _, stage := range tt.expectStages {
				if !strings.Contains(output, stage) {
					t.Errorf("expected output to contain stage %q, got:\n%s", stage, output)
				}
			}

			// Verify excluded stages do not appear in the summary.
			for _, stage := range tt.excludeStages {
				// Check that the stage name doesn't appear in the "Update Summary:" section.
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
	// Set PATH to empty dir so devenv is not found.
	t.Setenv("PATH", t.TempDir())

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Default options = all three stages.
	err := runFullUpdate(cmd, FullUpdateOptions{})

	output := buf.String()

	// All three stages should appear in the output.
	allStages := []string{"Self-update", "Config regeneration", "Devenv inputs"}
	for _, stage := range allStages {
		if !strings.Contains(output, stage) {
			t.Errorf("expected output to contain stage %q, got:\n%s", stage, output)
		}
	}

	// The config stage will fail (no saved answers file in test context),
	// so we expect an error.
	if err == nil {
		t.Error("expected error from runFullUpdate (config stage should fail), got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "one or more update stages failed") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Verify the progress indicators are present (e.g. "[1/3]", "[2/3]", "[3/3]").
	if !strings.Contains(output, "[1/3]") {
		t.Errorf("expected progress indicator [1/3] in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[2/3]") {
		t.Errorf("expected progress indicator [2/3] in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[3/3]") {
		t.Errorf("expected progress indicator [3/3] in output, got:\n%s", output)
	}

	// Verify summary section exists.
	if !strings.Contains(output, "Update Summary:") {
		t.Errorf("expected 'Update Summary:' in output, got:\n%s", output)
	}
}
