package devenv

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// normalizeCases are devenv.nix-shaped modules and their normalized form.
// Every input is valid Nix whose value must not change, except those marked
// duplicate: Nix rejects them, and the normalized form must evaluate.
var normalizeCases = []struct {
	name      string
	in        string
	want      string
	duplicate bool
}{
	{
		name: "unique keys are kept verbatim",
		in:   "{ pkgs, ... }:\n{\n  # Base\n  packages = [ pkgs.git ];\n  dotenv.enable = false;\n}\n",
		want: "{ pkgs, ... }:\n{\n  # Base\n  packages = [ pkgs.git ];\n  dotenv.enable = false;\n}\n",
	},
	{
		name: "repeated path keys become one attribute set",
		in: "{ pkgs, ... }:\n{\n  packages = [ pkgs.go ];\n\n  # Go\n  languages.go = {\n    enable = true;\n  };\n" +
			"  languages.go.lsp.enable = true;\n  languages.nix.enable = true;\n  languages.nix.lsp.enable = true;\n}\n",
		want: "{ pkgs, ... }:\n{\n  packages = [ pkgs.go ];\n\n  languages = {\n    # Go\n    go = {\n      enable = true;\n      lsp.enable = true;\n    };\n" +
			"    nix = {\n      enable = true;\n      lsp.enable = true;\n    };\n  };\n}\n",
	},
	{
		name: "path bindings merge into the plain attribute set",
		in:   "{ ... }:\n{\n  env = {\n    A = \"1\";\n  };\n  dotenv.enable = false;\n  # why B\n  env.B = \"2\";\n}\n",
		want: "{ ... }:\n{\n  env = {\n    A = \"1\";\n    # why B\n    B = \"2\";\n  };\n  dotenv.enable = false;\n}\n",
	},
	{
		name: "single child chain collapses and keeps the block position",
		in:   "{ pkgs, ... }:\n{\n  git-hooks.hooks.extra = {\n    enable = true;\n  };\n  x = 1;\n\n  # Git hooks\n  git-hooks.hooks = {\n    ripsecrets.enable = true;\n  };\n}\n",
		want: "{ ... }:\n{\n  x = 1;\n\n  # Git hooks\n  git-hooks.hooks = {\n    extra = {\n      enable = true;\n    };\n    ripsecrets.enable = true;\n  };\n}\n",
	},
	{
		name: "moved indented string keeps its value, double-quoted lines stay put",
		in: "{ ... }:\n{\n  scripts.a.exec = ''\n    echo a\n      \n    echo b\n  '';\n" +
			"  scripts.b.exec = \"line1\n  line2\";\n}\n",
		want: "{ ... }:\n{\n  scripts = {\n    a.exec = ''\n      echo a\n        \n      echo b\n    '';\n" +
			"    b.exec = \"line1\n  line2\";\n  };\n}\n",
	},
	{
		name: "let, with and interpolation inside values",
		in: "{ pkgs, lib, ... }:\n{\n  hooks.a = let x = \"}\"; in \"${x};\";\n  hooks.b = with lib; [ \"${toString 1}\" ];\n" +
			"  hooks.c = \"$${notInterpolated} \\${also}\";\n}\n",
		want: "{ lib, ... }:\n{\n  hooks = {\n    a = let x = \"}\"; in \"${x};\";\n    b = with lib; [ \"${toString 1}\" ];\n" +
			"    c = \"$${notInterpolated} \\${also}\";\n  };\n}\n",
	},
	{
		name: "repeated keys inside a nested attribute set",
		in:   "{ ... }:\n{\n  services.kafka = {\n    enable = true;\n    settings.a = 1;\n    settings.b = 2;\n  };\n}\n",
		want: "{ ... }:\n{\n  services.kafka = {\n    enable = true;\n    settings = {\n      a = 1;\n      b = 2;\n    };\n  };\n}\n",
	},
	{
		name:      "the same leaf set to the same value twice is defined once",
		in:        "{ pkgs, ... }:\n{\n  languages.java = {\n    enable = true;\n  };\n  languages.java.enable = true;\n  languages.scala.enable = true;\n}\n",
		want:      "{ ... }:\n{\n  languages = {\n    java.enable = true;\n    scala.enable = true;\n  };\n}\n",
		duplicate: true,
	},
	{
		name: "quoted and bare names group together",
		in:   "{ ... }:\n{\n  scripts.\"qsdev-build\".exec = \"b\";\n  scripts.qsdev-build.description = \"d\";\n}\n",
		want: "{ ... }:\n{\n  scripts.\"qsdev-build\" = {\n    exec = \"b\";\n    description = \"d\";\n  };\n}\n",
	},
}

func TestNormalizeNixModule(t *testing.T) {
	t.Parallel()
	for _, tt := range normalizeCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeNixModule(tt.in)
			if err != nil {
				t.Fatalf("normalizeNixModule: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeNixModule =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

// TestNormalizeNixModule_PreservesValue evaluates each case before and after
// normalization and requires the same result.
func TestNormalizeNixModule_PreservesValue(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	eval := func(t *testing.T, src string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "m.nix")
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		expr := `{ f }: builtins.toJSON (import (/. + f) { pkgs = { git = "git"; go = "go"; }; lib = { }; })`
		out, err := exec.CommandContext(ctx, nixInstantiate, "--eval", "--strict", "--argstr", "f", path, "--expr", expr).CombinedOutput()
		if err != nil {
			t.Fatalf("evaluating: %v\n%s\n%s", err, out, src)
		}
		return string(out)
	}
	for _, tt := range normalizeCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeNixModule(tt.in)
			if err != nil {
				t.Fatalf("normalizeNixModule: %v", err)
			}
			after := eval(t, got)
			if tt.duplicate {
				return
			}
			if before := eval(t, tt.in); before != after {
				t.Errorf("value changed:\nbefore %s\nafter  %s", before, after)
			}
		})
	}
}

func TestNormalizeNixModule_Errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		wantSub string
		syntax  bool
	}{
		{"leaf defined twice", "{ ... }:\n{\n  env = { A = \"1\"; };\n  env.A = \"2\";\n}\n", "defines env.A more than once", false},
		{"leaf and attribute set", "{ ... }:\n{\n  a.b = 1;\n  a.b.c = 2;\n}\n", "defines a.b more than once", false},
		{"unterminated string", "{ ... }:\n{\n  a = \"x;\n}\n", "unterminated string", true},
		{"inherit is not supported", "{ pkgs, ... }:\n{\n  inherit pkgs;\n}\n", "expected '='", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeNixModule(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("normalizeNixModule error = %v, want one containing %q", err, tt.wantSub)
			}
			if got := errors.Is(err, errNixSyntax); got != tt.syntax {
				t.Errorf("errors.Is(err, errNixSyntax) = %v, want %v", got, tt.syntax)
			}
		})
	}
}

// Error messages name attribute paths, but a quoted segment can hold any text
// (the module embeds service credentials in string literals), so only plain
// identifiers may be echoed.
func TestNormalizeNixModule_ErrorsOmitQuotedSegments(t *testing.T) {
	t.Parallel()
	const secret = "s3cr3t-Passw0rd"
	tests := []struct {
		name    string
		in      string
		wantSub string
	}{
		{"duplicate quoted key", "{ ... }:\n{\n  env.\"" + secret + "\" = \"1\";\n  env.\"" + secret + "\" = \"2\";\n}\n", `defines env."..." more than once`},
		{"missing semicolon", "{ ... }:\n{\n  env.\"" + secret + "\" = \"1\"\n}\n", `after the value of env."..."`},
		{"missing equals", "{ ... }:\n{\n  env.\"" + secret + "\" ;\n}\n", `expected '=' after env."..."`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeNixModule(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("normalizeNixModule error = %v, want one containing %q", err, tt.wantSub)
			}
			if strings.Contains(err.Error(), secret) {
				t.Errorf("error %q quotes the string segment %q", err, secret)
			}
		})
	}
}

func TestDescribeNixPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path []string
		want string
	}{
		{"identifiers", []string{"services", "minio", "enable"}, "services.minio.enable"},
		{"primes and dashes", []string{"a'", "b-c", "_d1"}, "a'.b-c._d1"},
		{"quoted", []string{"env", `"KEY"`}, `env."..."`},
		{"interpolated", []string{"env", "${name}"}, `env."..."`},
		{"empty", []string{""}, `"..."`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := describeNixPath(tt.path); got != tt.want {
				t.Errorf("describeNixPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDropNixBindings(t *testing.T) {
	t.Parallel()
	fragment := "  languages.go.enable = true;\n\n  # why\n  env.GOFLAGS = \"-mod=readonly\";\n  env.\"GOSUMDB\" = \"sum.golang.org\";\n"
	tests := []struct {
		name string
		drop []string
		want string
	}{
		{"nothing overridden", nil, fragment},
		{"bare name with its comment", []string{"GOFLAGS"}, "  languages.go.enable = true;\n  env.\"GOSUMDB\" = \"sum.golang.org\";\n"},
		{"quoted name", []string{"GOSUMDB"}, "  languages.go.enable = true;\n\n  # why\n  env.GOFLAGS = \"-mod=readonly\";\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := dropNixBindings(fragment, func(path []string) bool {
				return len(path) == 2 && path[0] == "env" && slices.Contains(tt.drop, path[1])
			})
			if err != nil {
				t.Fatalf("dropNixBindings: %v", err)
			}
			if got != tt.want {
				t.Errorf("dropNixBindings =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
