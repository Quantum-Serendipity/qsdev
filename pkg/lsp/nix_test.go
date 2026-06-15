package lsp

import (
	"strings"
	"testing"
)

func TestNixLSPFragment(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	tests := []struct {
		name        string
		ecosystem   string
		wantEmpty   bool
		wantSubstrs []string
		notSubstrs  []string
	}{
		{
			name:        "enable go",
			ecosystem:   "go",
			wantSubstrs: []string{"languages.go.lsp.enable = true;", ".lsp.json"},
		},
		{
			name:      "enable+override cpp",
			ecosystem: "cpp",
			wantSubstrs: []string{
				"languages.cplusplus.lsp.enable = true;",
				"languages.cplusplus.lsp.package = pkgs.clang-tools;",
			},
		},
		{
			name:      "enable+override ruby",
			ecosystem: "ruby",
			wantSubstrs: []string{
				"languages.ruby.lsp.enable = true;",
				"languages.ruby.lsp.package = pkgs.ruby-lsp;",
			},
		},
		{
			name:      "disable kotlin",
			ecosystem: "kotlin",
			wantSubstrs: []string{
				"languages.kotlin.lsp.enable = false;",
				"# languages.kotlin.lsp.enable = true;",
				"maintenance limbo",
			},
		},
		{
			name:      "package-list container",
			ecosystem: "container",
			wantEmpty: true,
		},
		{
			name:      "sdk-bundled dart",
			ecosystem: "dart",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, ok := r.ByEcosystem(tt.ecosystem)
			if !ok {
				t.Fatalf("ByEcosystem(%q) not found", tt.ecosystem)
			}
			got := NixLSPFragment(cfg)
			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("NixLSPFragment(%q) = %q, want empty", tt.ecosystem, got)
				}
				return
			}
			for _, sub := range tt.wantSubstrs {
				if !strings.Contains(got, sub) {
					t.Errorf("NixLSPFragment(%q) = %q, missing %q", tt.ecosystem, got, sub)
				}
			}
			for _, sub := range tt.notSubstrs {
				if strings.Contains(got, sub) {
					t.Errorf("NixLSPFragment(%q) = %q, should not contain %q", tt.ecosystem, got, sub)
				}
			}
		})
	}
}

func TestNixLSPFragmentDisableOrder(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("kotlin")
	if !ok {
		t.Fatal("ByEcosystem(\"kotlin\") not found")
	}
	got := NixLSPFragment(cfg)

	disableIdx := strings.Index(got, "languages.kotlin.lsp.enable = false;")
	commentIdx := strings.Index(got, "# Kotlin")
	if disableIdx < 0 || commentIdx < 0 {
		t.Fatalf("NixLSPFragment(kotlin) = %q, missing expected lines", got)
	}
	if disableIdx > commentIdx {
		t.Errorf("force-disable line should come before opt-in comment; got %q", got)
	}
}

func TestNixLSPFragmentIndentation(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("go")
	if !ok {
		t.Fatal("ByEcosystem(\"go\") not found")
	}
	got := NixLSPFragment(cfg)
	for _, line := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %q does not use 2-space indentation", line)
		}
	}
}

func TestPackageListPackage(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	tests := []struct {
		name     string
		eco      string
		wantPkg  string
		wantList bool
	}{
		{name: "container is package-list", eco: "container", wantPkg: "docker-language-server", wantList: true},
		{name: "helm is package-list", eco: "helm", wantPkg: "helm-ls", wantList: true},
		{name: "go is not package-list", eco: "go", wantPkg: "", wantList: false},
		{name: "dart sdk-bundled is not package-list", eco: "dart", wantPkg: "", wantList: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, ok := r.ByEcosystem(tt.eco)
			if !ok {
				t.Fatalf("ByEcosystem(%q) not found", tt.eco)
			}
			pkg, isList := PackageListPackage(cfg)
			if isList != tt.wantList {
				t.Fatalf("PackageListPackage(%q) ok = %v, want %v", tt.eco, isList, tt.wantList)
			}
			if pkg != tt.wantPkg {
				t.Errorf("PackageListPackage(%q) pkg = %q, want %q", tt.eco, pkg, tt.wantPkg)
			}
		})
	}
}
