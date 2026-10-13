package devenv

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func TestLanguageSpecValidate(t *testing.T) {
	t.Parallel()

	validate := languageSpec(true).validate
	tests := []struct {
		name    string
		lang    string
		wantErr bool
	}{
		{"core language", "go", false},
		{"cloud module aws", "aws", false},
		{"cloud module gcp", "gcp", false},
		{"cloud module azure", "azure", false},
		{"unknown", "bogus", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validate(tt.lang, "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate(%q) error = %v, wantErr %v", tt.lang, err, tt.wantErr)
			}
			if err == nil {
				return
			}
			// The error must list every registered ecosystem module so the
			// user sees the full set of accepted names.
			msg := err.Error()
			open, end := strings.LastIndex(msg, "["), strings.LastIndex(msg, "]")
			if open < 0 || end < open {
				t.Fatalf("error %q has no bracketed language list", msg)
			}
			listed := strings.Fields(msg[open+1 : end])
			for _, name := range ecosystem.DefaultRegistry().Names() {
				if !slices.Contains(listed, name) {
					t.Errorf("error %q does not list registered language %q", msg, name)
				}
			}
		})
	}
}
