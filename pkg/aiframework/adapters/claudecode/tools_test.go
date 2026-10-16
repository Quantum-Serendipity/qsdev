package claudecode_test

import (
	"context"
	"slices"
	"testing"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// TestCredentialExcludePaths_FromCanon verifies the sandbox exclusions come
// from the credential canon whatever the scope: an empty scope still keeps
// every home credential store (~/.ssh/**, ~/.aws/**, ~/.npmrc, ...) and the
// project secret files out of the agent's reach.
func TestCredentialExcludePaths_FromCanon(t *testing.T) {
	t.Parallel()
	arts, err := newAdapter().InjectCredentials(context.Background(), &aiframework.CredentialScope{})
	if err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}
	want := append([]string{"**/.env", "**/.env.*", "**/*.pem", "**/*.key", "**/secrets/**"},
		ccaddon.HomeCredentialReadGlobs()...)
	for _, w := range want {
		if !slices.Contains(arts.ExcludePaths, w) {
			t.Errorf("ExcludePaths lacks %q: %v", w, arts.ExcludePaths)
		}
	}
	for _, w := range []string{"~/.ssh/**", "~/.aws/**", "~/.npmrc"} {
		if !slices.Contains(arts.ExcludePaths, w) {
			t.Errorf("ExcludePaths lacks %q for an empty scope", w)
		}
	}
}
