package devinit

import (
	"runtime/debug"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/Quantum-Serendipity/qsdev/internal/version"
)

// gdevModulePath is the framework module qsdev builds on. qsdev requires it at
// a placeholder version satisfied only by a replace directive, and Go ignores
// replace directives of dependencies, so every scaffolded instance must repeat
// the replacement in its own go.mod.
const gdevModulePath = "fastcat.org/go/gdev"

// defaultGdevReplacement is the replacement target used when the running
// binary carries no module build info (e.g. some test binaries). It must match
// the replace directive in qsdev's go.mod; a test enforces this.
const defaultGdevReplacement = "github.com/Quantum-Serendipity/gdev v0.0.0-20260515232304-7ffd0edf72e3"

// scaffoldGdevReplacement returns the "path version" replacement target for
// gdev, taken from the running binary's build info so it always matches the
// gdev this qsdev was built against.
func scaffoldGdevReplacement() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range bi.Deps {
			// A directory replacement (no version) is only meaningful on the
			// machine that built qsdev, so fall back to the published fork.
			if dep.Path == gdevModulePath && dep.Replace != nil && dep.Replace.Version != "" {
				return dep.Replace.Path + " " + dep.Replace.Version
			}
		}
	}
	return defaultGdevReplacement
}

// scaffoldQsdevVersion returns the qsdev module version the scaffold should
// require: the running release (so the scaffold builds against the framework
// that generated it), or "" for development builds, in which case the require
// is omitted and `go mod tidy` resolves the latest release.
func scaffoldQsdevVersion() string {
	v := version.Info().Version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return v
}

const scaffoldMainGoTmpl = `package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

func main() {
	instance.SetBranding(branding.Config{
		AppName:       "{{.AppName}}",
		ConfigFile:    ".{{.AppName}}.yaml",
		LocalConfig:   ".{{.AppName}}.local.yaml",
		StateDir:      ".{{.AppName}}",
		EnvLogVar:     "{{.AppNameUpper}}_LOG",
		EnvLogDirVar:  "{{.AppNameUpper}}_LOG_DIR",
		EnvNoUpdate:   "{{.AppNameUpper}}_NO_UPDATE_CHECK",
		EnvPrefix:     "{{.AppNameUpper}}_",
		LogFilePrefix: "{{.AppName}}-",
		TempPrefix:    ".{{.AppName}}-tmp-",
		GitHubOwner:   "{{.GitHubOwner}}",
		GitHubRepo:    "{{.GitHubRepo}}",
	})
	bootstrap.Configure(
		bootstrap.WithSteps(
			devenv.InstallDevenvStep(),
			devenv.InstallDirenvStep(),
			claudecode.InstallClaudeStep(),
		),
	)

	devenv.Configure(devenv.WithDirenv(true))
	claudecode.Configure(claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard))
	devinit.Configure(
		devinit.WithDetectProjectType(true),
		devinit.WithPlanPreview(true),
	)

	// Same runtime as qsdev itself: MCP framework adapters, external-log
	// providers, the release version stamped via -ldflags (see Makefile), the
	// project's add-or-tighten-only .{{.AppName}}/defaults.yaml, the
	// self-update, logs and report commands, and redacting session logging.
	instance.Main()
}
`

const scaffoldGoModTmpl = `module {{.Module}}

go 1.24
{{- if .QsdevVersion}}

require github.com/Quantum-Serendipity/qsdev {{.QsdevVersion}}
{{- end}}

// qsdev builds on a gdev fork reachable only through a replace directive, and
// Go ignores replace directives of dependencies, so it is repeated here.
replace fastcat.org/go/gdev => {{.GdevReplacement}}
`

const scaffoldMakefileTmpl = `MODULE  := {{.Module}}
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
# The framework reads its version from qsdev's version package (self-update,
# version ratchet, generated state), so that is the package to stamp.
VERSION_PKG := {{.VersionPackage}}
LDFLAGS := -X $(VERSION_PKG).version=$(VERSION) \
           -X $(VERSION_PKG).commit=$(COMMIT)

.PHONY: build test lint clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/{{.AppName}} ./cmd/{{.AppName}}

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run

clean:
	rm -rf bin/
`

const scaffoldGoreleaserTmpl = `version: 2

builds:
  - id: {{.AppName}}
    main: ./cmd/{{.AppName}}
    binary: {{.AppName}}
    env:
      - CGO_ENABLED=0
    ldflags:
      - -s -w
      - -X {{.VersionPackage}}.version={{"{{"}} .Version {{"}}"}}
      - -X {{.VersionPackage}}.commit={{"{{"}} .Commit {{"}}"}}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    name_template: "{{.AppName}}_{{"{{"}} .Version {{"}}"}}_{{"{{"}} .Os {{"}}"}}_{{"{{"}} .Arch {{"}}"}}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^chore:"
      - "^test:"
`

const scaffoldReadmeTmpl = `# {{.AppName}}

A security-hardened development environment tool built on [qsdev](https://github.com/Quantum-Serendipity/qsdev).

## Quick Start

` + "```" + `bash
go build ./cmd/{{.AppName}}
./{{.AppName}} init --yes
` + "```" + `

## Commands

` + "```" + `
{{.AppName}} init        # Generate secure dev environment
{{.AppName}} status      # Security posture assessment
{{.AppName}} check       # CI enforcement checks
{{.AppName}} update      # Update configs
{{.AppName}} teardown    # Remove all configuration
` + "```" + `

## Build

` + "```" + `bash
make build
make test
make lint
` + "```" + `

## License

[Apache-2.0](LICENSE)
`

const scaffoldGitignoreTmpl = `# Binaries
bin/
{{.AppName}}
{{.AppName}}.exe

# Go
*.test
*.out
vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local
.direnv/
.devenv/
`
