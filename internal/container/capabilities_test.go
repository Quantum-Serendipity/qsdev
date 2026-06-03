package container

import (
	"context"
	"testing"
)

func TestDetectCapabilities_GPUPresent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		globResults: map[string][]string{
			"/dev/nvidia*": {"/dev/nvidia0", "/dev/nvidiactl"},
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimePodmanRootless, Rootless: true}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if !caps.GPUPassthrough {
		t.Error("GPUPassthrough = false, want true")
	}
}

func TestDetectCapabilities_GPUAbsent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		globResults: map[string][]string{},
		user:        "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if caps.GPUPassthrough {
		t.Error("GPUPassthrough = true, want false")
	}
}

func TestDetectCapabilities_NFSPresent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		files: map[string][]byte{
			"/proc/mounts": []byte("server:/export /mnt/nfs nfs4 rw,relatime 0 0\n"),
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if !caps.NFSMounts {
		t.Error("NFSMounts = false, want true")
	}
}

func TestDetectCapabilities_NFSAbsent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		files: map[string][]byte{
			"/proc/mounts": []byte("tmpfs /tmp tmpfs rw,nosuid,nodev 0 0\n/dev/sda1 / ext4 rw,relatime 0 0\n"),
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if caps.NFSMounts {
		t.Error("NFSMounts = true, want false")
	}
}

func TestDetectCapabilities_SubUidConfigured(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		files: map[string][]byte{
			"/etc/subuid": []byte("testuser:100000:65536\nother:200000:65536\n"),
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if !caps.UserNamespaceConfigured {
		t.Error("UserNamespaceConfigured = false, want true")
	}
}

func TestDetectCapabilities_SubUidMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		files: map[string][]byte{
			"/etc/subuid": []byte("otheruser:100000:65536\n"),
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if caps.UserNamespaceConfigured {
		t.Error("UserNamespaceConfigured = true, want false")
	}
}

func TestDetectCapabilities_CgroupsV2(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		fileInfos: map[string]bool{
			"/sys/fs/cgroup/cgroup.controllers": true,
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if !caps.CgroupsV2 {
		t.Error("CgroupsV2 = false, want true")
	}
}

func TestDetectCapabilities_CgroupsV1(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		fileInfos: map[string]bool{},
		user:      "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if caps.CgroupsV2 {
		t.Error("CgroupsV2 = true, want false")
	}
}

func TestDetectCapabilities_NilInfo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{}

	caps, err := DetectCapabilities(ctx, prober, nil)
	if err == nil {
		t.Fatal("DetectCapabilities() expected error for nil RuntimeInfo")
	}
	if caps != nil {
		t.Errorf("expected nil Capabilities, got %+v", caps)
	}
}

func TestDetectCapabilities_AllPresent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		globResults: map[string][]string{
			"/dev/nvidia*": {"/dev/nvidia0"},
		},
		files: map[string][]byte{
			"/proc/mounts": []byte("server:/vol /data nfs rw 0 0\n"),
			"/etc/subuid":  []byte("testuser:100000:65536\n"),
			"/proc/sys/net/ipv4/ip_unprivileged_port_start": []byte("80\n"),
		},
		fileInfos: map[string]bool{
			"/sys/fs/cgroup/cgroup.controllers": true,
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimePodmanRootless, Rootless: true}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if !caps.GPUPassthrough {
		t.Error("GPUPassthrough = false, want true")
	}
	if !caps.NFSMounts {
		t.Error("NFSMounts = false, want true")
	}
	if !caps.UserNamespaceConfigured {
		t.Error("UserNamespaceConfigured = false, want true")
	}
	if !caps.CgroupsV2 {
		t.Error("CgroupsV2 = false, want true")
	}
	if !caps.RootlessSupported {
		t.Error("RootlessSupported = false, want true")
	}
	if !caps.PrivilegedPorts {
		t.Error("PrivilegedPorts = false, want true")
	}
}

func TestDetectCapabilities_NonePresent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		globResults: map[string][]string{},
		files: map[string][]byte{
			"/proc/mounts": []byte("tmpfs /tmp tmpfs rw 0 0\n"),
			"/etc/subuid":  []byte("otheruser:100000:65536\n"),
		},
		fileInfos: map[string]bool{},
		user:      "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if caps.GPUPassthrough {
		t.Error("GPUPassthrough = true, want false")
	}
	if caps.NFSMounts {
		t.Error("NFSMounts = true, want false")
	}
	if caps.UserNamespaceConfigured {
		t.Error("UserNamespaceConfigured = true, want false")
	}
	if caps.CgroupsV2 {
		t.Error("CgroupsV2 = true, want false")
	}
	if caps.RootlessSupported {
		t.Error("RootlessSupported = true, want false")
	}
	if caps.PrivilegedPorts {
		t.Error("PrivilegedPorts = true, want false")
	}
}

func TestDetectCapabilities_RootlessSupported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		info     *RuntimeInfo
		wantRoot bool
	}{
		{
			name:     "podman rootless",
			info:     &RuntimeInfo{Active: RuntimePodmanRootless, Rootless: true},
			wantRoot: true,
		},
		{
			name:     "podman rootful",
			info:     &RuntimeInfo{Active: RuntimePodmanRootful, Rootless: false},
			wantRoot: false,
		},
		{
			name:     "docker",
			info:     &RuntimeInfo{Active: RuntimeDocker, Rootless: false},
			wantRoot: false,
		},
		{
			name:     "nspawn",
			info:     &RuntimeInfo{Active: RuntimeNspawn},
			wantRoot: false,
		},
		{
			name:     "none",
			info:     &RuntimeInfo{Active: RuntimeNone},
			wantRoot: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			prober := &mockProber{user: "testuser"}

			caps, err := DetectCapabilities(ctx, prober, tt.info)
			if err != nil {
				t.Fatalf("DetectCapabilities() error = %v", err)
			}
			if caps.RootlessSupported != tt.wantRoot {
				t.Errorf("RootlessSupported = %v, want %v", caps.RootlessSupported, tt.wantRoot)
			}
		})
	}
}

func TestDetectCapabilities_PrivilegedPorts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		threshold string
		want      bool
	}{
		{"default 1024", "1024\n", true},
		{"lowered to 80", "80\n", true},
		{"set to 0", "0\n", false},
		{"raised to 2048", "2048\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			prober := &mockProber{
				files: map[string][]byte{
					"/proc/sys/net/ipv4/ip_unprivileged_port_start": []byte(tt.threshold),
				},
				user: "testuser",
			}
			info := &RuntimeInfo{Active: RuntimeDocker}

			caps, err := DetectCapabilities(ctx, prober, info)
			if err != nil {
				t.Fatalf("DetectCapabilities() error = %v", err)
			}
			if caps.PrivilegedPorts != tt.want {
				t.Errorf("PrivilegedPorts = %v, want %v (threshold=%q)", caps.PrivilegedPorts, tt.want, tt.threshold)
			}
		})
	}
}

func TestDetectCapabilities_EmptyUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		files: map[string][]byte{
			"/etc/subuid": []byte("testuser:100000:65536\n"),
		},
		user: "", // empty user
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	// With empty user, subuid lookup should be skipped.
	if caps.UserNamespaceConfigured {
		t.Error("UserNamespaceConfigured = true, want false when user is empty")
	}
}

func TestDetectCapabilities_NFSv3(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prober := &mockProber{
		files: map[string][]byte{
			"/proc/mounts": []byte("server:/export /mnt/nfs nfs rw,vers=3 0 0\n"),
		},
		user: "testuser",
	}
	info := &RuntimeInfo{Active: RuntimeDocker}

	caps, err := DetectCapabilities(ctx, prober, info)
	if err != nil {
		t.Fatalf("DetectCapabilities() error = %v", err)
	}
	if !caps.NFSMounts {
		t.Error("NFSMounts = false, want true for nfs (v3) mount")
	}
}
