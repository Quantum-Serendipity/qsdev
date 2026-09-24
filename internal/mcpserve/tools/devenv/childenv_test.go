package devenv

import (
	"context"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestChildEnv checks the environment nix_run's children receive (F247): every
// variable the secrets canon or env_info withholds is dropped, everything else
// passes through unchanged, and an empty environment stays non-nil (a nil
// exec.Cmd.Env would inherit the server's whole environment).
func TestChildEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		environ []string
		want    []string
	}{
		{
			name: "credentials dropped, benign kept",
			environ: []string{
				"PATH=/usr/bin",
				"HOME=/home/dev",
				"GITHUB_TOKEN=fixture-value",
				"AWS_SECRET_ACCESS_KEY=fixture-value",
				"DATABASE_URL=postgres://u:p@db/app",
				"NIX_PATH=nixpkgs=/nix/var/nix/profiles/per-user/root/channels/nixpkgs",
				"LANG=C.UTF-8",
			},
			want: []string{
				"PATH=/usr/bin",
				"HOME=/home/dev",
				"NIX_PATH=nixpkgs=/nix/var/nix/profiles/per-user/root/channels/nixpkgs",
				"LANG=C.UTF-8",
			},
		},
		{
			name:    "cloud namespace withheld by env_info is dropped",
			environ: []string{"AWS_PROFILE=prod", "GCP_PROJECT=p", "TERM=xterm"},
			want:    []string{"TERM=xterm"},
		},
		{
			name:    "separator-less credential roots are dropped",
			environ: []string{"MYAPIKEY=x", "SECRETKEY=x", "SSH_AUTH_SOCK=/tmp/agent", "EDITOR=vi"},
			want:    []string{"EDITOR=vi"},
		},
		{
			name:    "empty environment stays non-nil",
			environ: nil,
			want:    []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := childEnv(tt.environ)
			if got == nil {
				t.Fatal("childEnv returned nil; exec.Cmd would inherit the full environment")
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("childEnv = %v, want %v", got, tt.want)
			}
			for _, kv := range got {
				name, _, _ := strings.Cut(kv, "=")
				if isSensitiveEnv(name) {
					t.Errorf("child receives %s, which env_info withholds", name)
				}
			}
		})
	}
}

// TestRunProcessGroupScrubsCredentials is the F247 end-to-end regression: a
// process nix_run starts cannot print the server's credentials with `env`.
func TestRunProcessGroupScrubsCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell semantics; not run on windows")
	}
	t.Setenv("GITHUB_TOKEN", "fixture-gh-value")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "fixture-aws-value")
	t.Setenv("QSDEV_F247_BENIGN", "visible")

	res := runProcessGroup(context.Background(), "", "sh", []string{"-c", "env"}, "", 5*time.Second)
	if res.startErr != nil || res.exitCode != 0 {
		t.Fatalf("env failed: start=%v exit=%d stderr=%q", res.startErr, res.exitCode, res.stderr)
	}
	// Match whole NAME= lines: another variable's value may legitimately
	// mention a credential variable's name.
	lines := strings.Split(res.stdout, "\n")
	for _, name := range []string{"GITHUB_TOKEN", "AWS_SECRET_ACCESS_KEY"} {
		if slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, name+"=") }) {
			t.Errorf("child environment leaks %s", name)
		}
	}
	for _, value := range []string{"fixture-gh-value", "fixture-aws-value"} {
		if strings.Contains(res.stdout, value) {
			t.Errorf("child environment leaks the value %q", value)
		}
	}
	if !strings.Contains(res.stdout, "QSDEV_F247_BENIGN=visible") {
		t.Error("benign variable QSDEV_F247_BENIGN missing from child environment")
	}
}

// TestToolsNixRunOptIn checks that qsdev_nix_run is registered only when
// selected (the serve command leaves it out in gateway mode), while env_info is
// always offered.
func TestToolsNixRunOptIn(t *testing.T) {
	t.Parallel()
	for _, nixRun := range []bool{false, true} {
		var names []string
		for _, r := range Tools(t.TempDir(), nixRun) {
			names = append(names, r.Name)
		}
		if got := slices.Contains(names, "qsdev_nix_run"); got != nixRun {
			t.Errorf("Tools(nixRun=%t) registers qsdev_nix_run = %t; tools %v", nixRun, got, names)
		}
		if !slices.Contains(names, "qsdev_env_info") {
			t.Errorf("Tools(nixRun=%t) = %v, want qsdev_env_info", nixRun, names)
		}
	}
}
