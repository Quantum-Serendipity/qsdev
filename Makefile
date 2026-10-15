.PHONY: build build-all test lint vet archtest bench clean completions dogfood dev-install

MODULE  := github.com/Quantum-Serendipity/qsdev
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

# Architecture rules with a shrink-only baseline (internal/archtest/baseline.txt).
# Set ARCHTEST_BASE to an older baseline.txt to also check that it only shrank.
archtest:
	go test -count=1 -run '^(TestArchitecture|TestBaselineMonotone)$$' ./internal/archtest/

# In-process micro-benchmarks, run on demand: save the output of two runs
# and compare them with benchstat. Not a CI gate; the CPU budgets in
# internal/perfbudget/budgets.yaml are (go test -tags perf -run Perf).
bench:
	go test -run '^$$' -bench '^(BenchmarkSelfprotectEvaluate|BenchmarkCatalogLoad|BenchmarkLoadState)$$' \
	  -benchtime=200x -benchmem ./internal/selfprotect/judge ./internal/catalog ./internal/state

lint: vet archtest
	golangci-lint run

# Check and regenerate a `git archive HEAD` copy of this repository with a
# source-built qsdev; fail on any failed check or drift (the CI dogfood job).
# See CONTRIBUTING.md, "Dogfooding".
dogfood:
	bash scripts/dogfood.sh

# Install a plain source build into $GOPATH/bin. Inside devenv that directory
# precedes the system qsdev on PATH, so the .mcp.json `qsdev mcp serve`
# servers run the code in this tree.
dev-install:
	GOWORK=off go install -mod=vendor ./cmd/qsdev

completions: build
	@mkdir -p dist_share/completions
	./bin/qsdev completion bash > dist_share/completions/qsdev.bash
	./bin/qsdev completion zsh  > dist_share/completions/qsdev.zsh
	./bin/qsdev completion fish > dist_share/completions/qsdev.fish

clean:
	rm -rf bin/ dist/ dist_share/ qsdev
