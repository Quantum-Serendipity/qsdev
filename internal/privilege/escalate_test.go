package privilege

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestIsElevated_InverseOfNeedsElevation(t *testing.T) {
	if NeedsElevation() == IsElevated() {
		t.Error("NeedsElevation() and IsElevated() must be inverses")
	}
}

func TestDetectElevationTool_ReturnsRecognized(t *testing.T) {
	tool := DetectElevationTool()
	valid := map[string]bool{
		"": true, "sudo": true, "doas": true, "pkexec": true,
		"gsudo": true, "powershell": true,
	}
	if !valid[tool] {
		t.Errorf("DetectElevationTool() = %q, not recognized", tool)
	}
}

func TestBatchElevatedInstall_EmptyPackages(t *testing.T) {
	err := BatchElevatedInstall(context.Background(), "apt-get", []string{"install", "-y"}, nil)
	if err != nil {
		t.Errorf("BatchElevatedInstall with nil packages = %v, want nil", err)
	}
}

func TestBatchElevatedInstall_EmptySlice(t *testing.T) {
	err := BatchElevatedInstall(context.Background(), "apt-get", []string{"install", "-y"}, []string{})
	if err != nil {
		t.Errorf("BatchElevatedInstall with empty packages = %v, want nil", err)
	}
}

func TestPowerShellQuoteEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean", "package-name", "'package-name'"},
		{"single quote", "test'inject", "'test''inject'"},
		{"injection attempt", "test'; Get-Process; '", "'test''; Get-Process; '''"},
		{"multiple quotes", "a'b'c", "'a''b''c'"},
		{"empty", "", "''"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := strings.ReplaceAll(tt.input, "'", "''")
			got := "'" + escaped + "'"
			if got != tt.expected {
				t.Errorf("quote(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDetectElevationTool_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}
	tool := DetectElevationTool()
	// On most Linux systems, sudo should be available
	if tool == "" {
		t.Log("No elevation tool found — expected on minimal systems")
	}
}
