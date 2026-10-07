package modules

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// credentialKeeps returns the names keeper passes through devenv.yaml
// clean.keep that are credentials: listed in secrets.KnownCredentialVars or
// the catalog unset_vars, or sensitive by name.
func credentialKeeps(keeper ecosystem.EnvKeeper, unset []string) []string {
	var bad []string
	for _, name := range keeper.KeepEnvVars() {
		if slices.Contains(secrets.KnownCredentialVars, name) ||
			slices.Contains(unset, name) ||
			secrets.IsSensitiveName(name) {
			bad = append(bad, name)
		}
	}
	return bad
}

// TestEnvKeepers_NeverKeepCredentials guards that no registered module can
// pass a credential into the devenv shell. Every KeepEnvVars name lands in
// devenv.yaml clean.keep, which inherits the variable from the user's shell,
// so a credential there would undo the clean-environment stripping. The guard
// subtest proves the check fails for a module keeping AWS_SECRET_ACCESS_KEY.
func TestEnvKeepers_NeverKeepCredentials(t *testing.T) {
	t.Parallel()

	cat, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("catalog.LoadEmbeddedOnly: %v", err)
	}
	unset := cat.UnsetVars()

	keepers := 0
	for _, mod := range ecosystem.DefaultRegistry().All() {
		keeper, ok := mod.(ecosystem.EnvKeeper)
		if !ok {
			continue
		}
		keepers++
		if bad := credentialKeeps(keeper, unset); len(bad) > 0 {
			t.Errorf("module %q keeps credential variables %v", mod.Name(), bad)
		}
	}
	if keepers == 0 {
		t.Fatal("no registered module implements ecosystem.EnvKeeper; the invariant checks nothing")
	}

	t.Run("guard", func(t *testing.T) {
		t.Parallel()
		mock := &ecosystem.MockModule{
			NameVal:        "leaky",
			KeepEnvVarsVal: []string{"AWS_REGION", "AWS_SECRET_ACCESS_KEY"},
		}
		bad := credentialKeeps(mock, unset)
		if !slices.Equal(bad, []string{"AWS_SECRET_ACCESS_KEY"}) {
			t.Errorf("credentialKeeps = %v, want [AWS_SECRET_ACCESS_KEY]", bad)
		}
	})
}
