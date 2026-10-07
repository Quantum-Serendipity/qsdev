package cmdutil

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// SensitiveAnnotation marks a command whose invocations (all of them, or
// those its Sensitivity names) weaken or remove a guardrail, so only a human
// may run them. The root's human gate (InstallHumanGate) refuses them from an
// agent session or without a terminal, and the self-protection hook derives
// the commands it blocks from the same marks (SensitiveCommands).
const SensitiveAnnotation = "qsdev.sensitive"

// Sensitivity narrows which invocations of a sensitive command need a human.
// The zero value means every invocation, except a read-only one (see
// MarkReadOnly).
type Sensitivity struct {
	// Flags maps a boolean flag's name to the value that makes an invocation
	// sensitive: {"force": true} for --force, {"strict": false} for
	// --strict=false.
	Flags map[string]bool
	// Args returns the positional arguments that make an invocation
	// sensitive. It is called when a matching invocation is judged, not when
	// the command is built, so it may read the catalog.
	Args func() []string
}

// sensitivities holds the Sensitivity of every command MarkSensitive marked.
var sensitivities sync.Map // *cobra.Command -> Sensitivity

// MarkSensitive marks cmd as sensitive in the invocations s names. It
// returns cmd so it can wrap a constructor.
func MarkSensitive(cmd *cobra.Command, s Sensitivity) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[SensitiveAnnotation] = "true"
	sensitivities.Store(cmd, s)
	return cmd
}

// sensitivityOf returns cmd's Sensitivity and whether cmd is marked.
func sensitivityOf(cmd *cobra.Command) (Sensitivity, bool) {
	if cmd.Annotations[SensitiveAnnotation] != "true" {
		return Sensitivity{}, false
	}
	v, ok := sensitivities.Load(cmd)
	if !ok {
		// Annotated by hand: fail safe, every invocation is sensitive.
		return Sensitivity{}, true
	}
	return v.(Sensitivity), true
}

// IsSensitiveInvocation reports whether running cmd with the parsed flags
// and positional args needs a human.
func IsSensitiveInvocation(cmd *cobra.Command, args []string) bool {
	s, ok := sensitivityOf(cmd)
	if !ok || readOnlyInvocation(cmd) {
		return false
	}
	if len(s.Flags) == 0 && s.Args == nil {
		return true
	}
	for name, want := range s.Flags {
		f := cmd.Flags().Lookup(name)
		if f == nil || !f.Changed {
			continue
		}
		if got, err := strconv.ParseBool(f.Value.String()); err != nil || got == want {
			return true
		}
	}
	if s.Args != nil {
		sensitive := s.Args()
		return slices.ContainsFunc(args, func(a string) bool { return slices.Contains(sensitive, a) })
	}
	return false
}

// readOnlyInvocation reports whether cmd runs in the read-only form
// MarkReadOnly declared through a flag (e.g. --dry-run).
func readOnlyInvocation(cmd *cobra.Command) bool {
	viaFlag := cmd.Annotations[ReadOnlyAnnotation]
	if viaFlag == "" {
		return false
	}
	f := cmd.Flags().Lookup(viaFlag)
	if f == nil || !f.Changed {
		return false
	}
	on, err := strconv.ParseBool(f.Value.String())
	return err == nil && on
}

// SensitiveCommands returns a spec for every sensitive command in root's
// tree, for the self-protection hook to match agent shell commands against.
// Paths are relative to root.
func SensitiveCommands(root *cobra.Command) []cmdscan.CommandSpec {
	var specs []cmdscan.CommandSpec
	var walk func(c *cobra.Command, path [][]string)
	walk = func(c *cobra.Command, path [][]string) {
		if s, ok := sensitivityOf(c); ok && len(path) > 0 {
			specs = append(specs, commandSpec(c, path, s))
		}
		for _, sub := range c.Commands() {
			walk(sub, append(slices.Clone(path), append([]string{sub.Name()}, sub.Aliases...)))
		}
	}
	walk(root, nil)
	return specs
}

// commandSpec translates c's Sensitivity into a spec at path.
func commandSpec(c *cobra.Command, path [][]string, s Sensitivity) cmdscan.CommandSpec {
	spec := cmdscan.CommandSpec{Path: path, Args: s.Args, ReadOnly: infoFlagSpellings(c), ValueFlags: valueFlagSpellings(c)}
	if viaFlag := c.Annotations[ReadOnlyAnnotation]; viaFlag != "" {
		spec.ReadOnly = append(spec.ReadOnly, flagSpellings(c, viaFlag)...)
	}
	for _, name := range slices.Sorted(maps.Keys(s.Flags)) {
		spec.Flags = append(spec.Flags, cmdscan.FlagCond{Spellings: flagSpellings(c, name), Value: s.Flags[name]})
	}
	return spec
}

// infoFlagSpellings returns the spellings of the flags with which c prints
// instead of running: cobra's help flag, which every command has, and
// --version, which prints the root's version or, on a command without a
// version flag, fails as an unknown flag. None does when c does not parse
// its flags, and --version runs c when c defines a version flag of its own
// or ignores unknown flags.
func infoFlagSpellings(c *cobra.Command) []string {
	if c.DisableFlagParsing {
		return nil
	}
	// Cobra adds both just before parsing; adding them here finds the
	// spellings it gives them.
	c.InitDefaultHelpFlag()
	c.InitDefaultVersionFlag()
	spellings := flagSpellings(c, "help")
	switch f := c.Flags().Lookup("version"); {
	case f == nil && !c.FParseErrWhitelist.UnknownFlags,
		f != nil && len(f.Annotations[cobra.FlagSetByCobraAnnotation]) > 0:
		spellings = append(spellings, flagSpellings(c, "version")...)
	}
	return spellings
}

// valueFlagSpellings returns the spellings of c's flags, its own and those
// it inherits, that take the next word as their value.
func valueFlagSpellings(c *cobra.Command) []string {
	var spellings []string
	add := func(f *pflag.Flag) {
		if f.NoOptDefVal == "" {
			spellings = append(spellings, "--"+f.Name)
			if f.Shorthand != "" {
				spellings = append(spellings, "-"+f.Shorthand)
			}
		}
	}
	c.LocalFlags().VisitAll(add)
	c.InheritedFlags().VisitAll(add)
	return spellings
}

// flagSpellings returns the ways to write c's flag name: --name and, when it
// has one, its -shorthand.
func flagSpellings(c *cobra.Command, name string) []string {
	spellings := []string{"--" + name}
	if f := c.Flags().Lookup(name); f != nil && f.Shorthand != "" {
		spellings = append(spellings, "-"+f.Shorthand)
	}
	return spellings
}

// InstallHumanGate makes root refuse every sensitive invocation in its tree
// that a human does not run (RequireHuman), before the command runs. It
// chains any PersistentPreRunE root already has, and only fires for commands
// without their own PersistentPreRunE (cobra runs the nearest one).
func InstallHumanGate(root *cobra.Command) {
	next := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if IsSensitiveInvocation(cmd, args) {
			if err := RequireHuman(cmd); err != nil {
				return err
			}
		}
		if next != nil {
			return next(cmd, args)
		}
		return nil
	}
}

// agentSessionMarker returns the first agent marker set in the environment,
// or "" when the command is not running inside an agent session.
func agentSessionMarker() string {
	for _, name := range canon.AgentEnvMarkers {
		if os.Getenv(name) != "" {
			return name
		}
	}
	return ""
}

// Terminal is an input a test can present as an interactive terminal.
type Terminal interface {
	io.Reader
	IsTerminal() bool
}

// atTerminal reports whether in is an interactive terminal.
func atTerminal(in io.Reader) bool {
	switch r := in.(type) {
	case *os.File:
		return term.IsTerminal(r.Fd())
	case Terminal:
		return r.IsTerminal()
	}
	return false
}

// RequireHuman refuses cmd unless a human runs it: an agent's tool calls run
// inside its session (and usually without a terminal) and cannot answer a
// confirmation prompt. A pseudo-terminal wrapper such as script(1) defeats the
// terminal check alone, hence the agent-environment check too. It is defence
// in depth: the enforcing control for Claude Code is the self-protection
// hook's SP-014, which blocks the same commands before they run.
func RequireHuman(cmd *cobra.Command) error {
	app := branding.Get().AppName
	name := strings.TrimSpace(strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()))
	if marker := agentSessionMarker(); marker != "" {
		return fmt.Errorf("'%s %s' requires a human: it is refused inside an AI agent session (%s is set); run it from your own terminal",
			app, name, marker)
	}
	if !atTerminal(cmd.InOrStdin()) {
		return fmt.Errorf("'%s %s' requires a human at an interactive terminal: it weakens or removes a guardrail", app, name)
	}
	return nil
}
