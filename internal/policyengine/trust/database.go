package trust

var knownServers = map[string]McpServerInfo{
	"man-pages": {
		Name:                  "man-pages",
		Command:               "qsdev",
		IsLocalBinary:         true,
		OfflineCapable:        true,
		ControlledUpdates:     true,
		VerifiedInstallSource: true,
		PinnedVersion:         true,
	},
	"mcp-nixos": {
		Name:                  "mcp-nixos",
		Command:               "qsdev",
		IsLocalBinary:         true,
		OfflineCapable:        true,
		ControlledUpdates:     true,
		VerifiedInstallSource: true,
		PinnedVersion:         true,
	},
	"github": {
		Name:                    "github",
		Command:                 "npx",
		ServesCommunityCContent: true,
	},
	"context7": {
		Name:                    "context7",
		Command:                 "npx",
		ServesCommunityCContent: true,
	},
	"filesystem": {
		Name:           "filesystem",
		Command:        "npx",
		IsLocalBinary:  false,
		OfflineCapable: true,
	},
	"postgres": {
		Name:           "postgres",
		Command:        "uvx",
		IsLocalBinary:  false,
		OfflineCapable: true,
	},
	"fetch": {
		Name:                    "fetch",
		Command:                 "uvx",
		ServesCommunityCContent: true,
	},
	"socket": {
		Name:                    "socket",
		Command:                 "npx",
		ServesCommunityCContent: true,
	},
	"semble": {
		Name:                  "semble",
		Command:               "qsdev",
		IsLocalBinary:         true,
		ControlledUpdates:     true,
		VerifiedInstallSource: true,
		PinnedVersion:         true,
	},
}

func KnownServerInfo(name string) (McpServerInfo, bool) {
	info, ok := knownServers[name]
	return info, ok
}

// ResolveServerInfo returns the server description to score for name. The
// configured definition (the project's .mcp.json entry) is authoritative for
// what actually runs. The known-server database contributes its trust signals
// only when the configured command matches the known server's command, so a
// different server cannot inherit a known server's trust by reusing its name.
// A server with no configured definition carries no trust signals and scores
// into the fallback tier.
func ResolveServerInfo(name string, configured *McpServerInfo) McpServerInfo {
	if configured == nil {
		return McpServerInfo{Name: name}
	}

	info := *configured
	info.Name = name
	if known, ok := KnownServerInfo(name); ok && known.Command == configured.Command {
		known.Name = name
		known.Args = configured.Args
		known.Env = configured.Env
		known.Transport = configured.Transport
		info = known
	}
	return info
}
