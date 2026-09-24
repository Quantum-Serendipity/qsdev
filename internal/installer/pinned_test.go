package installer_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

func TestIsExactSemver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"2.1.273", true},
		{"v1.0.0", true},
		{"1.2.3-rc.1", true},
		{"1.2.3+build.5", true},
		{"", false},
		{"latest", false},
		{"stable", false},
		{"^2.1.0", false},
		{"~2.1.0", false},
		{"2.1", false},
		{"2.x", false},
		{">=2.0.0", false},
		{"2.1.273 || 2.1.274", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			if got := installer.IsExactSemver(tt.version); got != tt.want {
				t.Errorf("IsExactSemver(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestReleaseCutoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 24, 7, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	if got, want := installer.ReleaseCutoff(now, installer.NpmMinReleaseAge), "2026-09-21T11:30:00Z"; got != want {
		t.Errorf("ReleaseCutoff = %q, want %q", got, want)
	}
}

// TestNpmGlobalInstallCmd covers F110: qsdev's own global npm installs name
// an exact release, are age-gated with --before, and run no lifecycle
// scripts unless the package is explicitly allowed them.
func TestNpmGlobalInstallCmd(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	const cutoff = "--before=2026-09-21T12:00:00Z"

	tests := []struct {
		name    string
		pkg     installer.NpmPackage
		want    []string
		wantErr error
	}{
		{
			name: "scripts disabled by default",
			pkg:  installer.NpmPackage{Name: "@scope/tool", Version: "1.2.3"},
			want: []string{"npm", "install", "-g", "--ignore-scripts", cutoff, "@scope/tool@1.2.3"},
		},
		{
			name: "scripts allowed",
			pkg:  installer.NpmPackage{Name: "@scope/tool", Version: "1.2.3", RunInstallScripts: true},
			want: []string{"npm", "install", "-g", cutoff, "@scope/tool@1.2.3"},
		},
		{
			name:    "dist-tag refused",
			pkg:     installer.NpmPackage{Name: "tool", Version: "latest"},
			wantErr: installer.ErrUnpinned,
		},
		{
			name:    "range refused",
			pkg:     installer.NpmPackage{Name: "tool", Version: "^1.2.3"},
			wantErr: installer.ErrUnpinned,
		},
		{
			name:    "missing version refused",
			pkg:     installer.NpmPackage{Name: "tool"},
			wantErr: installer.ErrUnpinned,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := installer.NpmGlobalInstallCmd(tt.pkg, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NpmGlobalInstallCmd: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("command = %q, want %q", got, tt.want)
			}
		})
	}

	for _, name := range []string{"", "--registry=https://evil.example/x", "-g"} {
		t.Run("bad name refused/"+name, func(t *testing.T) {
			t.Parallel()
			if _, err := installer.NpmGlobalInstallCmd(installer.NpmPackage{Name: name, Version: "1.2.3"}, now); err == nil {
				t.Errorf("NpmGlobalInstallCmd with name %q succeeded, want an error", name)
			}
		})
	}
}
