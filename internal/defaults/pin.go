package defaults

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// pinCmd returns "defaults pin", which records the org overlay this run
// resolves as the one every later run reads (catalog.RecordOrgConfigPin).
// Without a pin, runs read only the account's home overlay, whatever
// <EnvPrefix>ORG_CONFIG says, so pinning is what lets that variable take
// effect. It is sensitive: only a human at their own terminal may approve an
// overlay, and the self-protection hook blocks it for the agent.
func pinCmd() *cobra.Command {
	var global bool
	b := branding.Get()
	cmd := &cobra.Command{
		Use:   "pin",
		Short: "Approve the org overlay " + b.EnvPrefix + "ORG_CONFIG names for this project (or, with --global, every project)",
		Long: "Records the org overlay this run resolves (" + b.EnvPrefix + "ORG_CONFIG, or the account's home overlay) " +
			"as the one " + b.AppName + " reads for the project, in the account's home configuration directory. " +
			"Without a pin, " + b.AppName + " reads only the account's home overlay and ignores " + b.EnvPrefix +
			"ORG_CONFIG with a warning. An overlay below the project or the temporary directory is refused.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPin(cmd, global)
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "Pin the overlay for every project without a pin of its own")
	return cmdutil.MarkSensitive(cmd, cmdutil.Sensitivity{})
}

func runPin(cmd *cobra.Command, global bool) error {
	root := ""
	if !global {
		var err error
		if root, err = cmdutil.ProjectRoot(); err != nil {
			return err
		}
	}
	path, err := catalog.RecordOrgConfigPin(root)
	if err != nil {
		return err
	}
	scope := "every project without a pin of its own"
	if root != "" {
		scope = root
	}
	if path == "" {
		path = "no overlay"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Pinned the org overlay %s for %s\n", path, scope)
	return nil
}
