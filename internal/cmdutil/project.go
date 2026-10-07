package cmdutil

import (
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
)

// RootAnnotation records how a command picks its project root. Its only
// value is RootHere; an unmarked command acts on the enclosing project.
const RootAnnotation = "qsdev.root"

// RootHere is the RootAnnotation value of a command that creates a project
// where it is run (projectctx.Here).
const RootHere = "here"

// MarkRootHere declares that cmd and its subcommands act on the working
// directory itself rather than on the project enclosing it: they create (or
// re-initialize) a project where they are run. It returns cmd so it can wrap
// a constructor.
func MarkRootHere(cmd *cobra.Command) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[RootAnnotation] = RootHere
	return cmd
}

// RootMode returns the projectctx.Mode cmd resolves its project with: Here
// when cmd or an ancestor is marked with MarkRootHere, Enclosing otherwise.
func RootMode(cmd *cobra.Command) projectctx.Mode {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[RootAnnotation] == RootHere {
			return projectctx.Here
		}
	}
	return projectctx.Enclosing
}

// Project returns the project cmd acts on: the projectctx.Context the process
// initializer resolved once for the executing command and stored in its
// context, or, for a command run on its own (as in an addon unit test), the
// project resolved from the working directory with cmd's RootMode.
func Project(cmd *cobra.Command) (projectctx.Context, error) {
	if ctx := cmd.Context(); ctx != nil {
		if pc, ok := projectctx.FromContext(ctx); ok {
			return pc, nil
		}
	}
	return projectctx.ResolveWorkingDir(RootMode(cmd))
}
