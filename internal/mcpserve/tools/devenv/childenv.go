package devenv

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

// childEnv returns environ without its credential-bearing variables: the
// environment a process nix_run starts receives. A child that inherited the
// server's environment could print GITHUB_TOKEN or AWS_* with `env`, making
// env_info's withholding moot and leaving only ContentSafety's value-pattern
// redaction between the secret and the caller. A variable is dropped when the
// shared canon names it sensitive (secrets.IsSensitiveName) or env_info
// withholds its value (isSensitiveEnv), so no variable env_info hides reaches a
// child.
//
// The result is never nil: exec.Cmd treats a nil Env as "inherit everything".
func childEnv(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if secrets.IsSensitiveName(name) || isSensitiveEnv(name) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
