package mcpregistry

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// LaunchSpec is a server definition qsdev vouches for: a stdio command line or
// a remote endpoint, with its ${VAR} templates unexpanded.
type LaunchSpec struct {
	Command string
	Args    []string
	Env     map[string]string
	Headers map[string]string
	URL     string
}

// untrustedRemoteReason explains why an HTTP entry matching no trusted
// definition is not probed.
const untrustedRemoteReason = "untrusted remote endpoint: not probed (env references would be sent to it)"

// MatchesTrusted reports whether cfg is exactly one of specs, comparing the
// templates before expansion. A stdio entry must match the command, args and
// env; a remote entry (URL set) must match the URL, headers and env, since
// those decide where the expanded references are sent.
func MatchesTrusted(cfg mcphealth.ServerConfig, specs []LaunchSpec) bool {
	for _, s := range specs {
		if !maps.Equal(s.Env, cfg.Env) {
			continue
		}
		if cfg.URL != "" {
			if s.URL == cfg.URL && maps.Equal(s.Headers, cfg.Headers) {
				return true
			}
			continue
		}
		if s.Command != "" && s.Command == cfg.Command && slices.Equal(s.Args, cfg.Args) {
			return true
		}
	}
	return false
}

// PartitionTrusted splits servers into those matching a trusted definition of
// the same name, returned with ExpandEnv set so they are probed as Claude Code
// would run them, and the rest, mapped to a human-readable reason they were
// not probed. The input map is not modified.
func PartitionTrusted(servers map[string]mcphealth.ServerConfig, trusted map[string][]LaunchSpec) (probe map[string]mcphealth.ServerConfig, skipped map[string]string) {
	probe = make(map[string]mcphealth.ServerConfig, len(servers))
	skipped = make(map[string]string)
	for name, cfg := range servers {
		switch {
		case MatchesTrusted(cfg, trusted[name]):
			cfg.ExpandEnv = true
			probe[name] = cfg
		case cfg.URL != "":
			skipped[name] = untrustedRemoteReason
		default:
			cmdline := strings.Join(append([]string{cfg.Command}, cfg.Args...), " ")
			skipped[name] = fmt.Sprintf("command %q matches no trusted definition", cmdline)
		}
	}
	return probe, skipped
}
