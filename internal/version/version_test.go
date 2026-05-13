package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestInfoDefaults(t *testing.T) {
	info := Info()

	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}

func TestInfoStringFormat(t *testing.T) {
	info := Info()
	s := info.String()

	if !strings.Contains(s, info.GoVersion) {
		t.Errorf("String() = %q, missing GoVersion %q", s, info.GoVersion)
	}
	if !strings.Contains(s, info.OS+"/"+info.Arch) {
		t.Errorf("String() = %q, missing OS/Arch", s)
	}
}

func TestCommitTruncation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
	}{
		{"short", "abc123", 12},
		{"exact", "abcdef123456", 12},
		{"long", "abcdef1234567890abcdef", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := commit
			commit = tt.input
			defer func() { commit = old }()

			info := Info()
			if len(info.Commit) > tt.maxLen {
				t.Errorf("Commit %q has length %d, want <= %d", info.Commit, len(info.Commit), tt.maxLen)
			}
		})
	}
}

func TestCommitDirtyTruncation(t *testing.T) {
	old := commit
	commit = "abcdef1234567890abcdef-dirty"
	defer func() { commit = old }()

	info := Info()
	if !strings.HasSuffix(info.Commit, "-dirty") {
		t.Errorf("Commit %q should end with -dirty", info.Commit)
	}
	if len(info.Commit) > 18 {
		t.Errorf("Commit %q length %d, want <= 18", info.Commit, len(info.Commit))
	}
}
