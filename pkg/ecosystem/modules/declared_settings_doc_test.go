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
	configReference    = "../../../docs/configuration-reference.md"
	declaredTableBegin = "<!-- declared-settings:begin -->\n"
	declaredTableEnd   = "<!-- declared-settings:end -->"
)

// TestDeclaredSettingsDocumented checks the configuration reference's table
// of committable language extras is exactly the settings the registered
// modules declare (the allowlist ValidateQsdevConfig enforces), rendered by
// declaredSettingsTable.
func TestDeclaredSettingsDocumented(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(configReference)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	_, rest, ok := strings.Cut(doc, declaredTableBegin)
	got, _, ok2 := strings.Cut(rest, declaredTableEnd)
	if !ok || !ok2 {
		t.Fatalf("%s lacks the %q ... %q markers", configReference, declaredTableBegin, declaredTableEnd)
	}
	if want := declaredSettingsTable(ecosystem.DefaultRegistry().DeclaredSettings()); got != want {
		t.Errorf("declared settings table is stale; replace it with:\n%s", want)
	}
}

// declaredSettingsTable renders d as a Markdown table, by language and then
// in each module's declaration order.
func declaredSettingsTable(d ecosystem.DeclaredSettings) string {
	var b strings.Builder
	b.WriteString("| Language | Extra | Values |\n|---|---|---|\n")
	for _, lang := range slices.Sorted(maps.Keys(d)) {
		for _, f := range d[lang] {
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
		return "comma-separated words, such as `" + f.Placeholder + "`"
	}
	if f.Type == ecosystem.FieldTypeMultiSelect {
		return "comma-separated subset of " + strings.Join(values, ", ")
	}
	return strings.Join(values, ", ")
}
