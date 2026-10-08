package devinit

import (
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
)

// gatedRoot returns a root carrying the human gate the real tree has, with
// cmds as its subcommands.
func gatedRoot(cmds ...*cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "qsdev", SilenceUsage: true}
	cmdutil.InstallHumanGate(root)
	root.AddCommand(cmds...)
	return root
}
