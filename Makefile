.PHONY: build build-all test lint vet clean completions

MODULE  := github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
  -X $(MODULE)/internal/version.version=$(VERSION) \
  -X $(MODULE)/internal/version.commit=$(COMMIT) \
  -X $(MODULE)/internal/version.date=$(DATE) \
  -X $(MODULE)/internal/version.builtBy=make

build:
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/qsdev ./cmd/qsdev

build-all:
	@mkdir -p bin
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/qsdev-linux-amd64   ./cmd/qsdev
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/qsdev-linux-arm64   ./cmd/qsdev
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/qsdev-darwin-amd64  ./cmd/qsdev
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/qsdev-darwin-arm64  ./cmd/qsdev
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/qsdev-windows-amd64.exe ./cmd/qsdev

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	golangci-lint run

completions: build
	@mkdir -p dist_share/completions
	./bin/qsdev completion bash > dist_share/completions/qsdev.bash
	./bin/qsdev completion zsh  > dist_share/completions/qsdev.zsh
	./bin/qsdev completion fish > dist_share/completions/qsdev.fish

clean:
	rm -rf bin/ dist/ dist_share/
