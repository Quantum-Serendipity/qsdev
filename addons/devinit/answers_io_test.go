package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAnswersOrEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string // "" means no answers file on disk
		wantErr  string
		wantProj string
	}{
		{name: "missing file is empty state"},
		{name: "valid file", content: "project_name: demo\n", wantProj: "demo"},
		// A corrupt file must surface an error that names the file, not a bare
		// YAML error the caller cannot place.
		{name: "corrupt file", content: "languages: [unterminated\n", wantErr: answersFile()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.content != "" {
				path := answersPath(dir)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := loadAnswersOrEmpty(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ProjectName != tt.wantProj {
				t.Errorf("ProjectName = %q, want %q", got.ProjectName, tt.wantProj)
			}
		})
	}
}
