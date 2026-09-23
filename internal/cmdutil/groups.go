package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// RejectUnknownSubcommands makes every command group in the given trees fail
// on an unknown subcommand. Cobra treats a command without Run/RunE as
// non-runnable and, when invoked with arguments it cannot resolve to a
// subcommand (`qsdev devenv add-pakage jq`), prints its help and returns nil,
// so a typo in a CI step or agent skill exits 0 as if it had succeeded. Each
// group (a non-runnable command with subcommands) is given a RunE that shows
// help when called bare and returns an "unknown command" error otherwise. A
// runnable group without an Args validator (e.g. `logs`, which lists logs when
// called bare) would otherwise run with the bogus word silently ignored, so it
// gets a validator that rejects positional arguments the same way. Leaf
// commands and groups with their own Args validator are left untouched. It
// returns cmds so it can wrap a registration call directly.
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
	c.Args = cobra.ArbitraryArgs
	c.RunE = runGroup
}

// runGroup is the RunE installed on non-runnable command groups: help for a
// bare invocation or `<group> help [subcommand...]` (which cobra only wires on
// the root), an error naming the unknown subcommand otherwise.
func runGroup(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return c.Help()
	}
	if args[0] == "help" {
		target, rest, err := c.Find(args[1:])
		if err != nil {
			return fmt.Errorf("finding help topic: %w", err)
		}
		if len(rest) > 0 {
			return unknownSubcommand(target, rest[0])
		}
		return target.Help()
	}
	return unknownSubcommand(c, args[0])
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
