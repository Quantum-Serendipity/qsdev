package devenv

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDeclaredEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want map[string]string
	}{
		{
			name: "env block",
			src:  "{ pkgs, ... }:\n{\n  env = {\n    AWS_PROFILE = \"dev\";\n    QUOTED = \"a \\\"b\\\"\";\n  };\n}\n",
			want: map[string]string{"AWS_PROFILE": "dev", "QUOTED": `a "b"`},
		},
		{
			name: "dotted binding",
			src:  "{ pkgs, ... }:\n{\n  env.CLOUDSDK_ACTIVE_CONFIG_NAME = \"proj\";\n}\n",
			want: map[string]string{"CLOUDSDK_ACTIVE_CONFIG_NAME": "proj"},
		},
		{
			name: "commented guidance is not a declaration",
			src:  "{ pkgs, ... }:\n{\n  # env.ARM_SUBSCRIPTION_ID = \"<subscription>\";\n  packages = [ ];\n}\n",
			want: map[string]string{},
		},
		{
			name: "non-literal values keep their source text",
			src:  "{ lib, ... }:\n{\n  env.A = lib.mkForce \"x\";\n  env.B = \"${HOME}/x\";\n}\n",
			want: map[string]string{"A": `lib.mkForce "x"`, "B": `"${HOME}/x"`},
		},
		{
			name: "module without an argument set",
			src:  "{\n  env.AWS_PROFILE = \"local\";\n}\n",
			want: map[string]string{"AWS_PROFILE": "local"},
		},
		{
			name: "other attributes are ignored",
			src:  "{ pkgs, ... }:\n{\n  languages.go.enable = true;\n  envx.A = \"1\";\n}\n",
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := DeclaredEnv(tt.src)
			if err != nil {
				t.Fatalf("DeclaredEnv: %v", err)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("DeclaredEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeclaredEnv_InvalidNix(t *testing.T) {
	t.Parallel()

	if _, err := DeclaredEnv("{ pkgs, ... }: { env.A = ; "); err == nil {
		t.Error("DeclaredEnv accepted malformed Nix")
	}
}

func TestProjectDeclaredEnv(t *testing.T) {
	t.Parallel()

	writeFile := func(t *testing.T, dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("local file wins", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, "devenv.nix", "{ pkgs, ... }:\n{\n  env = {\n    AWS_PROFILE = \"shared\";\n    OTHER = \"1\";\n  };\n}\n")
		writeFile(t, dir, DevenvLocalNixFile, "{ ... }:\n{\n  env.AWS_PROFILE = \"mine\";\n}\n")
		got, err := ProjectDeclaredEnv(dir)
		if err != nil {
			t.Fatalf("ProjectDeclaredEnv: %v", err)
		}
		want := map[string]string{"AWS_PROFILE": "mine", "OTHER": "1"}
		if !maps.Equal(got, want) {
			t.Errorf("ProjectDeclaredEnv() = %v, want %v", got, want)
		}
	})

	t.Run("no files", func(t *testing.T) {
		t.Parallel()
		got, err := ProjectDeclaredEnv(t.TempDir())
		if err != nil || len(got) != 0 {
			t.Errorf("ProjectDeclaredEnv() = %v, %v; want empty, nil", got, err)
		}
	})

	t.Run("unparsable file is reported, the other still read", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFile(t, dir, "devenv.nix", "{ pkgs, ... }:\n{\n  env.A = \"1\";\n}\n")
		writeFile(t, dir, DevenvLocalNixFile, "not nix at all")
		got, err := ProjectDeclaredEnv(dir)
		if err == nil {
			t.Error("ProjectDeclaredEnv did not report the unparsable file")
		}
		if got["A"] != "1" {
			t.Errorf("ProjectDeclaredEnv() = %v, want A=1 from devenv.nix", got)
		}
	})
}

// generatedSecurityNix is the shape devenv.nix.tmpl renders the always-on
// hooks and stripped variables in.
const generatedSecurityNix = `{ pkgs, lib, config, options, ... }:
{
  env = {
    QSDEV_SECURITY_PROFILE = "strict";
  };

  # Credential-bearing variables stripped from the shell.
  unsetEnvVars = options.unsetEnvVars.default ++ [ "AWS_SECRET_ACCESS_KEY" "GITHUB_TOKEN" ];

  git-hooks.hooks = {
    # Always-on hooks
    ripsecrets.enable = true;
    check-added-large-files.enable = true;
    gofmt = {
      enable = true;
      excludes = [ "vendor/" ];
    };
    lock-file-audit = {
      enable = true;
      name = "Lock file change audit";
      entry = "echo \"# not a comment\"";
      pass_filenames = true;
    };
  };
}
`

func TestDeclaredSecurity(t *testing.T) {
	t.Parallel()

	allHooks := []string{"check-added-large-files", "gofmt", "lock-file-audit", "ripsecrets"}
	allVars := []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN"}
	tests := []struct {
		name      string
		nix       string
		local     string // devenv.local.nix; "" writes none
		wantHooks []string
		wantVars  []string
	}{
		{
			name:      "generated form",
			nix:       generatedSecurityNix,
			wantHooks: allHooks,
			wantVars:  allVars,
		},
		{
			name:      "CRLF line endings",
			nix:       strings.ReplaceAll(generatedSecurityNix, "\n", "\r\n"),
			wantHooks: allHooks,
			wantVars:  allVars,
		},
		{
			name:      "enable = false is not enabled",
			nix:       strings.Replace(generatedSecurityNix, "ripsecrets.enable = true;", "ripsecrets.enable = false;", 1),
			wantHooks: []string{"check-added-large-files", "gofmt", "lock-file-audit"},
			wantVars:  allVars,
		},
		{
			name:      "mkForce true is enabled",
			nix:       strings.Replace(generatedSecurityNix, "ripsecrets.enable = true;", "ripsecrets.enable = lib.mkForce true;", 1),
			wantHooks: allHooks,
			wantVars:  allVars,
		},
		{
			name:      "a deleted variable is not stripped",
			nix:       strings.Replace(generatedSecurityNix, `"AWS_SECRET_ACCESS_KEY" `, "", 1),
			wantHooks: allHooks,
			wantVars:  []string{"GITHUB_TOKEN"},
		},
		{
			name:      "a commented-out variable is not stripped",
			nix:       strings.Replace(generatedSecurityNix, `[ "AWS_SECRET_ACCESS_KEY" "GITHUB_TOKEN" ]`, "[\n    # \"AWS_SECRET_ACCESS_KEY\"\n    /* \"GITHUB_TOKEN\" */\n  ]", 1),
			wantHooks: allHooks,
			wantVars:  []string{},
		},
		{
			name:      "local mkForce false disables a hook",
			nix:       generatedSecurityNix,
			local:     "{ lib, ... }:\n{\n  git-hooks.hooks.ripsecrets.enable = lib.mkForce false;\n}\n",
			wantHooks: []string{"check-added-large-files", "gofmt", "lock-file-audit"},
			wantVars:  allVars,
		},
		{
			name:      "local plain false disables a hook",
			nix:       generatedSecurityNix,
			local:     "{\r\n  git-hooks.hooks.gofmt.enable = false;\r\n}\r\n",
			wantHooks: []string{"check-added-large-files", "lock-file-audit", "ripsecrets"},
			wantVars:  allVars,
		},
		{
			name:      "local true keeps a hook",
			nix:       generatedSecurityNix,
			local:     "{ ... }:\n{\n  git-hooks.hooks.ripsecrets.enable = true;\n}\n",
			wantHooks: allHooks,
			wantVars:  allVars,
		},
		{
			name:      "local mkForce unsetEnvVars replaces the list",
			nix:       generatedSecurityNix,
			local:     "{ lib, ... }:\n{\n  unsetEnvVars = lib.mkForce [ \"GITHUB_TOKEN\" ];\n}\n",
			wantHooks: allHooks,
			wantVars:  []string{"GITHUB_TOKEN"},
		},
		{
			name:      "local mkOverride unsetEnvVars replaces the list",
			nix:       generatedSecurityNix,
			local:     "{ lib, ... }:\n{\n  unsetEnvVars = lib.mkOverride 10 [ ];\n}\n",
			wantHooks: allHooks,
			wantVars:  []string{},
		},
		{
			name:      "local plain unsetEnvVars adds to the list",
			nix:       generatedSecurityNix,
			local:     "{ ... }:\n{\n  unsetEnvVars = [ \"MY_TEAM_TOKEN\" ];\n}\n",
			wantHooks: allHooks,
			wantVars:  []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN", "MY_TEAM_TOKEN"},
		},
		{
			name:      "no devenv.nix declares nothing",
			wantHooks: []string{},
			wantVars:  []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.nix != "" {
				writeNixFile(t, filepath.Join(dir, "devenv.nix"), tt.nix)
			}
			if tt.local != "" {
				writeNixFile(t, filepath.Join(dir, DevenvLocalNixFile), tt.local)
			}
			declared, err := ProjectDeclaredSecurity(dir)
			if err != nil {
				t.Fatalf("ProjectDeclaredSecurity: %v", err)
			}
			hooks, vars := declared.Hooks, declared.UnsetVars
			if !slices.Equal(hooks, tt.wantHooks) {
				t.Errorf("hooks = %v, want %v", hooks, tt.wantHooks)
			}
			if !slices.Equal(vars, tt.wantVars) {
				t.Errorf("unset vars = %v, want %v", vars, tt.wantVars)
			}
			if tt.local != "" || tt.nix == "" {
				return
			}
			declared, err = DeclaredSecurity(tt.nix)
			if err != nil {
				t.Fatalf("DeclaredSecurity: %v", err)
			}
			hooks, vars = declared.Hooks, declared.UnsetVars
			if !slices.Equal(hooks, tt.wantHooks) || !slices.Equal(vars, tt.wantVars) {
				t.Errorf("DeclaredSecurity() = %v, %v; want %v, %v", hooks, vars, tt.wantHooks, tt.wantVars)
			}
		})
	}
}

func TestProjectDeclaredSecurity_UnparsableFails(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"devenv.nix", DevenvLocalNixFile} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeNixFile(t, filepath.Join(dir, "devenv.nix"), generatedSecurityNix)
			writeNixFile(t, filepath.Join(dir, name), "not nix at all")
			if _, err := ProjectDeclaredSecurity(dir); err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("ProjectDeclaredSecurity error = %v, want one naming %s", err, name)
			}
		})
	}
}

// A module whose security settings the static reader cannot see (imported
// modules, an explicit config attribute, interpolated names, a computed
// unsetEnvVars) fails rather than passing on what it can see.
func TestProjectDeclaredSecurity_UnverifiableFails(t *testing.T) {
	t.Parallel()

	const weakModule = "{ lib, ... }: { git-hooks.hooks.ripsecrets.enable = lib.mkForce false; unsetEnvVars = lib.mkForce [ ]; }\n"
	tests := []struct {
		name  string
		nix   string
		local string
	}{
		{
			name: "devenv.nix imports a module",
			nix:  strings.Replace(generatedSecurityNix, "{\n  env = {", "{\n  imports = [ ./weak.nix ];\n  env = {", 1),
		},
		{
			name:  "devenv.local.nix imports a module",
			nix:   generatedSecurityNix,
			local: "{ ... }:\n{\n  imports = [ ./weak.nix ];\n}\n",
		},
		{
			name:  "explicit config attribute",
			nix:   generatedSecurityNix,
			local: "{ lib, ... }:\n{\n  config.git-hooks.hooks.ripsecrets.enable = lib.mkForce false;\n}\n",
		},
		{
			name:  "interpolated attribute name",
			nix:   generatedSecurityNix,
			local: "{ lib, ... }:\n{\n  ${\"git-hooks\"}.hooks.ripsecrets.enable = lib.mkForce false;\n}\n",
		},
		{
			name: "filtered unsetEnvVars",
			nix: strings.Replace(generatedSecurityNix, `options.unsetEnvVars.default ++ [ "AWS_SECRET_ACCESS_KEY" "GITHUB_TOKEN" ]`,
				`builtins.filter (v: v != "AWS_SECRET_ACCESS_KEY") (options.unsetEnvVars.default ++ [ "AWS_SECRET_ACCESS_KEY" "GITHUB_TOKEN" ])`, 1),
		},
		{
			name:  "unsetEnvVars through a let binding",
			nix:   generatedSecurityNix,
			local: "{ lib, ... }:\n{\n  unsetEnvVars = lib.mkForce (let keep = [ ]; in keep);\n}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeNixFile(t, filepath.Join(dir, "devenv.nix"), tt.nix)
			writeNixFile(t, filepath.Join(dir, "weak.nix"), weakModule)
			if tt.local != "" {
				writeNixFile(t, filepath.Join(dir, DevenvLocalNixFile), tt.local)
			}
			if _, err := ProjectDeclaredSecurity(dir); !errors.Is(err, ErrUnverifiableModule) {
				t.Errorf("ProjectDeclaredSecurity error = %v, want ErrUnverifiableModule", err)
			}
		})
	}
}

// A hook neutralised without touching its enable (a no-op entry, an exclude
// of every file, a git-hooks-wide exclude) shows up as a changed setting of
// that hook; settings of other hooks are out of scope.
func TestDevenvSecurity_SecuritySettings(t *testing.T) {
	t.Parallel()

	security := []string{"check-added-large-files", "lock-file-audit", "ripsecrets"}
	generated, err := DeclaredSecurity(generatedSecurityNix)
	if err != nil {
		t.Fatalf("DeclaredSecurity: %v", err)
	}
	want := generated.SecuritySettings(security)
	if want["git-hooks.hooks.lock-file-audit.entry"] == "" {
		t.Fatalf("generated settings %v lack lock-file-audit's entry", want)
	}
	if _, ok := want["git-hooks.hooks.gofmt.excludes"]; ok {
		t.Errorf("settings %v include the non-security gofmt hook", want)
	}

	tests := []struct {
		name, local, changed string
	}{
		{"no-op entry", `git-hooks.hooks.ripsecrets.entry = "true";`, "git-hooks.hooks.ripsecrets.entry"},
		{"exclude every file", `git-hooks.hooks.ripsecrets.excludes = [ ".*" ];`, "git-hooks.hooks.ripsecrets.excludes"},
		{"replace a custom entry", `git-hooks.hooks.lock-file-audit.entry = lib.mkForce "true";`, "git-hooks.hooks.lock-file-audit.entry"},
		{"exclude every file for every hook", `git-hooks.excludes = [ ".*" ];`, "git-hooks.excludes"},
		{"replace a whole hook", `git-hooks.hooks.ripsecrets = lib.mkForce { enable = true; entry = "true"; };`, "git-hooks.hooks.ripsecrets"},
		{"non-security hook", `git-hooks.hooks.gofmt.excludes = [ ".*" ];`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeNixFile(t, filepath.Join(dir, "devenv.nix"), generatedSecurityNix)
			writeNixFile(t, filepath.Join(dir, DevenvLocalNixFile), "{ lib, ... }:\n{\n  "+tt.local+"\n}\n")
			declared, err := ProjectDeclaredSecurity(dir)
			if err != nil {
				t.Fatalf("ProjectDeclaredSecurity: %v", err)
			}
			got := declared.SecuritySettings(security)
			if tt.changed == "" {
				if !maps.Equal(got, want) {
					t.Errorf("settings = %v, want the generated %v", got, want)
				}
				return
			}
			if v, ok := got[tt.changed]; ok && v == want[tt.changed] {
				t.Errorf("settings[%s] = %q, want it changed from the generated value", tt.changed, v)
			} else if !ok {
				t.Errorf("settings = %v, missing %s", got, tt.changed)
			}
		})
	}
}

func writeNixFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
