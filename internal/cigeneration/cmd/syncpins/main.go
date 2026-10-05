// Command syncpins copies the SHA and tag of every action this repository's
// workflows pin into the matching entries of the cigeneration catalog
// (sha_pins.go). It is run by `go generate ./internal/cigeneration/`, from the
// package directory, after Dependabot bumps actions in .github/workflows.
//
// It is this repository's own code plus its vendored dependencies (through
// internal/cigeneration: pkg/branding, the gdev fork, cobra, pflag and
// golang.org/x/sys). The dependabot-fixup workflow runs it only on
// github_actions branches, which change workflow YAML and never go.mod or
// vendor/, so every dependency is at the version on main and it executes no
// code the bot's bump could have supplied.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func main() {
	src := flag.String("src", "sha_pins.go", "catalog source file to rewrite")
	workflows := flag.String("workflows", filepath.Join("..", "..", ".github", "workflows"), "workflow directory to read pins from")
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("syncpins: ")
	if err := run(*src, *workflows); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(srcPath, workflowsDir string) error {
	pins, err := cigeneration.ParseWorkflowPins(workflowsDir)
	if err != nil {
		return fmt.Errorf("parsing workflow pins: %w", err)
	}
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading catalog: %w", err)
	}
	out, err := cigeneration.SyncActionPins(src, pins)
	if err != nil {
		return fmt.Errorf("syncing %s: %w", srcPath, err)
	}
	if bytes.Equal(out, src) {
		return nil
	}
	if err := fileutil.WriteFileAtomic(srcPath, out, 0o644); err != nil {
		return fmt.Errorf("writing catalog: %w", err)
	}
	return nil
}
