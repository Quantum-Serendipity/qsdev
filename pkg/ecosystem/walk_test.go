package ecosystem

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestProjectDirsWith(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{name: "root only", files: []string{"main.tf"}, want: []string{"."}},
		{
			name:  "monorepo layout",
			files: []string{"infra/main.tf", "infra/modules/aws/aws.tf", "README.md"},
			want:  []string{"infra", "infra/modules/aws"},
		},
		{
			name:  "hidden, vendored and dependency trees skipped",
			files: []string{".terraform/modules/x/main.tf", "vendor/a/main.tf", "node_modules/a/main.tf"},
		},
		{name: "deeper than ProjectScanDepth skipped", files: []string{"a/b/c/d/main.tf"}},
		{name: "at ProjectScanDepth found", files: []string{"a/b/c/main.tf"}, want: []string{"a/b/c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, rel := range tt.files {
				full := filepath.Join(root, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := ProjectDirsWith(root, func(name string) bool { return strings.HasSuffix(name, ".tf") })
			if !slices.Equal(got, tt.want) {
				t.Errorf("ProjectDirsWith = %v, want %v", got, tt.want)
			}
		})
	}
}
