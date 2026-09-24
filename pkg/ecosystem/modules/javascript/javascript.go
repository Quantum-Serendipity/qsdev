package javascript

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.ToolchainRequirementProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the JavaScript/TypeScript ecosystem.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return ecosystem.NameJavaScript }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "JavaScript/TypeScript" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for JavaScript/TypeScript indicators.
// It checks for package.json (Certain confidence) in the project root or,
// failing that, in a subproject directory such as frontend/ (recorded as the
// ExtraDirectory extra), determines the package manager from the package.json
// pin or lockfiles, reads Node.js version from .nvmrc or package.json engines,
// and checks for TypeScript via tsconfig.json.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	dir, others := findProjectDir(projectRoot)
	if dir == "" {
		return ecosystem.DetectionAbsent()
	}
	extras := make(map[string]string)
	evidence := []string{"package.json found"}
	if dir != "." {
		extras[ecosystem.ExtraDirectory] = dir
		evidence = []string{fmt.Sprintf("package.json found in %s/ (no package.json in the project root)", dir)}
	}
	if len(others) > 0 {
		evidence = append(evidence, fmt.Sprintf("WARNING: other JavaScript projects are not configured (devenv supports one JavaScript project directory): %s/", strings.Join(others, "/, ")))
	}
	// Every other file is read from the JavaScript project directory.
	jsRoot := filepath.Join(projectRoot, filepath.FromSlash(dir))
	pkg, _ := readPackageJSON(filepath.Join(jsRoot, "package.json"))

	pm := detectPackageManager(jsRoot, pkg)
	evidence = append(evidence, fmt.Sprintf("package manager: %s", pm))

	// Determine Node.js version: .nvmrc takes priority over engines.node.
	version := ""
	nvmrcPath := filepath.Join(jsRoot, ".nvmrc")
	if fileutil.FileExists(nvmrcPath) {
		version = strings.TrimPrefix(fileutil.ReadFirstLine(nvmrcPath), "v")
		if version != "" {
			evidence = append(evidence, fmt.Sprintf("node version %s (from .nvmrc)", version))
		}
	}
	if version == "" {
		version = pkg.Engines.Node
		if version != "" {
			evidence = append(evidence, fmt.Sprintf("node version %s (from engines.node)", version))
		}
	}
	if _, note := resolveNodeMajor(version); note != "" {
		evidence = append(evidence, "WARNING: "+note)
	}

	tsconfigPath := filepath.Join(jsRoot, "tsconfig.json")
	if fileutil.FileExists(tsconfigPath) {
		extras["typescript"] = "true"
		evidence = append(evidence, "tsconfig.json found")
	}
	if pm == "yarn" && isYarnClassic(jsRoot, pkg) {
		extras[ExtraYarnClassic] = "true"
		evidence = append(evidence, "Yarn Classic (v1) project")
	}
	if pm == "pnpm" {
		if name, pin := pkg.packageManagerPin(); name == "pnpm" && pin != "" {
			extras[ExtraPnpmVersion] = pin
			if !pnpmSupportsHardening(pin) {
				evidence = append(evidence, fmt.Sprintf("WARNING: package.json pins pnpm %s, which ignores the pnpm-workspace.yaml hardening (needs pnpm >= %s)", pin, pnpmHardeningMinVersion))
			}
		}
	}
	if src := detectJSTool(jsRoot, pkg, "eslint", eslintConfigFiles, pkg.ESLintConfig); src != "" {
		extras[ExtraESLint] = src
		evidence = append(evidence, "eslint configured")
	}
	if src := detectJSTool(jsRoot, pkg, "prettier", prettierConfigFiles, pkg.Prettier); src != "" {
		extras[ExtraPrettier] = src
		evidence = append(evidence, "prettier configured")
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: ecosystem.ConfidenceCertain,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version:        version,
			PackageManager: pm,
			Extras:         extras,
		},
	}
}

// packageManagers lists the package managers the module supports.
var packageManagers = []string{"npm", "pnpm", "yarn", "bun"}

// lockfilePackageManagers maps each package manager lockfile to its package
// manager, in the priority order detectPackageManager applies.
var lockfilePackageManagers = []struct{ lockfile, pm string }{
	{"pnpm-lock.yaml", "pnpm"},
	{"yarn.lock", "yarn"},
	{"bun.lock", "bun"},
	{"bun.lockb", "bun"},
	{"package-lock.json", "npm"},
	{"npm-shrinkwrap.json", "npm"},
}

// detectPackageManager determines the package manager. A package.json pin
// ("packageManager", then devEngines.packageManager) is authoritative, since
// Corepack and the package managers themselves enforce it; otherwise the
// lockfile decides, in priority order pnpm-lock.yaml > yarn.lock >
// bun.lock/bun.lockb > package-lock.json > npm (default).
func detectPackageManager(jsRoot string, pkg packageJSON) string {
	if name, _ := pkg.packageManagerPin(); slices.Contains(packageManagers, name) {
		return name
	}
	for _, l := range lockfilePackageManagers {
		if fileutil.FileExists(filepath.Join(jsRoot, l.lockfile)) {
			return l.pm
		}
	}
	return "npm"
}

// hasLockfile reports whether dir holds any package manager lockfile.
func hasLockfile(dir string) bool {
	return slices.ContainsFunc(lockfilePackageManagers, func(l struct{ lockfile, pm string }) bool {
		return fileutil.FileExists(filepath.Join(dir, l.lockfile))
	})
}

// findProjectDir returns the JavaScript project directory, relative to
// projectRoot in slash form: "." when package.json is in the project root, or
// else the subproject directory (frontend/, web/, ...) holding one within
// ecosystem.ProjectScanDepth levels, or "" when there is none. devenv
// configures a single JavaScript project directory, so when several
// subprojects exist the one with a lockfile wins, then the shallowest, then
// the first by name; the other subprojects are returned as others.
// Directories nested inside another candidate are its workspace members, not
// separate projects, and are left out of others.
func findProjectDir(projectRoot string) (dir string, others []string) {
	if fileutil.FileExists(filepath.Join(projectRoot, "package.json")) {
		return ".", nil
	}
	candidates := ecosystem.ShellSafeDirs(ecosystem.ProjectDirsWith(projectRoot, func(name string) bool {
		return name == "package.json"
	}))
	if len(candidates) == 0 {
		return "", nil
	}
	locked := func(d string) bool { return hasLockfile(filepath.Join(projectRoot, filepath.FromSlash(d))) }
	slices.SortStableFunc(candidates, func(a, b string) int {
		if la, lb := locked(a), locked(b); la != lb {
			if la {
				return -1
			}
			return 1
		}
		if da, db := strings.Count(a, "/"), strings.Count(b, "/"); da != db {
			return da - db
		}
		return strings.Compare(a, b)
	})
	dir = candidates[0]
	for _, c := range candidates[1:] {
		nested := slices.ContainsFunc(candidates, func(parent string) bool { return strings.HasPrefix(c, parent+"/") })
		if !nested {
			others = append(others, c)
		}
	}
	slices.Sort(others)
	return dir, others
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for JavaScript/TypeScript support.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	major, note := resolveNodeMajor(config.Version)
	pm := config.PM("npm")

	var b strings.Builder
	if note != "" {
		b.WriteString("  # " + note + "\n")
	}
	b.WriteString("  languages.javascript = {\n")
	b.WriteString("    enable = true;\n")
	if pm == "npm" {
		// npm comes from npm.package, so Node.js is the slim build: the full
		// one would also put its bundled (possibly too old) npm on PATH.
		fmt.Fprintf(&b, "    package = %s;\n", nodeSlimNixPackage(major))
	} else {
		fmt.Fprintf(&b, "    package = %s;\n", nodeNixPackage(major))
	}
	if dir := config.Directory(); dir != "" {
		// An absolute path: devenv also puts <directory>/node_modules/.bin on
		// PATH, which must not depend on the shell's working directory.
		fmt.Fprintf(&b, "    directory = \"${config.devenv.root}/%s\";\n", dir)
	}

	// Package manager specific configuration.
	switch pm {
	case "npm":
		b.WriteString("    npm.enable = true;\n")
		// The .npmrc age gate (min-release-age) needs npm >= 11.10, which
		// Node.js 22's bundled npm 10 is not, so npm is chosen independently
		// of the Node.js major.
		fmt.Fprintf(&b, "    # npm >= %s: enforces the .npmrc min-release-age gate\n", npmMinReleaseAgeVersion)
		fmt.Fprintf(&b, "    npm.package = %s;\n", npmNixPackage(major))
	case "pnpm":
		b.WriteString("    pnpm.enable = true;\n")
	case "yarn":
		b.WriteString("    yarn.enable = true;\n")
		if config.Extra(ExtraYarnClassic, "") != "true" {
			// devenv defaults to pkgs.yarn, which is Yarn Classic: it refuses
			// to run in a project pinning Yarn >= 2 and never reads the
			// generated .yarnrc.yml.
			b.WriteString("    yarn.package = pkgs.yarn-berry;\n")
		}
	case "bun":
		b.WriteString("    bun.enable = true;\n")
	}

	b.WriteString("  };\n")

	// TypeScript support.
	if config.Extra("typescript", "") == "true" {
		b.WriteString("\n  languages.typescript.enable = true;\n")
	}

	return b.String(), nil
}

// ToolchainRequirements returns the tool versions the generated hardening
// needs to take effect: an npm project's .npmrc age gate is ignored by npm
// older than npmMinReleaseAgeVersion. pnpm, Yarn and Bun gate release age in
// their own config files and need no probe.
func (m *Module) ToolchainRequirements(config ecosystem.ModuleConfig) []ecosystem.ToolchainRequirement {
	if config.PM("npm") != "npm" {
		return nil
	}
	return []ecosystem.ToolchainRequirement{{
		Binary:     "npm",
		VersionArg: "--version",
		MinVersion: npmMinReleaseAgeVersion,
		Setting:    ".npmrc min-release-age",
	}}
}

// jsLintExtensions is the file pattern the eslint hook lints: JavaScript and
// TypeScript, including their module and JSX variants. git-hooks.nix's
// default (`\.js$`) skips TypeScript entirely.
const jsLintExtensions = `\.(c|m)?[jt]sx?$`

// prettierTypes are the pre-commit (identify) file types the prettier hook
// formats. git-hooks.nix's default is every text file, which rewrites
// qsdev-generated YAML, JSON and Markdown and fails the commit, so the hook
// is limited to the JavaScript/TypeScript sources and stylesheets this module
// covers.
var prettierTypes = []string{"javascript", "jsx", "ts", "tsx", "css", "scss", "less"}

// PreCommitHooks returns pre-commit hook definitions for the JavaScript/TypeScript
// ecosystem. The eslint and prettier hooks are enabled only for projects that
// use those tools (a config file or a dependency, recorded by Detect): forcing
// them onto other projects fails every commit, since eslint without a config
// exits with an error and prettier reformats files the project never
// formatted. When the project depends on a tool, the hook runs the project's
// own copy so its version and plugins match. A project in a subdirectory
// (frontend/) gets hooks that run there instead; see subprojectHook.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	var hooks []ecosystem.HookConfig
	if src := config.Extra(ExtraPrettier, ""); src != "" {
		hooks = append(hooks, ecosystem.HookConfig{
			ID:            "prettier",
			Name:          "prettier",
			Description:   "Format JavaScript/TypeScript code with Prettier",
			Entry:         "prettier --write --list-different",
			Language:      "node",
			TypesOr:       prettierTypes,
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
			Settings:      nodeModulesBinPath(src, "prettier", nil),
		})
		if dir := config.Directory(); dir != "" {
			// git-hooks.nix's defaults: --ignore-unknown --list-different --write.
			hooks[len(hooks)-1] = subprojectHook(hooks[len(hooks)-1], dir, src, "", "--ignore-unknown --list-different --write")
		}
	}
	if src := config.Extra(ExtraESLint, ""); src != "" {
		hooks = append(hooks, ecosystem.HookConfig{
			ID:            "eslint",
			Name:          "eslint",
			Description:   "Lint JavaScript/TypeScript code with ESLint",
			Entry:         "eslint --fix",
			Language:      "node",
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
			Settings:      nodeModulesBinPath(src, "eslint", map[string]string{"extensions": jsLintExtensions}),
		})
		if dir := config.Directory(); dir != "" {
			hooks[len(hooks)-1] = subprojectHook(hooks[len(hooks)-1], dir, src, ".*"+jsLintExtensions, "--fix")
		}
	}
	return hooks
}

// nodeModulesBinPath adds a git-hooks.nix binPath setting pointing at the
// project's node_modules copy of bin when the project depends on it (src is
// toolFromNodeModules). It returns nil when there is nothing to set.
func nodeModulesBinPath(src, bin string, settings map[string]string) map[string]string {
	if src != toolFromNodeModules {
		return settings
	}
	if settings == nil {
		settings = make(map[string]string, 1)
	}
	settings["binPath"] = "./node_modules/.bin/" + bin
	return settings
}

// subprojectHookScript runs a tool from the JavaScript project directory
// (%[1]s) on the staged files there, which pre-commit passes relative to the
// repository root. Running from that directory is what makes the tool find
// its configuration: ESLint 9 looks for eslint.config.* in the working
// directory only, and prettier reads .prettierignore from it.
const subprojectHookScript = `cd %[1]s || exit 1
files=()
for f in "$@"; do
  files+=("${f#%[1]s/}")
done
exec %[2]s %[3]s "${files[@]}"`

// subprojectHook turns a git-hooks.nix built-in hook into a script hook for a
// JavaScript project in the subdirectory dir. The built-in hooks cannot
// change directory, and the project's tool and configuration live in dir,
// so they would lint every JavaScript file in the repository with the wrong
// (or no) configuration. The hook keeps the built-in's ID, so it replaces
// that definition, and runs only on files under dir whose path, relative to
// dir, matches pathPattern (any file when empty). src says where the tool
// comes from, as for nodeModulesBinPath: the project's node_modules, or the
// nixpkgs build otherwise.
func subprojectHook(hook ecosystem.HookConfig, dir, src, pathPattern, args string) ecosystem.HookConfig {
	bin := hook.ID
	hook.NixPackage = hook.ID
	if src == toolFromNodeModules {
		bin = "./node_modules/.bin/" + hook.ID
		hook.NixPackage = ""
	}
	hook.Entry = bin + " " + args
	hook.Script = fmt.Sprintf(subprojectHookScript, dir, bin, args)
	hook.Language = "system"
	hook.Files = "^" + regexp.QuoteMeta(dir) + "/" + pathPattern
	hook.BuiltIn = false
	hook.Settings = nil
	return hook
}

// remotePackageExecDenyRules block every package manager's "download and run
// a package" command. Each one fetches a package from the registry and
// executes it immediately, bypassing lockfiles and the package-guard age
// gate, so denying only npx would leave the same capability open through the
// others. These mirror the catalog's npx and remote_package_exec deny sets.
var remotePackageExecDenyRules = []string{
	"Bash(npx *)",
	"Bash(pnpm dlx *)",
	"Bash(pnpx *)",
	"Bash(yarn dlx *)",
	"Bash(bunx *)",
	"Bash(bun x *)",
	"Bash(npm exec *)",
	"Bash(npm x *)",
	"Bash(deno x *)",
	"Bash(deno run *npm:*)",
	"Bash(deno run *jsr:*)",
	"Bash(deno serve *npm:*)",
	"Bash(deno serve *jsr:*)",
	"Bash(deno npm:*)",
	"Bash(deno jsr:*)",
	"Bash(deno watch *npm:*)",
	"Bash(deno watch *jsr:*)",
	"Bash(deno -* npm:*)",
	"Bash(deno -* jsr:*)",
	"Bash(deno -* x *)",
}

// DenyRules returns Claude Code deny-rule patterns for the JavaScript/TypeScript ecosystem.
// These cover ALL four package managers regardless of which one is detected,
// plus pipe-to-shell patterns that are common JS supply chain attack vectors.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	// Package install commands (npm/pnpm/yarn/bun add/install) are handled by
	// base ask rules + package-guard hook. Only hard-deny patterns here that
	// must never execute regardless of hook validation.
	rules := slices.Clone(remotePackageExecDenyRules)
	return append(rules, ecosystem.PipeToShellDenyRules()...)
}

// CICommands returns CI pipeline commands for the JavaScript/TypeScript ecosystem.
// The frozen install command depends on the detected package manager and runs
// in the JavaScript project directory. npm projects also get an `npm audit`
// scan step: the generated .npmrc's audit-level only sets `npm audit`'s exit
// code (`npm ci` and `npm install` never fail on audit results), so without
// this step nothing gates on it.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	pm := config.PM("npm")

	var installCmd string
	switch pm {
	case "npm":
		installCmd = "npm ci --ignore-scripts"
	case "pnpm":
		installCmd = "pnpm install --frozen-lockfile"
	case "yarn":
		installCmd = "yarn install --immutable"
		if config.Extra(ExtraYarnClassic, "") == "true" {
			// --immutable is Berry-only; Yarn Classic spells it
			// --frozen-lockfile.
			installCmd = "yarn install --frozen-lockfile --ignore-scripts"
		}
	case "bun":
		installCmd = "bun install --frozen-lockfile"
	default:
		pm = "npm"
		installCmd = "npm ci --ignore-scripts"
	}

	cmds := []ecosystem.CICommand{
		{
			Name:        fmt.Sprintf("%s-install", pm),
			Command:     config.InDirectoryCommand(installCmd),
			Description: fmt.Sprintf("Install dependencies using %s with frozen lockfile", pm),
			Phase:       ecosystem.CIPhaseInstall,
		},
	}
	if pm == "npm" {
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "npm-audit",
			Command:     config.InDirectoryCommand("npm audit --audit-level=" + npmAuditLevel),
			Description: fmt.Sprintf("Fail on known %s-or-higher severity vulnerabilities in dependencies", npmAuditLevel),
			Phase:       ecosystem.CIPhaseScan,
		})
	}
	return cmds
}

// PackageManagers returns metadata about all JavaScript package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:           "npm",
			LockFile:       "package-lock.json",
			InstallCommand: "npm install",
		},
		{
			Name:           "pnpm",
			LockFile:       "pnpm-lock.yaml",
			InstallCommand: "pnpm install",
		},
		{
			Name:           "yarn",
			LockFile:       "yarn.lock",
			InstallCommand: "yarn install",
		},
		{
			Name:           "bun",
			LockFile:       "bun.lock",
			InstallCommand: "bun install",
		},
	}
}

// WizardFields returns additional wizard form fields for JavaScript/TypeScript configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         types.SettingVersion,
			Label:       "Node.js version",
			Description: "The Node.js major version; leave empty for the newest LTS",
			Type:        ecosystem.FieldTypeInput,
			Placeholder: "22",
		},
		{
			Key:         types.SettingPackageManager,
			Label:       "Package manager",
			Description: "Select the JavaScript package manager for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "npm", Value: "npm"},
				{Label: "pnpm", Value: "pnpm"},
				{Label: "Yarn", Value: "yarn"},
				{Label: "Bun", Value: "bun"},
			},
			Default:  "npm",
			Required: true,
		},
		{
			Key:         "typescript",
			Label:       "TypeScript",
			Description: "Enable TypeScript support",
			Type:        ecosystem.FieldTypeConfirm,
			Default:     "false",
		},
	}
}

// VerificationCommands returns project verification commands for the
// JavaScript/TypeScript ecosystem, run in the JavaScript project directory.
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	pm := config.PM("npm")
	vc := ecosystem.VerificationCommands{
		Build:  []string{pm + " run build"},
		Test:   []string{pm + " test"},
		Lint:   []string{pm + " run lint"},
		Format: []string{"prettier --check ."},
	}
	if pm == "bun" {
		vc.Test = []string{"bun run test"}
	}
	for _, cmds := range [][]string{vc.Build, vc.Test, vc.Lint, vc.Format} {
		for i, cmd := range cmds {
			cmds[i] = config.InDirectoryCommand(cmd)
		}
	}
	return vc
}

// ManifestFiles returns manifest file metadata for the JavaScript/TypeScript
// ecosystem, as paths in the JavaScript project directory.
func (m *Module) ManifestFiles(config ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	pm := config.PM("npm")
	info := ecosystem.ManifestFileInfo{
		Path:           "package.json",
		Ecosystem:      pm,
		VSSupported:    true,
		LockFilePolicy: ecosystem.LockFilePolicyRequired,
	}
	switch pm {
	case "npm":
		info.LockFile = "package-lock.json"
	case "pnpm":
		info.LockFile = "pnpm-lock.yaml"
	case "yarn":
		info.LockFile = "yarn.lock"
	case "bun":
		info.LockFile = "bun.lock"
	default:
		info.LockFile = "package-lock.json"
	}
	info.Path = config.InDirectory(info.Path)
	info.LockFile = config.InDirectory(info.LockFile)
	return []ecosystem.ManifestFileInfo{info}
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to JavaScript/TypeScript projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/typescript", "p/javascript", "p/react", "p/nextjs", "p/owasp-top-ten", "p/xss"}
}
