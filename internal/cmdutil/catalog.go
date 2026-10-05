package cmdutil

import "github.com/spf13/cobra"

// CatalogOptionalAnnotation marks a command that runs without the defaults
// catalog loading (see MarkCatalogOptional).
const CatalogOptionalAnnotation = "qsdev.catalog"

// MarkCatalogOptional declares that cmd and every subcommand under it run
// even when the defaults catalog fails to load: they diagnose or repair the
// defaults file themselves, or report the failure in-band. It returns cmd so
// it can wrap a constructor.
func MarkCatalogOptional(cmd *cobra.Command) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[CatalogOptionalAnnotation] = "optional"
	return cmd
}

// CatalogRequired reports whether cmd needs the defaults catalog to load
// before it runs: a command a person runs on a project (ProfileInteractive)
// that does more than show help, with no catalog-optional mark on itself or
// an ancestor. Hooks and the MCP server keep their own fail-closed contracts,
// global and unlogged commands act on no project's catalog, and a bare group
// (`qsdev claude`) only prints its usage.
func CatalogRequired(cmd *cobra.Command) bool {
	if ProfileOf(cmd) != ProfileInteractive || helpOnly(cmd) {
		return false
	}
	for c := cmd; c != nil; c = c.Parent() {
		if _, ok := c.Annotations[CatalogOptionalAnnotation]; ok {
			return false
		}
	}
	return true
}
