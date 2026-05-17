package devinit

const scaffoldMainGoTmpl = `package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"

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
	devinit.Configure(devinit.WithDetectProjectType(true))

	cmd.Main()
}
`

const scaffoldGoModTmpl = `module {{.Module}}

go 1.24

require (
	github.com/Quantum-Serendipity/qsdev v0.6.1
)
`

const scaffoldMakefileTmpl = `MODULE  := {{.Module}}
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X $(MODULE)/internal/version.version=$(VERSION) \
           -X $(MODULE)/internal/version.commit=$(COMMIT)

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
      - -X {{.Module}}/internal/version.version={{"{{"}} .Version {{"}}"}}
      - -X {{.Module}}/internal/version.commit={{"{{"}} .Commit {{"}}"}}
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
