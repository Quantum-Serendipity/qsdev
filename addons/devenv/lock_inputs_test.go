package devenv_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
)

// TestUnlockedInputs checks the inputs devenv.yaml declares are compared
// against the root inputs devenv.lock pins.
func TestUnlockedInputs(t *testing.T) {
	t.Parallel()
	const yml = "inputs:\n  nixpkgs:\n    url: github:NixOS/nixpkgs/nixpkgs-unstable\n" +
		"  git-hooks:\n    url: github:cachix/git-hooks.nix\n  go-overlay:\n    url: github:purpleclay/go-overlay\n"
	const lock = `{"root": "root", "nodes": {"root": {"inputs": {"devenv": "devenv", "git-hooks": "git-hooks", "nixpkgs": "nixpkgs"}}}, "version": 7}`
	tests := []struct {
		name    string
		lock    string // "" means no devenv.lock
		want    []string
		wantErr bool
	}{
		{"new input not locked", lock, []string{"go-overlay"}, false},
		{"no lock file yet", "", nil, false},
		{"unparseable lock", "{", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.lock != "" {
				if err := os.WriteFile(filepath.Join(dir, "devenv.lock"), []byte(tt.lock), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := devenv.UnlockedInputs(dir, []byte(yml))
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnlockedInputs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("UnlockedInputs() = %v, want %v", got, tt.want)
			}
		})
	}
}
