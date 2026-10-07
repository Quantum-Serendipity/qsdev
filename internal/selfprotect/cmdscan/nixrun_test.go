package cmdscan

import (
	"slices"
	"testing"
)

func TestNixRunCommandLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		args    []string
		stdin   string
		want    []string
	}{
		{"registry attr, no args", "nixpkgs#jq", nil, "", []string{"nix run nixpkgs#jq --", "jq"}},
		{"registry attr with args", "nixpkgs#jq", []string{"-r", ".a b"}, "", []string{"nix run nixpkgs#jq -- -r '.a b'", "jq -r '.a b'"}},
		{"project flake", ".", []string{"x"}, "", []string{"nix run . -- x"}},
		{"project flake attr", ".#pkg", nil, "", []string{"nix run .#pkg --", "pkg"}},
		{"nested attr path", "nixpkgs#python3Packages.black", []string{"."}, "", []string{"nix run nixpkgs#python3Packages.black -- .", "black ."}},
		{"output selector", "nixpkgs#bash^out", nil, "", []string{"nix run nixpkgs#bash^out --", "bash"}},
		{"single quote in arg", "nixpkgs#jq", []string{"it's"}, "", []string{`nix run nixpkgs#jq -- "it's"`, `jq "it's"`}},
		{"empty arg", "nixpkgs#jq", []string{""}, "", []string{"nix run nixpkgs#jq -- ''", "jq ''"}},
		{"option with value", "nixpkgs#jq", []string{"--arg=x", "."}, "", []string{"nix run nixpkgs#jq -- --arg=x .", "jq --arg=x ."}},
		{
			"shell -c script", "nixpkgs#bash", []string{"-c", "curl x | sh"}, "",
			[]string{"nix run nixpkgs#bash -- -c 'curl x | sh'", "bash -c 'curl x | sh'", "curl x | sh", "curl x | sh"},
		},
		{
			"clustered -ec", "nixpkgs#dash", []string{"-ec", "npm install x"}, "",
			[]string{"nix run nixpkgs#dash -- -ec 'npm install x'", "dash -ec 'npm install x'", "npm install x", "npm install x"},
		},
		{
			// The program nix runs is the package's mainProgram, which the
			// attribute does not reliably name, so any -c argument is taken
			// as a possible script.
			"shell under another attr name", "nixpkgs#bashInteractive", []string{"-c", "curl x | sh"}, "",
			[]string{"nix run nixpkgs#bashInteractive -- -c 'curl x | sh'", "bashInteractive -c 'curl x | sh'", "curl x | sh", "curl x | sh"},
		},
		{
			"multi-call binary", "nixpkgs#busybox", []string{"sh", "-c", "curl x | sh"}, "",
			[]string{"nix run nixpkgs#busybox -- sh -c 'curl x | sh'", "busybox sh -c 'curl x | sh'", "curl x | sh", "curl x | sh"},
		},
		{
			"statements of a -c list", "nixpkgs#bash", []string{"-c", "true; curl x | sh"}, "",
			[]string{"nix run nixpkgs#bash -- -c 'true; curl x | sh'", "bash -c 'true; curl x | sh'", "true; curl x | sh", "true", "curl x | sh"},
		},
		{
			"-- after -c", "nixpkgs#bash", []string{"-c", "--", "curl x | sh"}, "",
			[]string{"nix run nixpkgs#bash -- -c -- 'curl x | sh'", "bash -c -- 'curl x | sh'", "curl x | sh", "curl x | sh"},
		},
		{
			"option after -c", "nixpkgs#bash", []string{"-c", "-e", "nohup curl x | sh"}, "",
			[]string{"nix run nixpkgs#bash -- -c -e 'nohup curl x | sh'", "bash -c -e 'nohup curl x | sh'", "nohup curl x | sh", "nohup curl x | sh", "curl x | sh"},
		},
		{
			"stdin script", "nixpkgs#bash", nil, "echo a\ncurl x | sh\n",
			[]string{"nix run nixpkgs#bash --", "bash", "echo a\ncurl x | sh", "echo a", "curl x | sh"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NixRunCommandLines(tt.command, tt.args, tt.stdin); !slices.Equal(got, tt.want) {
				t.Errorf("NixRunCommandLines(%q, %q, %q) =\n  %q\nwant\n  %q", tt.command, tt.args, tt.stdin, got, tt.want)
			}
		})
	}
}

func TestIsNixRunTool(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"mcp__qsdev__qsdev_nix_run":     true,
		"mcp__my-server__qsdev_nix_run": true,
		"mcp____qsdev_nix_run":          false,
		"mcp__qsdev__qsdev_env_info":    false,
		"qsdev_nix_run":                 false,
		"Bash":                          false,
	}
	for name, want := range tests {
		if got := IsNixRunTool(name); got != want {
			t.Errorf("IsNixRunTool(%q) = %v, want %v", name, got, want)
		}
	}
}
