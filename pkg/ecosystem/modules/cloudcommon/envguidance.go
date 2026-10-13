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
// definition in devenv.local.nix or from `qsdev init --env`. A value the user
// exports reaches the shell because the module keeps the names in devenv.yaml
// clean.keep (see EnvHintNames and ecosystem.EnvKeeper).
func EnvGuidanceFragment(displayName string, hints []EnvVarHint) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  # %s: these per-project variables are inherited from your shell\n", displayName)
	b.WriteString("  # through devenv.yaml clean.keep; to pin them for this project, set them\n")
	b.WriteString("  # in devenv.local.nix or pass them to `qsdev init --env KEY=VALUE`:\n")
	for _, h := range hints {
		fmt.Fprintf(&b, "  #   env.%s = \"<%s>\";\n", h.Name, h.Description)
	}
	return b.String()
}

// EnvHintNames returns the variable names of hints in order. Modules that
// document per-project selectors with EnvGuidanceFragment return it from
// KeepEnvVars (ecosystem.EnvKeeper), so the variables the guidance names are
// exactly the ones devenv.yaml clean.keep passes through from the shell.
func EnvHintNames(hints []EnvVarHint) []string {
	names := make([]string, len(hints))
	for i, h := range hints {
		names[i] = h.Name
	}
	return names
}
