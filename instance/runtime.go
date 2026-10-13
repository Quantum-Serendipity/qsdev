package instance

import (
	"fmt"
	"log/slog"
	"sync"

	gdevinstance "fastcat.org/go/gdev/instance"

	// External-log providers register themselves with the extlog registry.
	// Importing them here gives every tool built on this package the same
	// provider set as qsdev without reaching into internal packages.
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers"

	// Universal MCP server framework adapters. The concrete adapters delegate
	// to addon packages (e.g. addons/claudecode) and so MUST NOT be imported by
	// the mcpserve server package itself — that would create an import cycle.
	// internal/mcpserve/adapters is the single list of them; they are wired in
	// explicitly by RegisterFrameworkAdapters rather than self-registering from
	// init(), so registration order is visible.
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
)

// VersionPackage is the import path whose version and commit variables a
// release build stamps with -ldflags, e.g.
//
//	-X github.com/Quantum-Serendipity/qsdev/internal/version.version=v1.2.3
//
// Downstream tools must stamp this package (not their own module path): the
// self-update check, the version ratchet and generated state all read it.
const VersionPackage = "github.com/Quantum-Serendipity/qsdev/internal/version"

var registerAdaptersOnce sync.Once

// RegisterFrameworkAdapters wires the universal MCP server's framework
// adapters into the default registry. It is safe to call more than once.
func RegisterFrameworkAdapters() {
	registerAdaptersOnce.Do(func() {
		RegisterFrameworkAdaptersInto(spi.DefaultRegistry())
	})
}

// RegisterFrameworkAdaptersInto registers every shipped framework adapter
// (adapters.All) into reg. A duplicate-id error can only mean an adapter was
// listed twice, or reg already holds them — a build wiring mistake worth
// surfacing loudly, so it panics.
func RegisterFrameworkAdaptersInto(reg *spi.AdapterRegistry) {
	for _, a := range adapters.All() {
		if err := reg.Register(a); err != nil {
			panic(fmt.Sprintf("registering framework adapter %q: %v", a.ID(), err))
		}
	}
}

// ApplyBuildVersion propagates the version stamped into VersionPackage to the
// framework's version override, so `version` output and update checks report
// the release rather than "dev". Development builds, and tools that already
// set their own version with SetVersionOverride, are left untouched.
func ApplyBuildVersion() {
	if ver, commit, ok := buildVersionOverride(version.Info(), versionOverridden.Load()); ok {
		gdevinstance.SetVersionOverride(ver, commit)
	}
}

// buildVersionOverride returns the version override ApplyBuildVersion sets
// for build info vi, and false when it sets none: for development builds and
// when the tool set its own version (overridden).
func buildVersionOverride(vi version.BuildInfo, overridden bool) (ver, commit string, ok bool) {
	if overridden || vi.Version == "dev" || vi.Version == "(devel)" {
		return "", "", false
	}
	return vi.Version, vi.Commit, true
}

// useProjectDefaults points the catalog at the project root the executing
// command resolved (Runtime.initCommand), so that project's committed
// .qsdev/defaults.yaml applies on top of the built-in defaults and under the
// user's own defaults file: a Here-mode command (init, devenv init) takes the
// working directory's layer, every other command the enclosing trusted
// project's. The project file may only add deny rules and hooks or raise
// compliance, and must pass the project trust rule (see
// catalog.ProjectConfigFile); anything else stops the command with an error.
// Nothing loads the catalog before the command is resolved; should something
// have loaded it for another root, catalog.SetProjectRoot poisons it and
// every catalog consumer fails closed.
//
// It also pins the org overlay (catalog.UseOrgConfigPin) for the same root
// to the one a human approved with the sensitive 'defaults pin' command, or
// without a pin to the account's home overlay, for every run: whether a
// human runs the CLI cannot be told reliably (an agent can drop its session
// marker and fake a terminal), so an agent's command, or a file it wrote that
// sets <EnvPrefix>ORG_CONFIG, cannot point a regeneration at an overlay of its
// own.
func useProjectDefaults(root string) {
	if err := catalog.SetProjectRoot(root); err != nil {
		// The catalog is now poisoned: every consumer fails closed
		// rather than applying another project's defaults.
		slog.Warn("cannot apply project defaults", "root", root, "error", err)
	}
	pin, err := catalog.LoadOrgConfigPin(root)
	if err != nil {
		slog.Warn("reading no pinned org overlay; using the account's home overlay", "error", err)
	}
	catalog.UseOrgConfigPin(root, pin)
}
