package lsp

import (
	"fmt"
	"strings"
)

// nixIndent is the leading indentation for emitted devenv.nix lines, matching
// the 2-space style of existing language fragments.
const nixIndent = "  "

// analyzerConfigComment notes where analyzer settings actually live, since
// devenv exposes no analyzer attributes — they go in the generated plugin.
const analyzerConfigComment = nixIndent + "# Analyzer config lives in .claude/skills/qsdev-lsp/.lsp.json.\n"

// NixLSPFragment returns the devenv.nix LSP lines for cfg, selected by
// cfg.Devenv. It returns "" for servers whose binary is added to the packages
// list (DevenvPackageList) or bundled in a base SDK (DevenvSDKBundled); callers
// detect those via PackageListPackage and the registry, respectively.
//
// Emitted lines use 2-space indentation. Enable cases append a trailing comment
// noting that analyzer configuration lives in the generated .lsp.json.
func NixLSPFragment(cfg *LSPServerConfig) string {
	switch cfg.Devenv {
	case DevenvEnable:
		var b strings.Builder
		fmt.Fprintf(&b, "%s%s.lsp.enable = true;\n", nixIndent, cfg.DevenvLSPAttr)
		b.WriteString(analyzerConfigComment)
		return b.String()

	case DevenvEnableOverridePackage:
		var b strings.Builder
		fmt.Fprintf(&b, "%s%s.lsp.enable = true;\n", nixIndent, cfg.DevenvLSPAttr)
		fmt.Fprintf(&b, "%s%s.lsp.package = pkgs.%s;\n", nixIndent, cfg.DevenvLSPAttr, cfg.NixPackage)
		b.WriteString(analyzerConfigComment)
		return b.String()

	case DevenvDisable:
		// Force-disable first (devenv would otherwise default this server on),
		// then a commented opt-in hint carrying the caveat.
		var b strings.Builder
		fmt.Fprintf(&b, "%s%s.lsp.enable = false;\n", nixIndent, cfg.DevenvLSPAttr)
		fmt.Fprintf(&b, "%s# %s: default-off (opt-in) — %s\n", nixIndent, cfg.DisplayName, cfg.Caveats)
		fmt.Fprintf(&b, "%s# %s.lsp.enable = true;\n", nixIndent, cfg.DevenvLSPAttr)
		return b.String()

	case DevenvPackageList, DevenvSDKBundled:
		return ""

	default:
		return ""
	}
}

// PackageListPackage reports the nixpkgs package that must be added to the
// devenv packages list for cfg, returning (NixPackage, true) only when
// cfg.Devenv == DevenvPackageList (no devenv lsp option exists). For all other
// actions it returns ("", false): DevenvEnable / DevenvEnableOverridePackage
// provision the package via the lsp option, and DevenvSDKBundled ships in a base
// SDK package.
func PackageListPackage(cfg *LSPServerConfig) (string, bool) {
	if cfg.Devenv == DevenvPackageList {
		return cfg.NixPackage, true
	}
	return "", false
}
