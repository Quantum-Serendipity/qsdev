package cmdutil

import (
	"strings"

	"github.com/spf13/cobra"
)

// ReadOnlyAnnotation marks a command whose read-only invocation starts no
// process other than procexec's declared local probes (so it runs no project
// code and fetches no packages), makes no network request and mutates no
// project or user state. Its value is the boolean flag that makes the command
// read-only, or "" when its default invocation is.
const ReadOnlyAnnotation = "qsdev.readonly"

// ReadOnlyLostByAnnotation lists, comma-separated, the flags that take a
// read-only command out of the contract (e.g. "probe" on `mcp status`).
const ReadOnlyLostByAnnotation = "qsdev.readonly.lostby"

// MarkReadOnly declares cmd read-only: in its default invocation when
// viaFlag is "", otherwise only when invoked with --viaFlag (e.g.
// "dry-run"). lostBy names the flags that leave the contract by starting,
// dialing or fetching something (e.g. "probe", "online"); a consumer must not
// treat an invocation that sets one of them as read-only. Tests derive their
// invocations from this mark, so a read-only command cannot silently drop out
// of the no-exec contract. It returns cmd so it can wrap a constructor.
func MarkReadOnly(cmd *cobra.Command, viaFlag string, lostBy ...string) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[ReadOnlyAnnotation] = viaFlag
	if len(lostBy) > 0 {
		cmd.Annotations[ReadOnlyLostByAnnotation] = strings.Join(lostBy, ",")
	}
	return cmd
}

// ReadOnlyLostBy returns the flags MarkReadOnly recorded as taking cmd out of
// the read-only contract.
func ReadOnlyLostBy(cmd *cobra.Command) []string {
	v := cmd.Annotations[ReadOnlyLostByAnnotation]
	if v == "" {
		return nil
	}
	return strings.Split(v, ",")
}

// ReadOnlyArgs returns the arguments, relative to the root command, that
// invoke cmd in its read-only form, and false when cmd is not marked.
func ReadOnlyArgs(cmd *cobra.Command) ([]string, bool) {
	viaFlag, ok := cmd.Annotations[ReadOnlyAnnotation]
	if !ok {
		return nil, false
	}
	args := strings.Fields(cmd.CommandPath())[1:]
	if viaFlag != "" {
		args = append(args, "--"+viaFlag)
	}
	return args, true
}

// ReadOnlyInvocation reports whether cmd, with the flags it was parsed with,
// runs in its read-only form: it is marked (MarkReadOnly), the flag that
// makes it read-only, if any, is set, and no flag that leaves the contract
// was given.
func ReadOnlyInvocation(cmd *cobra.Command) bool {
	viaFlag, ok := cmd.Annotations[ReadOnlyAnnotation]
	if !ok {
		return false
	}
	if viaFlag != "" {
		f := cmd.Flags().Lookup(viaFlag)
		if f == nil || f.Value.String() != "true" {
			return false
		}
	}
	for _, name := range ReadOnlyLostBy(cmd) {
		if cmd.Flags().Changed(name) {
			return false
		}
	}
	return true
}
