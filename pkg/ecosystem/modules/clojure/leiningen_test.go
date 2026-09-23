package clojure_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestDevenvPackageExprs_Leiningen guards the Leiningen environment: devenv's
// languages.clojure ships only the clojure CLI, so a project.clj project must
// get lein itself, built against the project JDK.
func TestDevenvPackageExprs_Leiningen(t *testing.T) {
	t.Parallel()

	const lein = "(pkgs.leiningen.override { jdk = config.languages.java.jdk.package; })"
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{name: "project.clj only", files: []string{"project.clj"}, want: []string{lein}},
		{name: "deps.edn", files: []string{"deps.edn"}},
		{name: "both prefers tools.deps", files: []string{"deps.edn", "project.clj"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			m := newModule()
			var provider ecosystem.PackageExprProvider = m
			got := provider.DevenvPackageExprs(m.Detect(dir).SuggestedConfig)
			if !slices.Equal(got, tt.want) {
				t.Errorf("DevenvPackageExprs = %v, want %v", got, tt.want)
			}
		})
	}
}
