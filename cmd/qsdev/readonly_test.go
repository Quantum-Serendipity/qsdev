package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// readOnlyAcceptance is the set of invocations the read-only contract
// promises (XD-WS9). Every one must be declared through cmdutil.MarkReadOnly,
// so dropping an annotation fails the test instead of shrinking it.
var readOnlyAcceptance = []string{
	"status", "check", "list", "info", "outdated",
	"devenv doctor",
	"mcp status", "mcp health", "mcp grade", "mcp list",
	"config show",
	"teardown --dry-run", "update --dry-run", "init --dry-run", "repair --dry-run",
}

// readOnlyLostBy is the opt-out flags the acceptance set records: each takes
// its command out of the read-only contract, so a consumer of the annotation
// must see it.
var readOnlyLostBy = map[string][]string{
	"mcp status": {"probe", "probe-untrusted"},
	"mcp health": {"probe", "probe-untrusted"},
	"outdated":   {"online"},
	"check":      {"auto-fix", "scan"},
	"status":     {"all-badges", "scan"},
	"update":     {"check"},
}

// readOnlyHarmlessFlags is, per read-only command, the boolean flags reviewed
// as keeping its read-only invocation inside the contract: they only shape
// output, select what is previewed, or are answers a dry-run never applies.
// Every other boolean flag must be recorded as an opt-out, so a new mutating
// or network flag cannot join a read-only command unreviewed.
var readOnlyHarmlessFlags = map[string][]string{
	"devenv doctor": {"check", "json"},
	"info":          {"json", "oneline"},
	"init": {
		"agent-postmortem", "agent-semble", "agent-semble-text-files", "agent-version-sentinel",
		"claude-code", "claude-only", "devenv-only", "direnv", "force", "list-profiles",
		"merge", "nix-hardening-guide", "quiet", "update", "yes",
	},
	"mcp grade":  {"all", "json"},
	"mcp health": {"json"},
	"mcp list":   {"json"},
	"mcp status": {"json"},
	"repair":     {"force", "reset"},
	"status":     {"fix", "json", "quiet", "sarif", "verbose"},
	"teardown":   {"archive", "compliance", "force", "quick"},
	"update": {
		"allow-downgrade", "changelog", "configs-only", "deps-only", "force",
		"no-strict", "overwrite-modified", "self-only", "skip-container",
	},
}

// TestReadOnlyAnnotationFlags: every flag a read-only annotation names (the
// flag that makes the command read-only and the flags that leave the
// contract) exists on its command, the opt-outs are the recorded ones, and
// every other boolean flag of a read-only command is reviewed as harmless.
func TestReadOnlyAnnotationFlags(t *testing.T) {
	t.Parallel()
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if args, ok := cmdutil.ReadOnlyArgs(c); ok {
			inv := strings.TrimPrefix(c.CommandPath(), c.Root().Name()+" ")
			lostBy := cmdutil.ReadOnlyLostBy(c)
			flags := slices.Clone(lostBy)
			via := c.Annotations[cmdutil.ReadOnlyAnnotation]
			if via != "" {
				flags = append(flags, via)
			}
			for _, f := range flags {
				if c.Flags().Lookup(f) == nil {
					t.Errorf("%s (read-only as %q): annotation names --%s, which it does not define", inv, args, f)
				}
			}
			if want := readOnlyLostBy[inv]; !slices.Equal(lostBy, want) {
				t.Errorf("%s: read-only opt-out flags = %q, want %q", inv, lostBy, want)
			}
			c.LocalFlags().VisitAll(func(f *pflag.Flag) {
				if f.Value.Type() != "bool" || f.Name == "help" || f.Name == via ||
					slices.Contains(lostBy, f.Name) || slices.Contains(readOnlyHarmlessFlags[inv], f.Name) {
					return
				}
				t.Errorf("%s: boolean flag --%s is neither a read-only opt-out (cmdutil.MarkReadOnly lostBy) nor reviewed as harmless (readOnlyHarmlessFlags)", inv, f.Name)
			})
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(instance.NewRootCommand())
}

// TestReadonlyCommandsNeverExec walks qsdev's real command tree and runs every
// invocation declared read-only against a freshly initialized project with
// procexec's forbid-exec guard on. Any exec that is not a declared local probe
// panics in procexec; the panic is recovered and reported with its argv.
func TestReadonlyCommandsNeverExec(t *testing.T) {
	invocations := readOnlyInvocations(instance.NewRootCommand())
	for _, want := range readOnlyAcceptance {
		if !slices.Contains(invocations, want) {
			t.Errorf("%q is not annotated read-only (cmdutil.MarkReadOnly); annotated: %q", want, invocations)
		}
	}

	dir := initFixture(t)
	t.Setenv(procexec.ForbidExecEnv, "1")
	for _, inv := range invocations {
		t.Run(inv, func(t *testing.T) {
			res := executeRecovering(strings.Fields(inv))
			if res.panicked != nil {
				t.Fatalf("qsdev %s ran a forbidden exec: %v\n%s", inv, res.panicked, res.out)
			}
			// A read-only command may legitimately fail on this fixture (a
			// finding, an offline refusal); a usage error means the
			// annotation names an invocation that does not exist.
			if err := res.err; err != nil {
				if msg := err.Error(); strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown command") {
					t.Fatalf("qsdev %s: %v", inv, err)
				}
				t.Logf("qsdev %s exited with error: %v", inv, err)
			}
			if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr != nil {
				t.Fatalf("fixture lost its git repository after qsdev %s", inv)
			}
		})
	}
}

// readOnlyInvocations returns, for every command under root marked read-only,
// the space-joined arguments that invoke its read-only form.
func readOnlyInvocations(root *cobra.Command) []string {
	var out []string
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if args, ok := cmdutil.ReadOnlyArgs(c); ok {
			out = append(out, strings.Join(args, " "))
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return out
}

// initFixture creates a Go project with its own git repository under fresh
// HOME and TMPDIR directories, runs `init --yes` in it in-process and leaves
// the test in that directory.
func initFixture(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/ro\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	t.Chdir(dir)
	if res := executeRecovering([]string{"init", "--yes", "--lang", "go"}); res.err != nil || res.panicked != nil {
		t.Fatalf("init --yes: err=%v panic=%v\n%s", res.err, res.panicked, res.out)
	}
	return dir
}

// runResult is the outcome of one in-process qsdev invocation.
type runResult struct {
	out      string
	err      error
	panicked any
}

// executeRecovering runs qsdev with args on a fresh command tree, recovering
// a panic so one forbidden exec is reported instead of aborting the run.
func executeRecovering(args []string) (res runResult) {
	root := instance.NewRootCommand()
	var buf bytes.Buffer
	root.SetIn(strings.NewReader(""))
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	defer func() {
		if r := recover(); r != nil {
			res = runResult{out: buf.String(), panicked: fmt.Sprint(r)}
		}
	}()
	err := root.Execute()
	return runResult{out: buf.String(), err: err}
}
