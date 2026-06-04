package selfupdate

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.GitHubOwner != "Quantum-Serendipity" {
		t.Errorf("GitHubOwner = %q, want %q", cfg.GitHubOwner, "Quantum-Serendipity")
	}
	if cfg.GitHubRepo != "qsdev" {
		t.Errorf("GitHubRepo = %q, want %q", cfg.GitHubRepo, "qsdev")
	}
	if cfg.BinaryName != "qsdev" {
		t.Errorf("BinaryName = %q, want %q", cfg.BinaryName, "qsdev")
	}
	if cfg.CheckInterval == 0 {
		t.Error("CheckInterval should not be zero")
	}
	if cfg.CacheDir == "" {
		t.Error("CacheDir should not be empty")
	}
}

func TestReleaseConstruction(t *testing.T) {
	r := Release{
		Version: "1.2.3",
		TagName: "v1.2.3",
		URL:     "https://github.com/example/repo/releases/tag/v1.2.3",
		Body:    "Some release notes",
		Assets: []Asset{
			{Name: "qsdev_1.2.3_Linux_x86_64.tar.gz", URL: "https://example.com/archive.tar.gz"},
			{Name: "checksums.txt", URL: "https://example.com/checksums.txt"},
		},
	}

	if r.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", r.Version, "1.2.3")
	}
	if r.TagName != "v1.2.3" {
		t.Errorf("TagName = %q, want %q", r.TagName, "v1.2.3")
	}
	if r.URL != "https://github.com/example/repo/releases/tag/v1.2.3" {
		t.Errorf("URL = %q, want %q", r.URL, "https://github.com/example/repo/releases/tag/v1.2.3")
	}
	if r.Body != "Some release notes" {
		t.Errorf("Body = %q, want %q", r.Body, "Some release notes")
	}
	if len(r.Assets) != 2 {
		t.Errorf("len(Assets) = %d, want 2", len(r.Assets))
	}
}

func TestArchiveFilename(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		version    string
		targetOS   string
		targetArch string
		want       string
	}{
		{
			name:       "linux amd64",
			binaryName: "qsdev",
			version:    "1.2.3",
			targetOS:   "linux",
			targetArch: "amd64",
			want:       "qsdev_1.2.3_Linux_x86_64.tar.gz",
		},
		{
			name:       "linux arm64",
			binaryName: "qsdev",
			version:    "1.2.3",
			targetOS:   "linux",
			targetArch: "arm64",
			want:       "qsdev_1.2.3_Linux_arm64.tar.gz",
		},
		{
			name:       "darwin amd64",
			binaryName: "qsdev",
			version:    "1.0.0",
			targetOS:   "darwin",
			targetArch: "amd64",
			want:       "qsdev_1.0.0_Darwin_x86_64.tar.gz",
		},
		{
			name:       "darwin arm64",
			binaryName: "qsdev",
			version:    "2.0.0",
			targetOS:   "darwin",
			targetArch: "arm64",
			want:       "qsdev_2.0.0_Darwin_arm64.tar.gz",
		},
		{
			name:       "windows amd64",
			binaryName: "qsdev",
			version:    "1.0.0",
			targetOS:   "windows",
			targetArch: "amd64",
			want:       "qsdev_1.0.0_Windows_x86_64.zip",
		},
		{
			name:       "unknown os/arch passthrough",
			binaryName: "qsdev",
			version:    "1.0.0",
			targetOS:   "freebsd",
			targetArch: "riscv64",
			want:       "qsdev_1.0.0_freebsd_riscv64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArchiveFilename(tt.binaryName, tt.version, tt.targetOS, tt.targetArch)
			if got != tt.want {
				t.Errorf("ArchiveFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOSMapping(t *testing.T) {
	expected := map[string]string{
		"linux":   "Linux",
		"darwin":  "Darwin",
		"windows": "Windows",
	}
	for k, v := range expected {
		if osMapping[k] != v {
			t.Errorf("osMapping[%q] = %q, want %q", k, osMapping[k], v)
		}
	}
}

func TestArchMapping(t *testing.T) {
	expected := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	}
	for k, v := range expected {
		if archMapping[k] != v {
			t.Errorf("archMapping[%q] = %q, want %q", k, archMapping[k], v)
		}
	}
}
