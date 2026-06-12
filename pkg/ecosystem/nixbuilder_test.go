package ecosystem_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func TestBuildLanguageFragment_SimpleEnable(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.NixLangConfig{
		EnablePath: "languages.zig",
	}
	got := ecosystem.BuildLanguageFragment(cfg)
	want := "  languages.zig.enable = true;\n"
	if got != want {
		t.Errorf("BuildLanguageFragment() =\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildLanguageFragment_WithProperties(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.NixLangConfig{
		EnablePath: "languages.ruby",
		Properties: []ecosystem.NixProperty{
			{Key: "bundler.enable", Value: "true"},
		},
	}
	got := ecosystem.BuildLanguageFragment(cfg)
	want := `  languages.ruby = {
    enable = true;
    bundler.enable = true;
  };
`
	if got != want {
		t.Errorf("BuildLanguageFragment() =\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildLanguageFragment_WithEnvVars(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.NixLangConfig{
		EnablePath: "languages.go",
		Properties: []ecosystem.NixProperty{
			{Key: "package", Value: "pkgs.go_1_24"},
		},
		EnvVars: []ecosystem.NixEnvVar{
			{
				Key:     "GOFLAGS",
				Value:   `"-mod=readonly"`,
				Comment: "Enforce module-aware mode — prevents unvetted dependency additions",
			},
			{
				Key:   "GONOSUMCHECK",
				Value: `""`,
			},
		},
	}
	got := ecosystem.BuildLanguageFragment(cfg)
	want := `  languages.go = {
    enable = true;
    package = pkgs.go_1_24;
  };

  # Enforce module-aware mode — prevents unvetted dependency additions
  env.GOFLAGS = "-mod=readonly";
  env.GONOSUMCHECK = "";
`
	if got != want {
		t.Errorf("BuildLanguageFragment() =\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildLanguageFragment_WithExtraBlocks(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.NixLangConfig{
		EnablePath: "languages.haskell",
		ExtraBlocks: []string{
			"  languages.haskell.stack.enable = true;\n",
			"  # NOTE: cabal.project.freeze is NOT a true lockfile — it pins\n  # versions but does not record content hashes.\n",
		},
	}
	got := ecosystem.BuildLanguageFragment(cfg)
	want := `  languages.haskell.enable = true;

  languages.haskell.stack.enable = true;

  # NOTE: cabal.project.freeze is NOT a true lockfile — it pins
  # versions but does not record content hashes.
`
	if got != want {
		t.Errorf("BuildLanguageFragment() =\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildLanguageFragment_FullConfig(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.NixLangConfig{
		EnablePath: "languages.python",
		Properties: []ecosystem.NixProperty{
			{Key: "version", Value: `"3.12"`},
			{Key: "uv.enable", Value: "true"},
			{Key: "venv.enable", Value: "true"},
		},
	}
	got := ecosystem.BuildLanguageFragment(cfg)
	want := `  languages.python = {
    enable = true;
    version = "3.12";
    uv.enable = true;
    venv.enable = true;
  };
`
	if got != want {
		t.Errorf("BuildLanguageFragment() =\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildLanguageFragment_ExtraBlockNoTrailingNewline(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.NixLangConfig{
		EnablePath:  "languages.scala",
		ExtraBlocks: []string{"  languages.kotlin.enable = true;"},
	}
	got := ecosystem.BuildLanguageFragment(cfg)
	want := `  languages.scala.enable = true;

  languages.kotlin.enable = true;
`
	if got != want {
		t.Errorf("BuildLanguageFragment() =\n%q\nwant:\n%q", got, want)
	}
}

func TestNixString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello", `"hello"`},
		{"with dollar-brace", "foo${bar}", `"foo\${bar}"`},
		{"empty", "", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ecosystem.NixString(tt.input)
			if got != tt.want {
				t.Errorf("NixString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
