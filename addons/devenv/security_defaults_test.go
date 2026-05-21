package devenv

import (
	"reflect"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

func TestDefaultBasePackages_MatchesCatalog(t *testing.T) {
	t.Parallel()
	got := defaultBasePackages()
	want := catalog.Default().BasePackages()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("defaultBasePackages() = %v, want catalog.BasePackages() = %v", got, want)
	}
}

func TestDefaultSecurityHooks_MatchesCatalog(t *testing.T) {
	t.Parallel()
	got := defaultSecurityHooks()
	want := catalog.Default().SecurityHooks()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("defaultSecurityHooks() = %v, want catalog.SecurityHooks() = %v", got, want)
	}
}

func TestDefaultUnsetEnvVars_MatchesCatalog(t *testing.T) {
	t.Parallel()
	got := defaultUnsetEnvVars()
	want := catalog.Default().UnsetVars()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("defaultUnsetEnvVars() = %v, want catalog.UnsetVars() = %v", got, want)
	}
}

func TestDefaultCleanKeep_MatchesCatalog(t *testing.T) {
	t.Parallel()
	got := defaultCleanKeep()
	want := catalog.Default().KeepVars()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("defaultCleanKeep() = %v, want catalog.KeepVars() = %v", got, want)
	}
}

func TestDefaultSpecializedHooks_MatchesCatalog(t *testing.T) {
	t.Parallel()
	got := defaultSpecializedHooks()
	catHooks := catalog.Default().CustomHooks()

	if len(got) != len(catHooks) {
		t.Fatalf("defaultSpecializedHooks() count = %d, want catalog.CustomHooks() count = %d", len(got), len(catHooks))
	}

	catIDs := make(map[string]bool, len(catHooks))
	for _, h := range catHooks {
		catIDs[h.ID] = true
	}
	for _, h := range got {
		if !catIDs[h.ID] {
			t.Errorf("defaultSpecializedHooks() has ID %q not found in catalog.CustomHooks()", h.ID)
		}
	}
}
