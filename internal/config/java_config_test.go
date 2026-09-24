package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestParseQsdevConfig_JavaRepositoryAllowlist checks the java block parses
// and survives the config -> answers -> config round trip join and init use.
func TestParseQsdevConfig_JavaRepositoryAllowlist(t *testing.T) {
	t.Parallel()
	cfg, err := ParseQsdevConfigBytes([]byte("version: 2\njava:\n  repository_allowlist: [confluent, company-nexus]\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"confluent", "company-nexus"}
	if !slices.Equal(cfg.Java.RepositoryAllowlist, want) {
		t.Fatalf("parsed allowlist = %q, want %q", cfg.Java.RepositoryAllowlist, want)
	}
	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/proj")
	if !slices.Equal(answers.Java.RepositoryAllowlist, want) {
		t.Errorf("answers allowlist = %q, want %q", answers.Java.RepositoryAllowlist, want)
	}
	answers.Java.RepositoryAllowlist[0] = "mutated"
	if cfg.Java.RepositoryAllowlist[0] != "confluent" {
		t.Error("ConfigToAnswers shares the allowlist slice with the config")
	}
	back := AnswersToConfig(ConfigToAnswers(cfg, types.DetectedProject{}, "/proj"), "")
	if !slices.Equal(back.Java.RepositoryAllowlist, want) {
		t.Errorf("round-tripped allowlist = %q, want %q", back.Java.RepositoryAllowlist, want)
	}
}

// TestValidateQsdevConfig_JavaRepositoryAllowlist checks that ids which would
// change the settings.xml mirrorOf expression are rejected.
func TestValidateQsdevConfig_JavaRepositoryAllowlist(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"confluent", false},
		{"spring-milestones", false},
		{"company.nexus_releases", false},
		{"a,*", true},
		{"!central", true},
		{"*", true},
		{"has space", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			cfg := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Java: types.JavaConfig{RepositoryAllowlist: []string{tt.id}}}
			errs := ValidateQsdevConfig(cfg, ValidateOptions{})
			if got := len(errs) > 0; got != tt.wantErr {
				t.Fatalf("errors = %v, wantErr %v", errs, tt.wantErr)
			}
			if tt.wantErr && errs[0].Field != "java.repository_allowlist[0]" {
				t.Errorf("error field = %q, want java.repository_allowlist[0]", errs[0].Field)
			}
		})
	}
}

// TestJavaConfig_CommittedIsAuthoritative checks the java block has a single
// source, the committed .qsdev.yaml: Apply installs it over whatever the
// saved answers held, re-creating a project keeps it, a day-2 sync leaves it
// alone, and organization defaults and the project are unioned.
func TestJavaConfig_CommittedIsAuthoritative(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{
		Version: types.ConfigVersionCurrent,
		Java:    types.JavaConfig{RepositoryAllowlist: []string{"confluent"}},
	}

	t.Run("apply", func(t *testing.T) {
		t.Parallel()
		policy, err := ResolveProjectPolicy(committed, nil)
		if err != nil {
			t.Fatal(err)
		}
		a := types.WizardAnswers{Java: types.JavaConfig{RepositoryAllowlist: []string{"stale"}}}
		policy.Apply(&a)
		if !slices.Equal(a.Java.RepositoryAllowlist, []string{"confluent"}) {
			t.Errorf("applied allowlist = %q, want [confluent]", a.Java.RepositoryAllowlist)
		}
	})

	t.Run("preserve on re-create", func(t *testing.T) {
		t.Parallel()
		fresh := types.QsdevConfig{}
		PreserveCommittedPolicy(&fresh, committed)
		if !slices.Equal(fresh.Java.RepositoryAllowlist, []string{"confluent"}) {
			t.Errorf("preserved allowlist = %q, want [confluent]", fresh.Java.RepositoryAllowlist)
		}
	})

	t.Run("sync keeps it", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, ".qsdev.yaml")
		if err := os.WriteFile(path, []byte("version: 2\ntier: standard\njava:\n  repository_allowlist: [confluent]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := SyncProjectConfig(dir, types.WizardAnswers{Tier: "full"}); err != nil {
			t.Fatal(err)
		}
		synced, err := ParseQsdevConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if synced.Tier != "full" {
			t.Fatalf("sync did not run: tier = %q", synced.Tier)
		}
		if !slices.Equal(synced.Java.RepositoryAllowlist, []string{"confluent"}) {
			t.Errorf("synced allowlist = %q, want [confluent]", synced.Java.RepositoryAllowlist)
		}
	})

	t.Run("org defaults union", func(t *testing.T) {
		t.Parallel()
		org := &types.QsdevConfig{Java: types.JavaConfig{RepositoryAllowlist: []string{"company-nexus", "confluent"}}}
		resolved, err := ResolveConfig(org, committed, nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"company-nexus", "confluent"}; !slices.Equal(resolved.Config.Java.RepositoryAllowlist, want) {
			t.Errorf("resolved allowlist = %q, want %q", resolved.Config.Java.RepositoryAllowlist, want)
		}
	})
}
