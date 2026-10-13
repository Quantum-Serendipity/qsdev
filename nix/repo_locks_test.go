package nix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// repoRoot is the repository root relative to this package's directory, which
// is the working directory `go test` runs the package's tests in.
const repoRoot = ".."

// flakeLock is the subset of a flake.lock (and devenv.lock, which shares the
// format) these tests read.
type flakeLock struct {
	Root  string `json:"root"`
	Nodes map[string]struct {
		Inputs map[string]any `json:"inputs"`
		Locked map[string]any `json:"locked"`
	} `json:"nodes"`
}

func readLock(t *testing.T, name string) flakeLock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	var lock flakeLock
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	return lock
}

// nixpkgsLocked returns the locked entry of the root's nixpkgs input. It also
// requires that input to resolve to the node named "nixpkgs", because the
// dependabot-fixup workflow's jq sync copies .nodes.nixpkgs.locked by that name.
func nixpkgsLocked(t *testing.T, name string) map[string]any {
	t.Helper()
	lock := readLock(t, name)
	root, ok := lock.Nodes[lock.Root]
	if !ok {
		t.Fatalf("%s: root node %q missing", name, lock.Root)
	}
	if got := root.Inputs["nixpkgs"]; got != "nixpkgs" {
		t.Fatalf("%s: root input nixpkgs resolves to %v, want node \"nixpkgs\" "+
			"(.github/workflows/dependabot-fixup.yml syncs .nodes.nixpkgs)", name, got)
	}
	locked := lock.Nodes["nixpkgs"].Locked
	if len(locked) == 0 {
		t.Fatalf("%s: node nixpkgs has no locked entry", name)
	}
	return locked
}

// TestNixpkgsLocksAgree guards the invariant the dependabot-fixup workflow
// maintains: flake.lock (bumped by Dependabot) and devenv.lock (the dev
// environment) pin the same nixpkgs.
func TestNixpkgsLocksAgree(t *testing.T) {
	t.Parallel()
	flake := nixpkgsLocked(t, "flake.lock")
	devenv := nixpkgsLocked(t, "devenv.lock")
	for _, key := range []string{"rev", "narHash"} {
		if flake[key] == nil || flake[key] != devenv[key] {
			t.Errorf("nixpkgs %s: flake.lock=%v devenv.lock=%v", key, flake[key], devenv[key])
		}
	}
	if !reflect.DeepEqual(flake, devenv) {
		t.Errorf("nixpkgs locked entries differ:\n flake.lock:  %v\n devenv.lock: %v\n"+
			"copy flake.lock's .nodes.nixpkgs.locked into devenv.lock", flake, devenv)
	}
}

var (
	nixBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	nixLineComment  = regexp.MustCompile(`(?m)#.*$`)
)

// TestFlakeHasNoDevShells pins `devenv shell` as the repository's only
// development environment: flake.nix must not define devShells, which would
// be a second, unhardened shell outside devenv.lock's pins.
func TestFlakeHasNoDevShells(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot, "flake.nix"))
	if err != nil {
		t.Fatalf("reading flake.nix: %v", err)
	}
	code := nixLineComment.ReplaceAllString(nixBlockComment.ReplaceAllString(string(data), ""), "")
	if strings.Contains(code, "devShells") {
		t.Error("flake.nix defines devShells; `devenv shell` is the only dev environment " +
			"(put dev tools in devenv.nix)")
	}
}
