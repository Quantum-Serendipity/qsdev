package cloudcommon

import (
	"fmt"
	"strings"
)

// EnvVarHint names a per-project environment variable a cloud CLI reads and
// describes the value the user has to supply for it.
type EnvVarHint struct {
	Name        string
	Description string
}

// EnvGuidanceFragment renders devenv.nix comment lines that explain how to set
// the per-project environment variables for a cloud provider.
//
// The variables are documented rather than defined. Real values are
// account-specific, a placeholder value breaks the CLI that reads it (gcloud
// switches to a nonexistent configuration, Terraform rejects a non-UUID
// subscription), and a generated definition would collide with the user's own
// definition in devenv.local.nix or from `qsdev init --env`.
func EnvGuidanceFragment(displayName string, hints []EnvVarHint) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  # %s: set these per-project variables in devenv.local.nix,\n", displayName)
	b.WriteString("  # or pass them to `qsdev init --env KEY=VALUE`:\n")
	for _, h := range hints {
		fmt.Fprintf(&b, "  #   env.%s = \"<%s>\";\n", h.Name, h.Description)
	}
	return b.String()
}
