package claudecode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
)

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run and manage MCP servers",
		Long: "Run qsdev's universal MCP server (`mcp serve`, whose --module flag " +
			"serves a single tool module such as agent-postmortem) and inspect, " +
			"grade, install and remove the project's MCP servers.",
	}

	// The universal MCP server (internal/mcpserve) is the single allowed
	// dependency edge from addons/claudecode into mcpserve. mcpserve must never
	// import back into addons/claudecode (see internal/mcpserve/doc.go).
	cmd.AddCommand(mcpserve.Command())
	cmd.AddCommand(mcpserve.LegacyModuleCommands()...)

	cmd.AddCommand(mcpStatusCmd())
	cmd.AddCommand(mcpListCmd())
	cmd.AddCommand(mcpGradeCmd())
	cmd.AddCommand(mcpInstallCmd())
	cmd.AddCommand(mcpUpdateCmd())
	cmd.AddCommand(mcpRemoveCmd())
	cmd.AddCommand(mcpHealthCmd())
	return cmd
}

// wireAttestationOnce makes wireAttestation idempotent and safe to call from
// concurrent tests.
var wireAttestationOnce sync.Once

// wireAttestation wires external-attestation verification into the MCP
// compliance grader. It is called explicitly from the addon's initialize (not
// from a package init), so importing this package has no side effects on the
// shared registries. This is the only place mcpregistry and contentsign are
// connected (mcpregistry must not import contentsign, to avoid an import
// cycle).
func wireAttestation() {
	wireAttestationOnce.Do(func() {
		mcpregistry.AttestationChecker = attestationChecker
	})
}

// attestationStore caches the trusted-keys-backed attestation store. The keys
// are loaded once on first use (lazily, to keep disk I/O out of startup), so
// grading many servers in one `mcp grade` run does not re-read and re-parse the
// trusted-keys directory per server.
var (
	attestationStoreOnce sync.Once
	attestationStore     contentsign.AttestationStore
)

// attestationChecker reports whether a server's launched code is attested,
// reusing a single loaded key set across invocations.
func attestationChecker(def *mcpregistry.McpServerDefinition) bool {
	attestationStoreOnce.Do(func() {
		// A missing keys dir yields no keys (not an error); any other load
		// failure is surfaced, since it silently disables attestation.
		keys, err := contentsign.LoadTrustedKeys("")
		if err != nil {
			slog.Warn("loading trusted keys for MCP attestation; no server can be graded attested", "error", err)
		}
		attestationStore = contentsign.AttestationStore{TrustedKeys: keys}
	})
	if len(attestationStore.TrustedKeys) == 0 {
		// Short-circuit: an empty TrustedKeys set would otherwise make
		// AttestationStore reload the keys directory on every IsAttested call.
		return false
	}
	return isServerAttested(context.Background(), attestationStore, def)
}

// isServerAttested reports whether the code a server runs is covered by
// trusted-key attestations. The command binary must be attested, and so must
// every argument naming a local file (a script or module the command
// executes, as in `node server.js`), so a signed interpreter or
// launcher does not attest the file it is told to run. Arguments that are not
// local files (subcommands, package names) cannot be attested here; launchers
// that fetch remote code are held below Attested by the local-only criterion.
func isServerAttested(ctx context.Context, store contentsign.AttestationStore, def *mcpregistry.McpServerDefinition) bool {
	if !store.IsAttested(ctx, def.Command) {
		return false
	}
	for _, arg := range def.Args {
		path, ok := localFileArg(arg)
		if ok && !store.IsAttested(ctx, path) {
			return false
		}
	}
	return true
}

// localFileArg returns the absolute path of the existing regular file an
// argument names, resolved against the working directory the server is
// launched from. Both positional arguments and the value of a --flag=value
// argument are considered, since interpreters also load code from flag values
// (as in `node --require=./preload.js`).
func localFileArg(arg string) (string, bool) {
	if strings.HasPrefix(arg, "-") {
		_, value, ok := strings.Cut(arg, "=")
		if !ok {
			return "", false
		}
		arg = value
	}
	if arg == "" {
		return "", false
	}
	path, err := filepath.Abs(arg)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return path, true
}
