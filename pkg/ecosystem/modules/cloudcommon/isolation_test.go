package cloudcommon

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// TestCLIConfigIsolation checks, per provider, the variable that relocates
// the CLI's configuration directory, the per-project directory under the
// gitignored .qsdev/, the devenv.nix lines that set it, and that the
// directory is masked from the agent's Read tool and `cat` (W135).
func TestCLIConfigIsolation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		provider CloudProvider
		envVar   string
		dir      string
		line     string
	}{
		{Azure, "AZURE_CONFIG_DIR", ".qsdev/cloud/azure", `  env.AZURE_CONFIG_DIR = lib.mkDefault "${config.devenv.root}/.qsdev/cloud/azure";`},
		{GCP, "CLOUDSDK_CONFIG", ".qsdev/cloud/gcp", `  env.CLOUDSDK_CONFIG = lib.mkDefault "${config.devenv.root}/.qsdev/cloud/gcp";`},
		// AWS keeps its SSO and CLI caches under ~/.aws whatever
		// AWS_CONFIG_FILE says, so it has no directory to relocate.
		{AWS, "", "", ""},
		{CloudProvider("unknown"), "", "", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			t.Parallel()
			if got := CLIConfigDirEnvVar(tt.provider); got != tt.envVar {
				t.Errorf("CLIConfigDirEnvVar = %q, want %q", got, tt.envVar)
			}
			if got := ProjectCLIConfigDir(tt.provider); got != tt.dir {
				t.Errorf("ProjectCLIConfigDir = %q, want %q", got, tt.dir)
			}
			frag := IsolatedCLIConfigFragment(tt.provider)
			if tt.line == "" {
				if frag != "" {
					t.Errorf("IsolatedCLIConfigFragment = %q, want empty", frag)
				}
				return
			}
			if !strings.Contains(frag, tt.line+"\n") {
				t.Errorf("IsolatedCLIConfigFragment lacks %q:\n%s", tt.line, frag)
			}
			for _, line := range strings.Split(strings.TrimSuffix(frag, "\n"), "\n") {
				if line != tt.line && !strings.HasPrefix(line, "  # ") {
					t.Errorf("unexpected non-comment line %q in:\n%s", line, frag)
				}
			}
			if want := "./" + tt.dir + "/**"; !slices.Contains(ReadDenyPaths(tt.provider), want) {
				t.Errorf("ReadDenyPaths(%s) lacks %q: %v", tt.provider, want, ReadDenyPaths(tt.provider))
			}
			rules := BashDenyRules(tt.provider)
			for _, cmd := range []string{"cat " + tt.dir + "/credentials.db", "cat ./" + tt.dir + "/credentials.db"} {
				if !slices.ContainsFunc(rules, func(r string) bool { return denyutil.MatchesBashRule(r, cmd) }) {
					t.Errorf("no BashDenyRules(%s) rule blocks %q: %v", tt.provider, cmd, rules)
				}
			}
		})
	}
}
