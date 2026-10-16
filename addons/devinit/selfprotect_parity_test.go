package devinit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/defaults"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// minParityCommands is the floor on the commands harvested from the rule
// fixtures, so a harvest that silently finds nothing cannot pass.
const minParityCommands = 250

// TestSelfprotect_MonitorParity pins that the Monitor tool, which runs its
// tool_input.command in the same shell as Bash, gets the same verdict as Bash
// for every Bash command the rule tests use (XS-WS4 D5): the same exit and
// the same rule ID, from the whole in-process hook (evasion, the Tier-1
// rules, the gate-dodge checks and GIT-001). The commands are harvested from
// internal/selfprotect/rules/*_test.go, so a fixture added there is covered
// here without being listed twice. Not parallel: testutil.Project changes
// the working directory and the per-user environment.
func TestSelfprotect_MonitorParity(t *testing.T) {
	commands := harvestRuleFixtureCommands(t)
	if len(commands) < minParityCommands {
		t.Fatalf("harvested %d commands from the rule fixtures, want at least %d", len(commands), minParityCommands)
	}
	t.Logf("harvested %d commands", len(commands))

	dir := testutil.Project(t, testutil.ProjectOptions{})
	sensitive := realSensitiveCommands()
	for _, command := range commands {
		bashExit, bashRule := selfprotectVerdict(t, sensitive, dir, "Bash", command)
		monExit, monRule := selfprotectVerdict(t, sensitive, dir, "Monitor", command)
		if bashExit != monExit || bashRule != monRule {
			t.Errorf("%q: Bash gives exit %d %s, Monitor gives exit %d %s", command, bashExit, bashRule, monExit, monRule)
		}
	}
}

// realSensitiveCommands returns the sensitive command specs of a command
// tree built from the constructors of the commands devinit and self-update
// register, as the hook derives them from its own root.
func realSensitiveCommands() []cmdscan.CommandSpec {
	root := &cobra.Command{Use: branding.Get().AppName}
	root.AddCommand(
		initCmd(), trialCmd(), scaffoldCmd(), enableCmd(), disableCmd(),
		statusCmd(), listCmd(), checkCmd(), evidenceCmd(), teamReportCmd(),
		repairCmd(), infoCmd(), outdatedCmd(), updateCmd(), teardownCmd(),
		containerCmd(), sandboxCmd(), selfprotectCmd(), enforceCmd(),
		sessionCmd(), policyCmd(), defaults.Command(),
		selfupdate.Command(),
	)
	return cmdutil.SensitiveCommands(root)
}

// reDenyRule extracts the rule ID (or EVASION-category) from a deny line on
// the hook's stderr.
var reDenyRule = regexp.MustCompile(`qsdev-selfprotect: (\S+)`)

// selfprotectVerdict runs the in-process hook on a tool call of tool with
// command in cwd, returning its exit code (0 allow, 2 deny) and the rule ID
// that denied it.
func selfprotectVerdict(t *testing.T, sensitive []cmdscan.CommandSpec, cwd, tool, command string) (int, string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"cwd":             cwd,
		"tool_name":       tool,
		"tool_input":      map[string]string{"command": command},
	})
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	err = evaluateSelfprotect(context.Background(), bytes.NewReader(payload), &stderr, sensitive)
	if err == nil {
		return 0, ""
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("%s %q: unexpected error %v", tool, command, err)
	}
	rule := ""
	if m := reDenyRule.FindStringSubmatch(stderr.String()); m != nil {
		rule = m[1]
	}
	return exitErr.Code, rule
}

// harvestRuleFixtureCommands returns, deduplicated and in file order, every
// string literal the rule tests give as a command: the value of a `command`
// or `Command` key in a composite literal (EvalContext, table rows), and the
// command column of an unkeyed row of a table whose struct has such a field.
// Literals of a composite literal naming another tool (ToolName: "Write")
// are left out: they are not shell commands.
func harvestRuleFixtureCommands(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "internal", "selfprotect", "rules", "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("found no rule test files")
	}
	h := &fixtureHarvest{seen: map[string]bool{}}
	fset := token.NewFileSet()
	for _, p := range files {
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, h.visit)
	}
	return h.commands
}

// fixtureHarvest collects commands for harvestRuleFixtureCommands.
type fixtureHarvest struct {
	seen     map[string]bool
	commands []string
}

// add records e when it is a non-blank string literal not seen before.
func (h *fixtureHarvest) add(e ast.Expr) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil || strings.TrimSpace(s) == "" || h.seen[s] {
		return
	}
	h.seen[s] = true
	h.commands = append(h.commands, s)
}

// visit harvests the command literals of one composite literal.
func (h *fixtureHarvest) visit(n ast.Node) bool {
	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return true
	}
	if col := commandColumn(lit.Type); col >= 0 {
		for _, row := range lit.Elts {
			if r, ok := row.(*ast.CompositeLit); ok && len(r.Elts) > col {
				if _, keyed := r.Elts[col].(*ast.KeyValueExpr); !keyed {
					h.add(r.Elts[col])
				}
			}
		}
	}
	var command ast.Expr
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		switch keyName(kv.Key) {
		case "command", "Command":
			command = kv.Value
		case "ToolName":
			if b, ok := kv.Value.(*ast.BasicLit); ok && b.Value != `"Bash"` {
				return true
			}
		}
	}
	if command != nil {
		h.add(command)
	}
	return true
}

// commandColumn returns the index of the string command field of a
// []struct{...} table type, or -1.
func commandColumn(typ ast.Expr) int {
	arr, ok := typ.(*ast.ArrayType)
	if !ok {
		return -1
	}
	st, ok := arr.Elt.(*ast.StructType)
	if !ok {
		return -1
	}
	col := 0
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			col++
			continue
		}
		for _, name := range field.Names {
			if name.Name == "command" || name.Name == "Command" {
				if id, ok := field.Type.(*ast.Ident); ok && id.Name == "string" {
					return col
				}
				return -1
			}
			col++
		}
	}
	return -1
}

// keyName returns the identifier of a composite literal key, or "".
func keyName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
