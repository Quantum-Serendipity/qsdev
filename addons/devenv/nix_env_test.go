package devenv

import (
	"maps"
	"os"
	"path/filepath"
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
