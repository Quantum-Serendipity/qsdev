package instance

import (
	"fmt"
	"sync"

	// External-log providers register themselves with the extlog registry.
	// Importing them here gives every tool built on this package the same
	// provider set as qsdev without reaching into internal packages.
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers"

	// Universal MCP server framework adapters. Concrete adapters under
	// internal/mcpserve/adapters/* delegate to addon packages (e.g.
	// addons/claudecode) and so MUST NOT be imported by the mcpserve server
	// package itself — that would create an import cycle. They are wired in
	// explicitly by RegisterFrameworkAdapters rather than self-registering from
	// init(), so registration order is visible. The aliases avoid colliding
	// with the addon packages of the same name.
	claudecodeadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/claudecode"
	clineadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cline"
	codexadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/codex"
	cursoradapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	windsurfadapter "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/windsurf"
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
// A duplicate-id error can only mean an adapter was listed twice — a build
// wiring mistake worth surfacing loudly.
func RegisterFrameworkAdapters() {
	registerAdaptersOnce.Do(func() {
		reg := spi.DefaultRegistry()
		for _, a := range []spi.FrameworkAdapter{
			claudecodeadapter.New(),
			clineadapter.New(),
			codexadapter.New(),
			cursoradapter.New(),
			windsurfadapter.New(),
		} {
			if err := reg.Register(a); err != nil {
				panic(fmt.Sprintf("registering framework adapter %q: %v", a.ID(), err))
			}
		}
	})
}

// ApplyBuildVersion propagates the version stamped into VersionPackage to the
// framework's version override, so `version` output and update checks report
// the release rather than "dev". Development builds are left untouched.
func ApplyBuildVersion() {
	if vi := version.Info(); vi.Version != "dev" && vi.Version != "(devel)" {
		SetVersionOverride(vi.Version, vi.Commit)
	}
}
