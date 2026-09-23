package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRoot(t *testing.T) {
	// Not parallel: t.Chdir changes the process working directory.
	tests := []struct {
		name    string
		markers []string // relative paths to create; a trailing "/" makes a directory
		cwd     string   // relative to the temp root
		want    string   // relative to the temp root
	}{
		{name: "no project falls back to cwd", cwd: "a/b", want: "a/b"},
		{name: "config file at cwd", markers: []string{".qsdev.yaml"}, cwd: ".", want: "."},
		{name: "config file in ancestor", markers: []string{".qsdev.yaml"}, cwd: "src/pkg", want: "."},
		{name: "state dir in ancestor", markers: []string{".devinit/"}, cwd: "src", want: "."},
		{name: "nearest project wins", markers: []string{".qsdev.yaml", "sub/.qsdev.yaml"}, cwd: "sub/x", want: "sub"},
		{name: "config name as directory is not a marker", markers: []string{"src/.qsdev.yaml/"}, cwd: "src", want: "src"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range tt.markers {
				p := filepath.Join(root, filepath.FromSlash(m))
				if m[len(m)-1] == '/' {
					if err := os.MkdirAll(p, 0o755); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			cwd := filepath.Join(root, filepath.FromSlash(tt.cwd))
			if err := os.MkdirAll(cwd, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(cwd)

			got, err := ProjectRoot()
			if err != nil {
				t.Fatalf("ProjectRoot() error = %v", err)
			}
			want := filepath.Join(root, filepath.FromSlash(tt.want))
			if got != want {
				t.Errorf("ProjectRoot() = %q, want %q", got, want)
			}
		})
	}
}

func TestWorkingDir(t *testing.T) {
	got, err := WorkingDir()
	if err != nil {
		t.Fatalf("WorkingDir() error = %v", err)
	}
	want, _ := os.Getwd()
	if got != want {
		t.Errorf("WorkingDir() = %q, want %q", got, want)
	}
}
