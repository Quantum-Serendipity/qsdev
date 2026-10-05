// Package powershell implements the PowerShell ecosystem module for
// qsdev. It detects PowerShell projects by scanning for
// requirements.psd1 and PowerShell script files, generates devenv.nix fragments
// with the PowerShell package, and provides CI commands, deny rules, and package
// manager metadata for the PowerShell toolchain.
//
// Security limitations: PSGallery (the primary PowerShell module repository) has
// no age-gating, no install-script blocking, and limited signing enforcement.
// While PowerShell supports Authenticode signatures, PSGallery does not require
// modules to be signed, and Install-Module does not verify signatures by default.
// The requirements.psd1 manifest provides version pinning but no integrity
// verification.
package powershell

import (
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the PowerShell scripting language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "powershell" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "PowerShell" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 4 }

// Detect scans projectRoot for PowerShell ecosystem indicators:
// requirements.psd1 (certain), *.ps1 (probable), and *.psm1 (probable).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
	)

	// Certain indicator.
	if fileutil.FileExists(projectRoot, "requirements.psd1") {
		evidence = append(evidence, "requirements.psd1 found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Probable indicators.
	if ps1Files, _ := filepath.Glob(filepath.Join(projectRoot, "*.ps1")); len(ps1Files) > 0 {
		evidence = append(evidence, "*.ps1 files found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if psmFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.psm1")); len(psmFiles) > 0 {
		evidence = append(evidence, "*.psm1 files found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}

	if !detected {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: make(map[string]string),
		},
	}
}

// DevenvPackages returns the Nix packages required for the PowerShell ecosystem.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"powershell"}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for PowerShell support. Packages are provided via DevenvPackages.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "", nil
}

// SecurityConfigs returns generated security configuration files.
// PSGallery has no age-gating and no install-script blocking, so no security
// configuration files are generated.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the PowerShell ecosystem.
// PSScriptAnalyzer is a PowerShell module, not a standalone CLI, so no
// pre-commit hooks are provided.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return nil
}

// DenyRules returns Claude Code deny-rule patterns for the PowerShell ecosystem.
// These prevent direct PSGallery module installation outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	// In Claude Code's PowerShell tool (the default shell on Windows, which
	// Bash(...) rules never match) each installer cmdlet is denied as a whole
	// word wherever it appears: at the start, after `;`, `|` (`Find-Module X
	// | Install-Module`), `&&` or a newline, inside a scriptblock or
	// Invoke-Expression string, or module-qualified
	// (`PowerShellGet\Install-Module`). Whole word means a longer local-only
	// cmdlet that starts with an installer's name (Update-ModuleManifest,
	// Update-ScriptFileInfo) stays allowed. The aliases are denied the same
	// way (see psWordRules). These rules are the only layer for the
	// PowerShell tool; the package-guard hook inspects only scripts passed to
	// pwsh/powershell. A command that merely names a cmdlet (`Get-Help
	// Install-Module`) is over-blocked; run it yourself in a terminal.
	//
	// In the Bash tool the cmdlet is denied bare or inside any
	// pwsh/powershell invocation, including behind an env prefix (-c,
	// -Command, -NoProfile -Command, pwsh.exe, `-c 'a; Install-Module'`,
	// `env X=1 pwsh -c ...`). Those rules match a substring, so they also
	// over-block any Bash command that mentions both words (`git commit -m
	// 'powershell: document Install-Module'`, `grep -rn pwsh docs | grep
	// Install-Module`) and a longer cmdlet under pwsh (`pwsh -c
	// Update-ModuleManifest`); run such a command yourself in a terminal.
	//
	// PowerShell(...) rules match case-insensitively, so one spelling covers
	// every casing. Bash(...) rules cannot be made case-insensitive: they
	// carry the documented and the all-lowercase spelling, and any other
	// casing under pwsh (INSTALL-MODULE) is left to the package-guard hook,
	// which matches case-insensitively.
	rules := psWordRules(psInstallCmdlets, []string{"*"})
	for _, cmdlet := range psInstallCmdlets {
		for _, spelling := range []string{cmdlet, strings.ToLower(cmdlet)} {
			rules = append(rules,
				"Bash("+spelling+"*)",
				"Bash(*pwsh*"+spelling+"*)",
				"Bash(*powershell*"+spelling+"*)",
			)
		}
	}
	return append(rules, psWordRules(psInstallAliases, psAliasBefore)...)
}

// psInstallAliases are PSResourceGet's aliases for Install-PSResource and
// Update-PSResource.
var psInstallAliases = []string{"isres", "udres"}

// psAliasBefore is what may precede an alias in a PowerShell tool call: the
// start of the command, or a space, `;`, `|`, newline, `{`, `(` or `&`
// (`&&` without spaces). Unlike the cmdlet names, the short aliases are
// not matched at the end of another word (`thisresult.txt`).
var psAliasBefore = []string{"", "* ", "*;", "*|", "*\n", "*{", "*(", "*&"}

// psWordAfter is what may follow a denied word: arguments, the end of the
// command, a statement or pipeline separator, the end of a scriptblock,
// subexpression or string, or a newline.
var psWordAfter = []string{" *", "", ";*", "|*", "}*", ")*", "'*", `"*`, "\n*"}

// psWordRules returns PowerShell tool deny rules that match each of words as
// a whole word: preceded by one of before and followed by one of
// psWordAfter, so a longer name that starts with the word is not matched.
func psWordRules(words, before []string) []string {
	var rules []string
	for _, word := range words {
		for _, b := range before {
			for _, a := range psWordAfter {
				if b == "" && a == "" {
					continue // "<word> *" already matches the bare word.
				}
				rules = append(rules, "PowerShell("+b+word+a+")")
			}
		}
	}
	return rules
}

// psInstallCmdlets install or update modules or scripts from PSGallery or
// NuGet: PowerShellGet's Install-/Save-/Update-Module and Install-/
// Update-Script, PSResourceGet's Install-/Save-/Update-PSResource (bundled
// since PowerShell 7.4) and PackageManagement's Install-Package.
var psInstallCmdlets = []string{
	"Install-Module", "Save-Module", "Update-Module",
	"Install-Script", "Update-Script",
	"Install-PSResource", "Save-PSResource", "Update-PSResource",
	"Install-Package",
}

// psScriptAnalyzerVersion is the PSScriptAnalyzer release the CI job installs
// from PSGallery. PSScriptAnalyzer is a PowerShell module, not a nixpkgs
// package, so devenv cannot provide it; the version is pinned so CI does not
// pick up whatever PSGallery serves on the day.
const psScriptAnalyzerVersion = "1.25.0"

// CICommands returns CI pipeline commands for the PowerShell ecosystem. The
// install step provisions the pinned PSScriptAnalyzer with PSResourceGet
// (bundled with PowerShell 7.4+; "[x.y.z]" is the NuGet exact-version
// range), and the scan fails when the analyzer
// reports any error-severity finding or a script it cannot parse:
// Invoke-ScriptAnalyzer only returns its findings, and pwsh exits 0
// regardless. Parse errors have their own ParseError severity, which
// `-Severity Error` alone filters out, so a script with a syntax error would
// otherwise pass. Both run with
// $ErrorActionPreference = 'Stop' so a failed install or import fails the
// step. The scripts are single-quoted so the shell leaves `$` alone.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name: "psscriptanalyzer-install",
			Command: `pwsh -NoProfile -NonInteractive -Command '$ErrorActionPreference = "Stop"; ` +
				`Install-PSResource -Name PSScriptAnalyzer -Version "[` + psScriptAnalyzerVersion + `]"` +
				` -Repository PSGallery -TrustRepository -Scope CurrentUser -Quiet'`,
			Description: "Install the pinned PSScriptAnalyzer module from PSGallery",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name: "psscriptanalyzer",
			Command: `pwsh -NoProfile -NonInteractive -Command '$ErrorActionPreference = "Stop"; ` +
				`Import-Module PSScriptAnalyzer -RequiredVersion ` + psScriptAnalyzerVersion + `; ` +
				`$findings = @(Invoke-ScriptAnalyzer -Path . -Recurse -Severity Error, ParseError); ` +
				`if ($findings.Count -gt 0) { $findings | Format-Table -AutoSize | Out-String -Width 4096 | Write-Host; exit 1 }'`,
			Description: "Scan PowerShell scripts with PSScriptAnalyzer, failing on error-severity findings and parse errors",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about PowerShell's PSGallery package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "psgallery",
			LockFile: "",
		},
	}
}

// VerificationCommands returns an empty set. PowerShell does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}
