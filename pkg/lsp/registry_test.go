package lsp

import "testing"

func TestRegistryValidation(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	const wantTotal = 26
	if got := len(r.All()); got != wantTotal {
		t.Fatalf("All() returned %d servers, want %d", got, wantTotal)
	}

	wantEcosystems := []string{
		"go", "javascript", "python", "rust", "cpp", "java", "dotnet", "scala",
		"elixir", "clojure", "container", "terraform", "php", "ruby", "shell",
		"helm", "nix", "perl", "lua", "zig", "dart", "r",
		"kotlin", "haskell", "swift", "ansible",
	}
	for _, name := range wantEcosystems {
		if _, ok := r.ByEcosystem(name); !ok {
			t.Errorf("ByEcosystem(%q) not found", name)
		}
	}
}

func TestNewRegistryChecked(t *testing.T) {
	t.Parallel()

	r, err := NewRegistryChecked()
	if err != nil {
		t.Fatalf("NewRegistryChecked() error = %v, want nil", err)
	}
	if r == nil {
		t.Fatal("NewRegistryChecked() returned nil registry")
	}
}

func TestDefaultOnCount(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	defaultOn := r.DefaultOn()
	const wantDefaultOn = 22
	if got := len(defaultOn); got != wantDefaultOn {
		t.Fatalf("DefaultOn() returned %d servers, want %d", got, wantDefaultOn)
	}

	for _, cfg := range defaultOn {
		if !cfg.DefaultOn {
			t.Errorf("DefaultOn() included %q with DefaultOn == false", cfg.EcosystemName)
		}
	}

	// And the complementary count of opt-in servers.
	var optIn int
	for _, cfg := range r.All() {
		if !cfg.DefaultOn {
			optIn++
		}
	}
	if optIn != 4 {
		t.Errorf("opt-in (default-off) count = %d, want 4", optIn)
	}
}

func TestDefaultOnSortedDeterministic(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	for _, getter := range []struct {
		name string
		fn   func() []*LSPServerConfig
	}{
		{"All", r.All},
		{"DefaultOn", r.DefaultOn},
	} {
		t.Run(getter.name, func(t *testing.T) {
			t.Parallel()
			got := getter.fn()
			for i := 1; i < len(got); i++ {
				if got[i-1].EcosystemName > got[i].EcosystemName {
					t.Fatalf("%s() not sorted: %q before %q", getter.name, got[i-1].EcosystemName, got[i].EcosystemName)
				}
			}
		})
	}
}

func TestByEcosystem(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	tests := []struct {
		name        string
		ecosystem   string
		wantOK      bool
		wantCommand string
	}{
		{name: "go", ecosystem: "go", wantOK: true, wantCommand: "gopls"},
		{name: "javascript", ecosystem: "javascript", wantOK: true, wantCommand: "typescript-language-server"},
		{name: "nix", ecosystem: "nix", wantOK: true, wantCommand: "nixd"},
		{name: "unknown", ecosystem: "cobol", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, ok := r.ByEcosystem(tt.ecosystem)
			if ok != tt.wantOK {
				t.Fatalf("ByEcosystem(%q) ok = %v, want %v", tt.ecosystem, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if cfg.Command != tt.wantCommand {
				t.Errorf("ByEcosystem(%q).Command = %q, want %q", tt.ecosystem, cfg.Command, tt.wantCommand)
			}
		})
	}
}

func TestNixdAlwaysDefaultOn(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("nix")
	if !ok {
		t.Fatal("ByEcosystem(\"nix\") not found")
	}
	if !cfg.DefaultOn {
		t.Errorf("nix server DefaultOn = false, want true")
	}
}

func TestSandboxCategorySet(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	for _, cfg := range r.All() {
		switch cfg.SandboxCategory {
		case "A", "B", "C":
		default:
			t.Errorf("server %q has SandboxCategory %q, want one of A/B/C", cfg.EcosystemName, cfg.SandboxCategory)
		}
	}
}

func TestValidateRejectsBadConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*LSPServerConfig)
	}{
		{name: "empty language id", mutate: func(c *LSPServerConfig) { c.LanguageID = "" }},
		{name: "no extensions", mutate: func(c *LSPServerConfig) { c.Extensions = nil }},
		{name: "bad sandbox category", mutate: func(c *LSPServerConfig) { c.SandboxCategory = "D" }},
		{name: "empty nix package not bundled", mutate: func(c *LSPServerConfig) {
			c.NixPackage = ""
			c.Devenv = DevenvEnable
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewRegistry()
			cfg, ok := r.ByEcosystem("go")
			if !ok {
				t.Fatal("ByEcosystem(\"go\") not found")
			}
			tt.mutate(cfg)
			if err := r.Validate(); err == nil {
				t.Fatalf("Validate() = nil, want error for %s", tt.name)
			}
		})
	}
}

func TestValidateAllowsSDKBundledEmptyPackage(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("dart")
	if !ok {
		t.Fatal("ByEcosystem(\"dart\") not found")
	}
	if cfg.NixPackage != "" {
		t.Errorf("dart NixPackage = %q, want empty (SDK-bundled)", cfg.NixPackage)
	}
	if cfg.Devenv != DevenvSDKBundled {
		t.Errorf("dart Devenv = %d, want DevenvSDKBundled", cfg.Devenv)
	}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil for SDK-bundled server", err)
	}
}
