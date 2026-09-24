package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// RejectUnknownSubcommands makes every command group in the given trees fail
// on an unknown subcommand. Cobra treats a command without Run/RunE as
// non-runnable and, when invoked with arguments it cannot resolve to a
// subcommand (`qsdev chek`, `qsdev devenv add-pakage jq`), prints its help and
// returns nil, so a typo in a CI step or agent skill exits 0 as if it had
// succeeded. Each group (a non-runnable command with subcommands, including
// the root) is made runnable: its Args validator rejects an unknown
// subcommand — before any pre-run hook runs — and its RunE
// shows help when called bare or as `<group> help [subcommand...]`. A runnable
// group without an Args validator (e.g. `logs`, which lists logs when called
// bare) would otherwise run with the bogus word silently ignored, so it gets a
// validator that rejects positional arguments the same way. Leaf commands and
// runnable groups with their own Args validator are left untouched. Applying
// it twice is harmless, so addons can wrap their commands at registration
// and instance.Main can still walk the whole tree. It returns cmds so it can
// wrap a call directly.
func RejectUnknownSubcommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		rejectUnknownSubcommands(c)
	}
	return cmds
}

func rejectUnknownSubcommands(c *cobra.Command) {
	for _, sub := range c.Commands() {
		rejectUnknownSubcommands(sub)
	}
	if !c.HasSubCommands() {
		return
	}
	if c.Runnable() {
		if c.Args == nil {
			c.Args = rejectPositional
		}
		return
	}
	c.Args = validateGroupArgs
	c.RunE = runGroup
}

// validateGroupArgs is the Args validator installed on non-runnable command
// groups. It accepts a bare invocation and `<group> help [subcommand...]`
// naming a real subcommand (cobra only wires the help command on the root),
// and rejects anything else as an unknown subcommand.
func validateGroupArgs(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if args[0] != "help" {
		return unknownSubcommand(c, args[0])
	}
	target, rest, err := c.Find(args[1:])
	if err != nil {
		return fmt.Errorf("finding help topic: %w", err)
	}
	if len(rest) > 0 {
		return unknownSubcommand(target, rest[0])
	}
	return nil
}

// runGroup is the RunE installed on non-runnable command groups. Its arguments
// have already passed validateGroupArgs, so it only has to show the right help.
func runGroup(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return c.Help()
	}
	target, _, err := c.Find(args[1:])
	if err != nil {
		return fmt.Errorf("finding help topic: %w", err)
	}
	return target.Help()
}

// rejectPositional is the Args validator installed on runnable groups that
// take no positional arguments of their own.
func rejectPositional(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return unknownSubcommand(c, args[0])
}

func unknownSubcommand(c *cobra.Command, name string) error {
	msg := fmt.Sprintf("unknown command %q for %q", name, c.CommandPath())
	if c.SuggestionsMinimumDistance <= 0 {
		c.SuggestionsMinimumDistance = 2 // cobra's own default for root suggestions
	}
	if suggestions := c.SuggestionsFor(name); len(suggestions) > 0 {
		msg += "\n\nDid you mean this?\n\t" + strings.Join(suggestions, "\n\t")
	}
	return fmt.Errorf("%s\nRun '%s --help' for usage", msg, c.CommandPath())
}
