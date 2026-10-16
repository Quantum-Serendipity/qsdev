package catalog

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// The built-in catalog exempts vendored trees from ripsecrets, and from no
// other always-on hook.
func TestSecurityHookExcludes_Embedded(t *testing.T) {
	t.Parallel()

	cat := loadTestCatalog(t)
	if got, want := cat.SecurityHookExcludes("ripsecrets"), []string{`(^|/)vendor/`}; !slices.Equal(got, want) {
		t.Errorf(`SecurityHookExcludes("ripsecrets") = %q, want %q`, got, want)
	}
	for _, id := range cat.SecurityHooks() {
		if id != "ripsecrets" && len(cat.SecurityHookExcludes(id)) > 0 {
			t.Errorf("SecurityHookExcludes(%q) = %q, want none", id, cat.SecurityHookExcludes(id))
		}
	}
}

// The accessor returns a copy: a caller cannot widen the catalog's exemptions.
func TestSecurityHookExcludes_ReturnsCopy(t *testing.T) {
	t.Parallel()

	cat := loadTestCatalog(t)
	got := cat.SecurityHookExcludes("ripsecrets")
	got[0] = ".*"
	if again := cat.SecurityHookExcludes("ripsecrets"); slices.Contains(again, ".*") {
		t.Errorf("SecurityHookExcludes() aliases catalog state: %q", again)
	}
}

// An org defaults file cannot exempt paths from an always-on hook: that
// lowers the floor. Restating a built-in exemption is not a change.
func TestLoad_OrgOverlayCannotExemptAlwaysOnHook(t *testing.T) {
	t.Parallel()

	for _, content := range []string{
		"security_hook_excludes:\n  ripsecrets: [\"(^|/)src/\"]\n",
		"security_hook_excludes:\n  shellcheck: [\".*\"]\n",
	} {
		_, err := Load(WithOrgConfigFile(writeUnifiedFile(t, content)))
		if !errors.Is(err, ErrOverlayLoosens) {
			t.Errorf("Load(org %q) error = %v, want ErrOverlayLoosens", content, err)
			continue
		}
		if !strings.Contains(err.Error(), "security_hook_excludes") {
			t.Errorf("error = %v, want it to name security_hook_excludes", err)
		}
	}

	cat, err := Load(WithOrgConfigFile(writeUnifiedFile(t, "security_hook_excludes:\n  ripsecrets: [\"(^|/)vendor/\"]\n")))
	if err != nil {
		t.Fatalf("Load(org restating the built-in exemption) error = %v", err)
	}
	if got := cat.SecurityHookExcludes("ripsecrets"); !slices.Equal(got, []string{`(^|/)vendor/`}) {
		t.Errorf(`SecurityHookExcludes("ripsecrets") = %q, want the built-in exemption once`, got)
	}
}

// A committed project defaults file cannot set the section at all.
func TestProjectOverlayRejectsSecurityHookExcludes(t *testing.T) {
	t.Parallel()

	_, err := loadProject(t, "security_hook_excludes:\n  ripsecrets: [\"(^|/)src/\"]\n")
	if !errors.Is(err, ErrProjectOverlayRejected) {
		t.Fatalf("Load(project) error = %v, want ErrProjectOverlayRejected", err)
	}
	if !strings.Contains(err.Error(), "security_hook_excludes") {
		t.Errorf("error = %v, want it to name security_hook_excludes", err)
	}
}

// Validate reports an exemption for a hook that is not always on, and one
// that is not a regular expression.
func TestValidate_SecurityHookExcludes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		excludes map[string][]string
		want     string
	}{
		{"unknown hook", map[string][]string{"no-such-hook": {"^x/"}}, `"no-such-hook" is not in security_hooks`},
		{"invalid regex", map[string][]string{"ripsecrets": {"("}}, `invalid exclude "("`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat := loadTestCatalog(t)
			cat.security.Hooks.Excludes = tt.excludes
			for _, e := range cat.Validate() {
				if strings.HasPrefix(e.Field, "security_hook_excludes.") && strings.Contains(e.Message, tt.want) {
					return
				}
			}
			t.Errorf("Validate() = %v, want a security_hook_excludes error containing %q", cat.Validate(), tt.want)
		})
	}
}
