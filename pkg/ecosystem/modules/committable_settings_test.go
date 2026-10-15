package modules

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

const (
	configReference       = "../../../docs/configuration-reference.md"
	committableTableBegin = "<!-- committable-settings:begin -->\n"
	committableTableEnd   = "<!-- committable-settings:end -->"
)

// TestCommittableSettings_OnlyOptedIn pins the committed surface: a wizard
// field is committable only when it opts in, and today only go_source does.
// A wizard field that decides which controls run must never be committable:
// committing container trusted_registries replaced hadolint's allowlist, a
// scala build_tool of mill on an sbt project dropped its CI lock check, OSV
// scan and proxy config, and a clojure build_tool of leiningen dropped the
// clj-watson step, each with qsdev check green.
func TestCommittableSettings_OnlyOptedIn(t *testing.T) {
	t.Parallel()
	c := ecosystem.DefaultRegistry().CommittableSettings()
	got := map[string][]string{}
	for lang, fields := range c {
		for _, f := range fields {
			got[lang] = append(got[lang], f.Key)
		}
	}
	want := map[string][]string{"go": {"go_source"}}
	if !maps.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("committable settings = %v, want %v", got, want)
	}
	for _, tt := range []struct{ lang, extra string }{
		{"container", "trusted_registries=evil.example.com"},
		{"scala", "build_tool=mill"},
		{"clojure", "build_tool=leiningen"},
	} {
		errs := c.CheckExtras(tt.lang, []string{tt.extra})
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "is not a committable setting") {
			t.Errorf("%s extra %s: errors = %v, want it rejected as not a committable setting", tt.lang, tt.extra, errs)
		}
	}
}

// TestCommittableSettingsDocumented checks the configuration reference's
// table of committable language extras is exactly the registered modules'
// committable settings (the allowlist ValidateQsdevConfig enforces), rendered
// by committableSettingsTable.
func TestCommittableSettingsDocumented(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(configReference)
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(data), committableTableBegin)
	got, _, ok2 := strings.Cut(rest, committableTableEnd)
	if !ok || !ok2 {
		t.Fatalf("%s lacks the %q ... %q markers", configReference, committableTableBegin, committableTableEnd)
	}
	if want := committableSettingsTable(ecosystem.DefaultRegistry().CommittableSettings()); got != want {
		t.Errorf("committable settings table is stale; replace it with:\n%s", want)
	}
}

// committableSettingsTable renders c as a Markdown table, by language and
// then in each module's declaration order.
func committableSettingsTable(c ecosystem.CommittableSettings) string {
	var b strings.Builder
	b.WriteString("| Language | Extra | Values |\n|---|---|---|\n")
	for _, lang := range slices.Sorted(maps.Keys(c)) {
		for _, f := range c[lang] {
			fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", lang, f.Key, settingValues(f))
		}
	}
	return b.String()
}

func settingValues(f ecosystem.WizardField) string {
	mark := func(v string) string {
		if v == f.Default {
			return "`" + v + "` (default)"
		}
		return "`" + v + "`"
	}
	var values []string
	switch f.Type {
	case ecosystem.FieldTypeSelect, ecosystem.FieldTypeMultiSelect:
		for _, o := range f.Options {
			values = append(values, mark(o.Value))
		}
	case ecosystem.FieldTypeConfirm:
		values = []string{mark("true"), mark("false")}
	default:
		return "a single word, such as `" + f.Placeholder + "`"
	}
	if f.Type == ecosystem.FieldTypeMultiSelect {
		return "comma-separated subset of " + strings.Join(values, ", ")
	}
	return strings.Join(values, ", ")
}
