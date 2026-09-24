package cloudcommon

import (
	"fmt"
	"path"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// CLIConfigDirEnvVar returns the variable that relocates a provider CLI's
// whole configuration directory (its logins, token caches and active
// subscription or configuration), or "" when the CLI has none.
//
// AWS has no equivalent: AWS_CONFIG_FILE and AWS_SHARED_CREDENTIALS_FILE move
// only the two INI files, while the SSO token and CLI credential caches stay
// under ~/.aws.
func CLIConfigDirEnvVar(provider CloudProvider) string {
	switch provider {
	case GCP:
		return "CLOUDSDK_CONFIG"
	case Azure:
		return "AZURE_CONFIG_DIR"
	default:
		return ""
	}
}

// ProjectCLIConfigDir returns the project-relative directory, under the
// gitignored .qsdev/ directory, that .qsdev.yaml cloud.isolate_cli_config
// gives the provider's CLI, or "" when the provider's CLI cannot be isolated
// (see CLIConfigDirEnvVar).
func ProjectCLIConfigDir(provider CloudProvider) string {
	if CLIConfigDirEnvVar(provider) == "" {
		return ""
	}
	return path.Join("."+branding.Get().AppName, "cloud", string(provider))
}

// projectCLIConfigReadDeny returns the ReadDeny path masking the provider's
// per-project CLI configuration directory, or nil when it has none. It is
// masked whether or not isolation is on, so turning isolation on never
// depends on regenerating .claude/settings.json.
func projectCLIConfigReadDeny(provider CloudProvider) []string {
	dir := ProjectCLIConfigDir(provider)
	if dir == "" {
		return nil
	}
	return []string{"./" + dir + "/**"}
}

// projectCLIConfigBashDeny returns the `cat` deny rules for the provider's
// per-project CLI configuration directory, matching the rule that covers its
// home-directory counterpart, or nil when it has none. Like
// projectCLIConfigReadDeny they apply whether or not isolation is on.
func projectCLIConfigBashDeny(provider CloudProvider) []string {
	dir := ProjectCLIConfigDir(provider)
	if dir == "" {
		return nil
	}
	return []string{
		"Bash(cat " + dir + "/*)",
		"Bash(cat ./" + dir + "/*)",
	}
}

// IsolatedCLIConfigFragment returns the devenv.nix lines that point the
// provider's CLI at its per-project configuration directory, or "" when the
// provider's CLI cannot be isolated. The value is a lib.mkDefault so a
// definition in devenv.local.nix takes precedence instead of conflicting;
// the CLI creates the directory on first use.
func IsolatedCLIConfigFragment(provider CloudProvider) string {
	envVar := CLIConfigDirEnvVar(provider)
	if envVar == "" {
		return ""
	}
	dir := ProjectCLIConfigDir(provider)
	var b strings.Builder
	b.WriteString("  # cloud.isolate_cli_config: this project's logins, tokens and active\n")
	fmt.Fprintf(&b, "  # account live in %s/ (gitignored), not in the shared home directory.\n", dir)
	fmt.Fprintf(&b, "  env.%s = lib.mkDefault \"${config.devenv.root}/%s\";\n", envVar, ecosystem.NixEscapeString(dir))
	return b.String()
}
