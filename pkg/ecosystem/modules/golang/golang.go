// Package golang implements the Go ecosystem module for qsdev.
// It detects Go projects by scanning for go.mod, generates devenv.nix fragments
// with security-hardened environment variables, and provides pre-commit hooks,
// CI commands, deny rules, and wizard fields for the Go toolchain.
package golang

import (
	"bufio"
	"fmt"
	goversion "go/version"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.ProxyKeyProvider = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.DependencyDeclarer = (*Module)(nil)
var _ ecosystem.DevenvYamlInputProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// goVersionRe matches the "go X.Y" or "go X.Y.Z" directive in go.mod.
var goVersionRe = regexp.MustCompile(`^go\s+(\d+\.\d+(?:\.\d+)?)`)

// goToolchainRe matches the "toolchain goX.Y.Z" (or "goX.YrcN") directive in
// go.mod, capturing the version without its "go" prefix.
var goToolchainRe = regexp.MustCompile(`^toolchain\s+go(\d+\.\d+(?:\.\d+|rc\d+)?)\s*$`)

// goReleaseRe parses a Go 1.x release version ("1.24", "1.24.1" or
// "1.24rc1"), capturing the minor version and the optional patch or
// release-candidate suffix.
var goReleaseRe = regexp.MustCompile(`^1\.(\d+)(\.\d+|rc\d+)?$`)

// ExtraToolchain is the ModuleConfig extra holding go.mod's toolchain
// directive (for example "1.26.8"). It takes precedence over Version, the go
// directive, as the release the project needs.
const ExtraToolchain = "toolchain"

// ExtraGoSource is the ModuleConfig extra declaring where the shell's Go
// comes from: GoSourceGoOverlay (the default) or GoSourceNixpkgs. It is a
// declared setting (see WizardFields), so .qsdev.yaml can commit it.
const ExtraGoSource = "go_source"

// The values ExtraGoSource accepts.
const (
	// GoSourceGoOverlay keeps nixpkgs' Go when it is new enough and otherwise
	// selects the required release from the go-overlay flake input.
	GoSourceGoOverlay = "go-overlay"
	// GoSourceNixpkgs declares that the project's nixpkgs, with its
	// registered overlays, supplies the required Go: no go-overlay input is
	// generated, and evaluating the shell fails if pkgs.go is older.
	GoSourceNixpkgs = "nixpkgs"
)

// goSourceField declares ExtraGoSource: a wizard field, and so a setting
// .qsdev.yaml may commit as a go extra.
var goSourceField = ecosystem.WizardField{
	Key:         ExtraGoSource,
	Label:       "Go source",
	Description: "Where the shell's Go comes from: go-overlay fetches the required release when nixpkgs' Go is older; nixpkgs declares that nixpkgs and the project's overlays supply it",
	Type:        ecosystem.FieldTypeSelect,
	Options: []ecosystem.WizardOption{
		{Label: "go-overlay (fetch the required release when needed)", Value: GoSourceGoOverlay},
		{Label: "nixpkgs (nixpkgs and the project's overlays supply it)", Value: GoSourceNixpkgs},
	},
	Default: GoSourceGoOverlay,
}

// goOverlayInput is the flake input devenv's languages.go.version selects
// exact Go releases from.
const goOverlayInput = "github:purpleclay/go-overlay"

// Module implements ecosystem.EcosystemModule for the Go programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return ecosystem.NameGo }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Go" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for a go.mod file and extracts the Go version directive.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	modPath := filepath.Join(projectRoot, "go.mod")
	if !fileutil.FileExists(modPath) {
		return ecosystem.DetectionAbsent()
	}

	version, toolchain := parseGoMod(projectRoot)

	evidence := []string{"go.mod found"}
	if version != "" {
		evidence = append(evidence, fmt.Sprintf("go version %s", version))
	}
	suggested := ecosystem.ModuleConfig{Version: version}
	if toolchain != "" {
		evidence = append(evidence, fmt.Sprintf("toolchain go%s", toolchain))
		suggested.Extras = map[string]string{ExtraToolchain: toolchain}
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      ecosystem.ConfidenceCertain,
		Evidence:        evidence,
		SuggestedConfig: suggested,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Go language support with supply-chain security hardening.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	// GOFLAGS is deliberately left unset. Build commands already default to
	// -mod=readonly, which does not stop `go get` or `go mod tidy` from adding
	// dependencies, and an explicit -mod=readonly would override the -mod=vendor
	// default of a module with a vendor/ directory, so builds would bypass the
	// reviewed vendored sources.
	var envVars []ecosystem.NixEnvVar
	if config.RegistryProxy != "" {
		// No ",direct" fallback: the go command moves to the next GOPROXY
		// entry on any 404/410, so a proxy that refuses a module (allow-list,
		// quarantine) would be bypassed by fetching straight from its origin.
		// Private modules matched by GOPRIVATE/GONOPROXY are still fetched
		// directly.
		envVars = append(envVars, ecosystem.NixEnvVar{
			Key:     "GOPROXY",
			Value:   ecosystem.NixString(config.RegistryProxy),
			Comment: "Fetch every public module through the registry proxy, with no direct fallback",
		})
	}
	// GOSUMDB is the variable that controls checksum verification; pinning it
	// overrides an inherited GOSUMDB=off. Go fetches the checksum database
	// through GOPROXY when the proxy supports it. Modules matched by
	// GOPRIVATE/GONOSUMDB remain exempt, as they must be for private code.
	envVars = append(envVars, ecosystem.NixEnvVar{
		Key:     "GOSUMDB",
		Value:   `"sum.golang.org"`,
		Comment: "Verify all public modules against the Go checksum database",
	})

	plan, err := planGoSource(config)
	if err != nil {
		return "", err
	}
	var props []ecosystem.NixProperty
	if p, ok := plan.property(); ok {
		props = append(props, p)
	}
	fragment := ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath: "languages.go",
		Properties: props,
		EnvVars:    envVars,
	})
	if plan.note != "" {
		fragment = "  # " + plan.note + "\n" + fragment
	}
	return fragment, nil
}

// DevenvYamlInputs contributes the go-overlay flake input when the fragment
// sets languages.go.version; devenv refuses to evaluate the version option
// without it. The input and the version line are an invariant pair, both
// decided by planGoSource. An invalid go_source yields no input: the
// fragment returns the error, so generation fails before either is written.
func (m *Module) DevenvYamlInputs(config ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	if plan, err := planGoSource(config); err != nil || !plan.goOverlay() {
		return nil
	}
	return []ecosystem.DevenvInput{{URL: goOverlayInput, Follows: "nixpkgs", Options: []string{"languages.go.version"}}}
}

// goSourcePlan is how the generated shell provides the Go release the
// project requires.
type goSourcePlan struct {
	release string // required release; "" leaves nixpkgs' Go unconstrained
	note    string // see requiredGoRelease
	source  string // GoSourceGoOverlay or GoSourceNixpkgs
}

// planGoSource decides, from the required release and the go_source extra,
// both the languages.go property and whether the go-overlay input is needed.
// An unknown go_source is an error: reading it as the default would keep the
// input a misspelled "nixpkgs" was meant to drop.
func planGoSource(config ecosystem.ModuleConfig) (goSourcePlan, error) {
	source := config.Extra(ExtraGoSource, GoSourceGoOverlay)
	if err := goSourceField.CheckValue(source); err != nil {
		return goSourcePlan{}, fmt.Errorf("go extra %s: %w", ExtraGoSource, err)
	}
	release, note := requiredGoRelease(config)
	return goSourcePlan{release: release, note: note, source: source}, nil
}

// goOverlay reports whether the shell selects its Go from go-overlay, which
// needs the go-overlay flake input.
func (p goSourcePlan) goOverlay() bool {
	return p.release != "" && p.source == GoSourceGoOverlay
}

// property returns the languages.go line enforcing the required release, if
// one is known. devenv sets GOTOOLCHAIN=local, so the shell's Go must be at
// least that release. nixpkgs' Go (binary-cached, and what gopls and the
// other Go tools are built against) is kept whenever it is new enough.
// Otherwise, by default, that exact release comes from go-overlay; when the
// project declares go_source=nixpkgs, evaluating the shell fails instead, so
// the declaration cannot silently lie. That package line has priority 999,
// so a plain languages.go.package in devenv.local.nix still overrides it.
// release passed goReleaseRe, so it is safe inside a Nix string.
func (p goSourcePlan) property() (ecosystem.NixProperty, bool) {
	if p.release == "" {
		return ecosystem.NixProperty{}, false
	}
	v := ecosystem.NixString(p.release)
	if p.goOverlay() {
		return ecosystem.NixProperty{
			Key:   "version",
			Value: fmt.Sprintf("lib.mkIf (!(lib.versionAtLeast pkgs.go.version %s)) %s", v, v),
		}, true
	}
	msg := fmt.Sprintf("qsdev: .qsdev.yaml declares %s=%s, but pkgs.go is ${pkgs.go.version}, "+
		"older than the required Go %s; raise it through nixpkgs or a registered overlay, "+
		"or remove %s and run qsdev init --update to fetch Go %s from go-overlay",
		ExtraGoSource, GoSourceNixpkgs, p.release, ExtraGoSource, p.release)
	return ecosystem.NixProperty{
		Key: "package",
		Value: fmt.Sprintf("lib.mkOverride 999 (if lib.versionAtLeast pkgs.go.version %s then pkgs.go else throw \"%s\")",
			v, msg),
	}, true
}

// SecurityConfigs returns generated security configuration files.
// Go's security settings are handled via environment variables in DevenvNixFragment.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// goToolIgnoredDirs matches the paths the go tool leaves out of `./...`:
// testdata and vendor directories and directories starting with "." or "_".
var goToolIgnoredDirs = []string{`(^|/)(testdata|vendor)/`, `(^|/)[._][^/]*/`}

// PreCommitHooks returns pre-commit hook definitions for the Go ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "gofmt",
			Name:          "gofmt",
			Description:   "Format Go source code with gofmt",
			Entry:         "gofmt -l -w",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
			// gofmt -w rewrites the files it is given. An all-files run
			// (prek run -a, devenv test) must not reformat vendored
			// dependencies or testdata fixtures, which go fmt ./... skips too.
			Excludes: goToolIgnoredDirs,
		},
		{
			ID:            "govet",
			Name:          "govet",
			Description:   "Run go vet to detect suspicious constructs",
			Entry:         "go vet ./...",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
			// The built-in hook runs `go vet` in each staged file's directory,
			// so it must skip the directories `./...` skips: testdata (often
			// //go:build ignore fixtures, where go vet fails with "build
			// constraints exclude all Go files"), vendor, and _/. prefixed dirs.
			Excludes: goToolIgnoredDirs,
		},
		{
			ID:            "staticcheck",
			Name:          "staticcheck",
			Description:   "Run staticcheck for advanced static analysis",
			Entry:         "staticcheck ./...",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    "go-tools",
		},
		{
			ID:            "govulncheck",
			Name:          "govulncheck",
			Description:   "Check for known vulnerabilities in Go dependencies",
			Entry:         "govulncheck ./...",
			Language:      "system",
			Types:         []string{"go"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
			NixPackage:    "govulncheck",
		},
	}
}

// CICommands returns CI pipeline commands for the Go ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "go-mod-download",
			Command:     "go mod download",
			Description: "Download Go module dependencies",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "go-mod-verify",
			Command:     "go mod verify",
			Description: "Verify Go module checksums against go.sum",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "govulncheck",
			Command:     "govulncheck ./...",
			Description: "Scan Go dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// ProxyKey returns "go": the generated devenv.nix points GOPROXY at the
// registry proxy.
func (m *Module) ProxyKey(_ ecosystem.ModuleConfig) string { return "go" }

// PackageManagers returns metadata about Go's module system.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "go modules",
			LockFile: "go.sum",
		},
	}
}

// WizardFields returns additional wizard form fields for Go configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         types.SettingVersion,
			Label:       "Go version",
			Description: "The Go release to pin; leave empty to use the release go.mod requires",
			Type:        ecosystem.FieldTypeInput,
			Placeholder: "1.24",
		},
		goSourceField,
	}
}

// VerificationCommands returns build/test/lint/format commands for Go projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build:  []string{"go build ./..."},
		Test:   []string{"go test ./..."},
		Lint:   []string{"go vet ./...", "golangci-lint run"},
		Format: []string{"gofmt -l ."},
	}
}

// ManifestFiles returns manifest file metadata for Go projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{
		{
			Path:           "go.mod",
			Ecosystem:      "go",
			VSSupported:    false,
			LockFile:       "go.sum",
			LockFilePolicy: ecosystem.LockFilePolicyRecommended,
		},
	}
}

// DeclaresDependencies reports whether go.mod has any require directive. A
// module that requires nothing never gets a go.sum (`go mod tidy` does not
// create one), so there is no lock file to enforce.
func (m *Module) DeclaresDependencies(projectRoot string) (bool, error) {
	path := filepath.Join(projectRoot, "go.mod")
	data, err := os.ReadFile(path) //nolint:gosec // project-root manifest
	if err != nil {
		return false, fmt.Errorf("reading go.mod: %w", err)
	}
	mf, err := modfile.ParseLax(path, data, nil)
	if err != nil {
		return false, fmt.Errorf("parsing go.mod: %w", err)
	}
	return len(mf.Require) > 0, nil
}

// DevenvPackages returns standard Go development tool packages, and
// govulncheck, which the CI scan runs whatever the hook tier.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"gopls", "golangci-lint", "delve", "goreleaser", "govulncheck"}
}

// requiredGoRelease returns the minimum Go release the project needs, as a
// go-overlay release name: the newer of Version (the go directive or
// --go-version) and the toolchain extra (go.mod's toolchain directive), so
// neither a toolchain line nor an explicit newer --go-version is dropped. It
// returns "" when no usable version is configured. note is non-empty when a
// configured version is not a Go 1.x release and is ignored, so the fragment
// never carries untrusted text outside a comment.
func requiredGoRelease(config ecosystem.ModuleConfig) (release, note string) {
	var ignored []string
	for _, v := range []string{config.Version, config.Extra(ExtraToolchain, "")} {
		if v == "" {
			continue
		}
		r, ok := goOverlayRelease(v)
		if !ok {
			ignored = append(ignored, strconv.Quote(v))
			continue
		}
		if release == "" || goversion.Compare("go"+r, "go"+release) > 0 {
			release = r
		}
	}
	if len(ignored) > 0 {
		note = fmt.Sprintf("unrecognized Go version %s ignored", strings.Join(ignored, ", "))
		if release == "" {
			note += "; using pkgs.go (latest)"
		}
	}
	return release, note
}

// goOverlayRelease converts a go.mod version ("1.24", "1.24.3", "1.26rc1")
// into the name of the Go release go-overlay publishes it under. A bare
// language version means its first release: "1.Y.0" from Go 1.21 on, and
// "1.Y" before that, when the first release of a minor had no ".0".
func goOverlayRelease(version string) (string, bool) {
	m := goReleaseRe.FindStringSubmatch(version)
	if m == nil {
		return "", false
	}
	minor, err := strconv.Atoi(m[1])
	if err != nil {
		return "", false
	}
	switch suffix := m[2]; {
	case suffix == "" && minor >= 21:
		return version + ".0", true
	case suffix == ".0" && minor < 21:
		return strings.TrimSuffix(version, ".0"), true
	}
	return version, true
}

// parseGoMod reads go.mod in projectRoot and extracts the version from the
// "go X.Y[.Z]" directive and, if present, the "toolchain goX.Y.Z" directive.
// Missing directives, or an unreadable file, yield empty strings.
func parseGoMod(projectRoot string) (version, toolchain string) {
	f, err := os.Open(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return "", ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if m := goVersionRe.FindStringSubmatch(line); m != nil && version == "" {
			version = m[1]
		}
		if m := goToolchainRe.FindStringSubmatch(line); m != nil && toolchain == "" {
			toolchain = m[1]
		}
	}
	return version, toolchain
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Go projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/golang", "p/owasp-top-ten"}
}
