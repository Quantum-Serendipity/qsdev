package cmdutil

import (
	"strings"

	"github.com/spf13/cobra"
)

// CompleteFrom returns a cobra ValidArgsFunction offering the values from
// list that start with the word being completed. It is a lazy ValidArgs: the
// list is resolved only when completion runs, so building the command tree
// never loads the catalog the values come from, and like ValidArgs it
// completes only the first positional argument. A nil list yields nil,
// leaving cobra's default completion in place.
func CompleteFrom(list func() []string) cobra.CompletionFunc {
	if list == nil {
		return nil
	}
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var out []cobra.Completion
		for _, v := range list() {
			if strings.HasPrefix(v, toComplete) {
				out = append(out, v)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
