// Package lsp is the single source of truth for the Language Server Protocol
// (LSP) server configurations qsdev provisions for detected ecosystems.
//
// The registry (registry.go) holds one LSPServerConfig per ecosystem, keyed by
// the same name an ecosystem module reports from EcosystemModule.Name(). Other
// units consume this package to:
//
//   - generate the consolidated .lsp.json Claude Code plugin (lspjson.go helpers),
//   - emit explicit devenv.nix lsp enable/disable fragments (nix.go), and
//   - drive path-scoped rule frontmatter (RuleGlobs).
//
// Each config also carries a SandboxCategory (A/B/C) so the downstream LSP
// sandboxing work (P43) has a complete picture for all 26 servers, including
// the four opt-in (default-off) servers that never trigger generation on their
// own.
package lsp

// DevenvAction selects how the devenv.nix fragment for a server is emitted.
// devenv already defaults languages.<lang>.lsp.enable = true and exposes only
// enable + package (no analyzer attributes), so analyzer settings always live
// in the generated .lsp.json rather than in devenv. These actions only control
// the enable/disable/package lines.
type DevenvAction int

const (
	// DevenvEnable emits "<attr>.lsp.enable = true;". Used when devenv's default
	// LSP package already matches qsdev's pick.
	DevenvEnable DevenvAction = iota

	// DevenvEnableOverridePackage emits the enable line plus
	// "<attr>.lsp.package = pkgs.<NixPackage>;" — used when devenv's default LSP
	// package differs from qsdev's pick (e.g. cpp defaults to ccls, ruby to
	// solargraph).
	DevenvEnableOverridePackage

	// DevenvDisable force-disables a server devenv would otherwise default on. It
	// emits an explicit "<attr>.lsp.enable = false;" followed by a commented
	// opt-in hint carrying the caveat. Used for the default-off servers that have
	// a devenv lsp option.
	DevenvDisable

	// DevenvPackageList signals that no devenv lsp option exists for this server;
	// the binary must be added to the packages list instead. NixLSPFragment
	// returns "" for this action.
	DevenvPackageList

	// DevenvSDKBundled signals the server binary ships inside a base SDK package
	// (e.g. dart's language server ships in pkgs.dart), so neither an lsp option
	// nor a separate packages entry is needed. NixLSPFragment returns "".
	DevenvSDKBundled
)

// LSPServerConfig describes a single LSP server qsdev knows how to provision for
// an ecosystem. It is the registry value type and the single source of truth
// consumed by .lsp.json generation, devenv.nix fragments, and rule frontmatter.
type LSPServerConfig struct {
	// EcosystemName is the registry key and matches EcosystemModule.Name()
	// (e.g. "go", "javascript", "nix"). It is also the answers.Languages[].Name
	// lookup key used during generation.
	EcosystemName string

	// DisplayName is the human-readable label (e.g. "Go (gopls)").
	DisplayName string

	// Command is the LSP server binary to launch (e.g. "gopls").
	Command string

	// Args are the arguments passed to Command (e.g. ["serve"]). It is nil when
	// the server needs no arguments.
	Args []string

	// LanguageID is the LSP language identifier. It is the top-level key for this
	// server's entry in the generated .lsp.json (e.g. "go", "typescript").
	LanguageID string

	// Extensions are the file extensions this server handles, each with a leading
	// dot (e.g. [".go"]). They derive both the .lsp.json extensionToLanguage map
	// and the rule-file globs.
	Extensions []string

	// RulePatterns are extra non-extension globs for rule frontmatter only
	// (e.g. ["**/go.mod", "**/go.sum"]). It is nil when there are none.
	RulePatterns []string

	// InitOptions populates the .lsp.json initializationOptions object. It is nil
	// when the server needs none.
	InitOptions map[string]any

	// Settings populates the .lsp.json settings object (analyzer configuration).
	// It is nil when the server needs none.
	Settings map[string]any

	// NixPackage is the nixpkgs attribute path that provides the server binary
	// (e.g. "gopls", "clang-tools", "rPackages.languageserver"). It is empty only
	// when Devenv == DevenvSDKBundled.
	NixPackage string

	// DevenvLSPAttr is the devenv option path WITHOUT the trailing ".lsp"
	// (e.g. "languages.go"). It is empty when no devenv lsp option exists for the
	// server (Devenv == DevenvPackageList or DevenvSDKBundled).
	DevenvLSPAttr string

	// Devenv selects how NixLSPFragment treats this server.
	Devenv DevenvAction

	// DefaultOn reports whether the server is provisioned automatically when its
	// ecosystem is detected (true for the 22 default-on servers, false for the 4
	// opt-in servers).
	DefaultOn bool

	// Caveats is a human-readable note explaining why an opt-in server is
	// default-off. It is empty for default-on servers.
	Caveats string

	// MemoryLimitMB is an advisory memory cap for the server in megabytes. Zero
	// means no limit.
	MemoryLimitMB int

	// SandboxCategory is the sandboxing tier consumed by downstream LSP
	// sandboxing (P43). It must be one of "A", "B", or "C".
	SandboxCategory string
}
