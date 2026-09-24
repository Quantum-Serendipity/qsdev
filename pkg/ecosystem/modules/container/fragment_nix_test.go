package container_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// wrapFragment embeds a module fragment the way devenv.nix.tmpl does: inside
// the top-level attrset of a devenv module that already defines env.
func wrapFragment(frag string) string {
	return "{ pkgs, lib, config, ... }:\n{\n  env = { EXISTING = \"1\"; };\n" + frag + "}\n"
}

// TestDevenvNixFragment_PodmanSocketPaths pins the socket each Podman mode
// targets and guards against the raw ${XDG_RUNTIME_DIR} antiquotation, which
// references an undefined Nix variable and breaks devenv.nix evaluation.
func TestDevenvNixFragment_PodmanSocketPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		runtime string
		want    string
	}{
		{"podman-rootless", `"unix://${xdgRuntimeDir}/podman/podman.sock"`},
		{"podman-rootful", `"unix:///run/podman/podman.sock"`},
	}
	for _, tt := range tests {
		t.Run(tt.runtime, func(t *testing.T) {
			t.Parallel()
			frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{
				Extras: map[string]string{"container_runtime": tt.runtime},
			})
			if err != nil {
				t.Fatalf("DevenvNixFragment error: %v", err)
			}
			if !strings.Contains(frag, tt.want) {
				t.Errorf("fragment missing %s:\n%s", tt.want, frag)
			}
			if strings.Contains(frag, "${XDG_RUNTIME_DIR}") {
				t.Errorf("fragment antiquotes undefined Nix variable XDG_RUNTIME_DIR:\n%s", frag)
			}
		})
	}
}

// TestDevenvNixFragment_NixParses parses every runtime variant with
// nix-instantiate (when available), catching undefined-variable and syntax
// errors that would make the generated devenv.nix unevaluable.
func TestDevenvNixFragment_NixParses(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}

	for _, rt := range []string{"", "docker", "podman-rootless", "podman-rootful"} {
		t.Run("runtime="+rt, func(t *testing.T) {
			t.Parallel()
			frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{
				Extras: map[string]string{"container_runtime": rt},
			})
			if err != nil {
				t.Fatalf("DevenvNixFragment error: %v", err)
			}
			out, err := exec.CommandContext(t.Context(), nixInstantiate, "--parse", "-E", wrapFragment(frag)).CombinedOutput()
			if err != nil {
				t.Fatalf("nix-instantiate --parse failed: %v\n%s\nfragment:\n%s", err, out, frag)
			}
		})
	}
}

// TestDevenvNixFragment_RootlessDockerHostEval evaluates the rootless
// DOCKER_HOST expression (when nix-instantiate is available) to confirm it
// resolves the runtime directory at evaluation time and keeps the user's
// DOCKER_HOST when no runtime directory exists.
func TestDevenvNixFragment_RootlessDockerHostEval(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{
		Extras: map[string]string{"container_runtime": "podman-rootless"},
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	expr := "(" + wrapFragment(frag) + ") { pkgs = {}; lib = {}; config = {}; }"

	tests := []struct {
		name string
		env  []string
		want string
	}{
		{"runtime dir set", []string{"XDG_RUNTIME_DIR=/run/user/4242", "DOCKER_HOST="}, `"unix:///run/user/4242/podman/podman.sock"`},
		{"no runtime dir keeps user value", []string{"XDG_RUNTIME_DIR=", "DOCKER_HOST=tcp://127.0.0.1:2375"}, `"tcp://127.0.0.1:2375"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.CommandContext(t.Context(), nixInstantiate, "--eval", "--strict", "-E", "("+expr+").env.DOCKER_HOST")
			cmd.Env = append(os.Environ(), tt.env...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("nix-instantiate --eval failed: %v\n%s", err, out)
			}
			if got := strings.TrimSpace(string(out)); got != tt.want {
				t.Errorf("DOCKER_HOST = %s, want %s", got, tt.want)
			}
		})
	}
}
