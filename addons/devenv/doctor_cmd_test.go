package devenv

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
)

func TestDoctorCmd_Flags(t *testing.T) {
	cmd := doctorCmd()

	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", cmd.Use, "doctor")
	}

	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("expected --json flag to be registered")
	}
	checkFlag := cmd.Flags().Lookup("check")
	if checkFlag == nil {
		t.Error("expected --check flag to be registered")
	}
}

func TestDoctorCmd_DefaultOutput(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{})

	// Ignore the error since some required tools may not be present.
	_ = cmd.Execute()

	output := buf.String()

	// The report should contain these sections.
	for _, section := range []string{"System", "Shell", "Required Tools"} {
		if !strings.Contains(output, section) {
			t.Errorf("output missing section %q", section)
		}
	}
}

func TestDoctorCmd_JSONOutput(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}

	// Verify valid JSON.
	var report doctor.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify some expected fields.
	if report.QsdevVersion == "" {
		t.Error("expected non-empty qsdev_version in JSON output")
	}
	if report.System.OS == "" {
		t.Error("expected non-empty system.os in JSON output")
	}
	if report.System.Arch == "" {
		t.Error("expected non-empty system.arch in JSON output")
	}
}

func TestDoctorCmd_CheckMode_Output(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--check"})

	err := cmd.Execute()
	output := buf.String()

	// Either all tools are present (success) or some are missing (error).
	if err == nil {
		if !strings.Contains(output, "All required tools are present") {
			t.Errorf("check mode success should say all tools present, got: %s", output)
		}
	} else {
		if !strings.Contains(output, "Missing required tools") {
			t.Errorf("check mode failure should list missing tools, got: %s", output)
		}
	}
}

func TestDoctorCmd_JSONContainsTools(t *testing.T) {
	cmd := doctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}

	var report doctor.Report
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// There should be at least some required tools listed.
	if len(report.RequiredTools) == 0 {
		t.Error("expected at least one required tool in JSON output")
	}

	// Verify required tool names include expected ones.
	names := make(map[string]bool)
	for _, t := range report.RequiredTools {
		names[t.Name] = true
	}
	for _, expected := range []string{"git", "go", "node", "npm"} {
		if !names[expected] {
			t.Errorf("expected required tool %q in JSON output", expected)
		}
	}
}
