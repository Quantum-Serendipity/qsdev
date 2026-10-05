package instance

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
)

// gateTestTree returns a command tree with one command of every kind the
// catalog gate distinguishes, and each command's arguments from the root.
func gateTestTree() (*cobra.Command, map[string][]string) {
	run := func(*cobra.Command, []string) error { return nil }
	root := &cobra.Command{Use: "qsdev", RunE: run, SilenceUsage: true, SilenceErrors: true}
	status := &cobra.Command{Use: "status", RunE: run}
	hook := cmdutil.MarkProfile(&cobra.Command{Use: "selfprotect", RunE: run}, cmdutil.ProfileAutomatedHook)
	mcp := cmdutil.MarkProfile(&cobra.Command{Use: "mcp", RunE: run}, cmdutil.ProfileMCPServer)
	version := cmdutil.MarkProfile(&cobra.Command{Use: "version", RunE: run}, cmdutil.ProfileGlobal)
	logs := cmdutil.MarkProfile(&cobra.Command{Use: "logs", RunE: run}, cmdutil.ProfileUnlogged)
	defaults := cmdutil.MarkCatalogOptional(&cobra.Command{Use: "defaults", RunE: run})
	validate := &cobra.Command{Use: "validate", RunE: run}
	defaults.AddCommand(validate)
	root.AddCommand(status, hook, mcp, version, logs, defaults)
	return root, map[string][]string{
		"interactive":            {"status"},
		"automated hook":         {"selfprotect"},
		"mcp server":             {"mcp"},
		"global":                 {"version"},
		"unlogged":               {"logs"},
		"catalog-optional child": {"defaults", "validate"},
		"bare root":              {},
		"catalog-optional group": {"defaults"},
	}
}

// TestCatalogGate pins the root catalog gate: with a catalog that fails to
// load, a command a person runs on a project fails with an error naming the
// repair command, while hooks, the MCP server, global and unlogged commands,
// catalog-optional commands and the bare root still run. With a catalog that
// loads, every command runs.
func TestCatalogGate(t *testing.T) {
	t.Parallel()
	loadErr := errors.New("project defaults may only add or tighten")
	tests := []struct {
		name    string
		load    error
		wantErr bool
	}{
		{"interactive", loadErr, true},
		{"automated hook", loadErr, false},
		{"mcp server", loadErr, false},
		{"global", loadErr, false},
		{"unlogged", loadErr, false},
		{"catalog-optional child", loadErr, false},
		{"catalog-optional group", loadErr, false},
		{"bare root", loadErr, false},
		{"interactive", nil, false},
		{"automated hook", nil, false},
		{"mcp server", nil, false},
		{"global", nil, false},
		{"unlogged", nil, false},
		{"catalog-optional child", nil, false},
		{"bare root", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/fails="+boolName(tt.load != nil), func(t *testing.T) {
			t.Parallel()
			root, args := gateTestTree()
			installCatalogGate(root, func() error { return tt.load })
			err := runRootHook(t, root, args[tt.name])
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("qsdev %v: unexpected error %v", args[tt.name], err)
				}
				return
			}
			if !errors.Is(err, loadErr) {
				t.Fatalf("qsdev %v: error %v does not wrap the load error", args[tt.name], err)
			}
			if !strings.Contains(err.Error(), "defaults validate") {
				t.Errorf("qsdev %v: error %q does not name 'defaults validate'", args[tt.name], err)
			}
		})
	}
}

// TestCatalogGate_ChainsExistingHook: the gate keeps the root's existing
// PersistentPreRunE and runs it once the catalog loads.
func TestCatalogGate_ChainsExistingHook(t *testing.T) {
	t.Parallel()
	root, args := gateTestTree()
	var ran int
	root.PersistentPreRunE = func(*cobra.Command, []string) error { ran++; return nil }
	installCatalogGate(root, func() error { return nil })
	if err := runRootHook(t, root, args["interactive"]); err != nil {
		t.Fatal(err)
	}
	if ran != 1 {
		t.Errorf("existing PersistentPreRunE ran %d times, want 1", ran)
	}
}

// runRootHook runs root's PersistentPreRunE for the command args select, as
// cobra does before running it. The hook is called directly rather than
// through Execute: executing a tree also runs the cobra initializers this
// package's DefaultRuntime installs process-wide, which share state across
// parallel tests.
func runRootHook(t *testing.T, root *cobra.Command, args []string) error {
	t.Helper()
	cmd, rest, err := root.Find(args)
	if err != nil {
		t.Fatalf("finding %q: %v", args, err)
	}
	return root.PersistentPreRunE(cmd, rest)
}

func boolName(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// TestGates_CatalogBeforeSensitivity pins the gate order NewRootCommand
// installs: the catalog gate runs before the human gate, whose sensitivity
// check may read the catalog (disable's security-tool list does). With a
// catalog that fails to load, such a command must fail with the wrapped load
// error rather than reach a catalog lookup that panics.
func TestGates_CatalogBeforeSensitivity(t *testing.T) {
	t.Parallel()
	loadErr := errors.New("project defaults may only add or tighten")
	run := func(*cobra.Command, []string) error { return nil }
	root := &cobra.Command{Use: "qsdev", RunE: run, SilenceUsage: true, SilenceErrors: true}
	var loaded bool
	disable := cmdutil.MarkSensitive(&cobra.Command{Use: "disable", Args: cobra.ExactArgs(1), RunE: run},
		cmdutil.Sensitivity{Args: func() []string {
			if !loaded {
				panic("sensitivity predicate read the catalog before it loaded")
			}
			return []string{"gitleaks"}
		}})
	root.AddCommand(disable)
	installGates(root, func() error { return loadErr })

	err := runRootHook(t, root, []string{"disable", "x"})
	if !errors.Is(err, loadErr) {
		t.Fatalf("qsdev disable x: error %v does not wrap the load error", err)
	}
	if !strings.Contains(err.Error(), "defaults validate") {
		t.Errorf("qsdev disable x: error %q does not name 'defaults validate'", err)
	}
}
