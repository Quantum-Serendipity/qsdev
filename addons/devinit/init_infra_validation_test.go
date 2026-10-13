package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestInit_RejectsInvalidInfraEndpointInEveryScope checks the endpoint flags
// are validated before anything is saved, whichever generators run: a
// --claude-only init never reaches the devenv generator's endpoint checks, so
// it used to exit 0 and record an embedded password in .qsdev.yaml and the
// answers file.
func TestInit_RejectsInvalidInfraEndpointInEveryScope(t *testing.T) {
	scopes := []struct {
		name  string
		flags []string
	}{{"claude-only", []string{"--claude-only"}}, {"devenv-only", []string{"--devenv-only"}}, {"full", nil}}
	endpoints := []struct {
		name  string
		flags []string
		want  string
	}{
		{"embedded credentials", []string{"--registry-proxy", "https://user:pass@r.example.com"}, "must not embed credentials"},
		{"port out of range", []string{"--registry-proxy", "https://nexus.corp.internal:65536"}, "outside 1-65535"},
		{"example cachix cache with trailing dot", []string{
			"--nix-cache", "https://myorg.cachix.org.",
			"--nix-cache-public-key", "corp.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=",
		}, "example Cachix cache"},
	}
	for _, scope := range scopes {
		for _, ep := range endpoints {
			t.Run(scope.name+"/"+ep.name, func(t *testing.T) {
				dir := newGoProject(t)
				args := append([]string{"--yes", "--lang", "go"}, scope.flags...)
				out, err := executeInitCmd(t, dir, append(args, ep.flags...)...)
				if err == nil || !strings.Contains(err.Error(), ep.want) {
					t.Fatalf("init error = %v, want one containing %q\n%s", err, ep.want, out)
				}
				if strings.Contains(err.Error(), "pass@") || strings.Contains(out, "pass@") {
					t.Errorf("init echoed the embedded password:\n%v\n%s", err, out)
				}
				for _, rel := range []string{
					branding.Get().ConfigFile,
					filepath.Join(answers.PrimaryDir(), answers.PrimaryFilename()),
				} {
					if _, statErr := os.Stat(filepath.Join(dir, rel)); statErr == nil {
						t.Errorf("%s was written for a rejected endpoint", rel)
					}
				}
			})
		}
	}
}
