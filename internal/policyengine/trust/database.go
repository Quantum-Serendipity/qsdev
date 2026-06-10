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
		Command:        "npx",
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
