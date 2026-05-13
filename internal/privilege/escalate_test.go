package privilege

import (
	"context"
	"runtime"
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
