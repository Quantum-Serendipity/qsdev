package container

import (
	"strings"
	"testing"
)

func TestRuntime_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		runtime Runtime
		want    string
	}{
		{RuntimePodmanRootless, "podman-rootless"},
		{RuntimePodmanRootful, "podman-rootful"},
		{RuntimeDocker, "docker"},
		{RuntimeNspawn, "nspawn"},
		{RuntimeNone, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.runtime.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuntime_IsPodman(t *testing.T) {
	t.Parallel()
	tests := []struct {
		runtime Runtime
		want    bool
	}{
		{RuntimePodmanRootless, true},
		{RuntimePodmanRootful, true},
		{RuntimeDocker, false},
		{RuntimeNspawn, false},
		{RuntimeNone, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.runtime), func(t *testing.T) {
			t.Parallel()
			if got := tt.runtime.IsPodman(); got != tt.want {
				t.Errorf("IsPodman() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuntime_IsRootless(t *testing.T) {
	t.Parallel()
	tests := []struct {
		runtime Runtime
		want    bool
	}{
		{RuntimePodmanRootless, true},
		{RuntimePodmanRootful, false},
		{RuntimeDocker, false},
		{RuntimeNspawn, false},
		{RuntimeNone, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.runtime), func(t *testing.T) {
			t.Parallel()
			if got := tt.runtime.IsRootless(); got != tt.want {
				t.Errorf("IsRootless() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCapabilities_NeedsRootfulFallback(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		caps *Capabilities
		want int // expected number of reasons
	}{
		{"nil capabilities", nil, 0},
		{"no fallback needed", &Capabilities{}, 0},
		{"GPU only", &Capabilities{GPUPassthrough: true}, 1},
		{"NFS only", &Capabilities{NFSMounts: true}, 1},
		{"GPU and NFS", &Capabilities{GPUPassthrough: true, NFSMounts: true}, 2},
		{"other fields only", &Capabilities{CgroupsV2: true, PrivilegedPorts: true}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reasons := tt.caps.NeedsRootfulFallback()
			if len(reasons) != tt.want {
				t.Errorf("NeedsRootfulFallback() returned %d reasons, want %d: %v", len(reasons), tt.want, reasons)
			}
		})
	}
}

func TestCapabilities_NeedsRootfulFallback_Content(t *testing.T) {
	t.Parallel()
	caps := &Capabilities{GPUPassthrough: true, NFSMounts: true}
	reasons := caps.NeedsRootfulFallback()

	hasGPU := false
	hasNFS := false
	for _, r := range reasons {
		if strings.Contains(r, "GPU") {
			hasGPU = true
		}
		if strings.Contains(r, "NFS") {
			hasNFS = true
		}
	}
	if !hasGPU {
		t.Error("expected GPU reason in fallback reasons")
	}
	if !hasNFS {
		t.Error("expected NFS reason in fallback reasons")
	}
}
