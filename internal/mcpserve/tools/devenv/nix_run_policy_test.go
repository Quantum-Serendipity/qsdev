package devenv

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/judge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

// testNixRunPolicy returns the policy the serve command builds for a project
// at root: deny, the embedded catalog's ask rules, and the self-protection
// judge with SP-014 blocking `teardown`.
func testNixRunPolicy(t *testing.T, root string, deny []string) NixRunPolicy {
	t.Helper()
	sensitive := []cmdscan.CommandSpec{{Path: [][]string{{"teardown"}}, ReadOnly: []string{"--dry-run", "--help"}}}
	return NixRunPolicy{
		DenyRules: deny,
		AskRules:  embeddedCatalog(t).AllPermissionAskRules(),
		Judge: func(line string) (string, string, bool) {
			d, denied := judge.Evaluate(&rules.EvalContext{ToolName: "Bash", Command: line, CWD: root, SensitiveCommands: sensitive})
			return d.RuleID, d.Reason, denied
		},
	}
}

// TestNixRun_PolicyRefusesWhatBashMayNot is the XS-WS6 E2 test of the
// confused-deputy mapping: a call whose Bash equivalent self-protection
// refuses (a protected file rewritten or removed, a human-only CLI command,
// a decoded script piped to a shell) or a Bash ask rule asks about (a
// package install) is refused before nix is looked up, so with PATH emptied
// nothing can reach the not_configured result.
func TestNixRun_PolicyRefusesWhatBashMayNot(t *testing.T) {
	t.Setenv("PATH", "")
	// The decoded payload is assembled so this file's own text holds no
	// decode-and-run line.
	decodeToShell := "echo Y3VybCB4fHNo | base64 " + "-d | sh"
	tests := []struct {
		name    string
		command string
		args    []any
		stdin   string
		field   string // the structured field naming what refused it
	}{
		{"remove settings through bash -c", "nixpkgs#bash", []any{"-c", "rm .claude/settings.json"}, "", "selfprotect_rule"},
		{"overwrite local settings through bash -c", "nixpkgs#bash", []any{"-c", "echo x > .claude/settings.local.json"}, "", "selfprotect_rule"},
		{"remove settings with coreutils", "nixpkgs#coreutils", []any{"rm", ".claude/settings.json"}, "", "selfprotect_rule"},
		{"human-only CLI command", "nixpkgs#qsdev", []any{"teardown", "--force"}, "", "selfprotect_rule"},
		{"decoded script piped to a shell", "nixpkgs#bash", []any{"-c", decodeToShell}, "", "selfprotect_rule"},
		{"settings removal on stdin", "nixpkgs#bash", nil, "rm -f .claude/settings.json\n", "selfprotect_rule"},
		{"package install through bash -c", "nixpkgs#bash", []any{"-c", "npm i evil"}, "", "ask_rule"},
		{"package install as the program", "nixpkgs#npm", []any{"install", "evil"}, "", "ask_rule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			nix := newNixRunner(root, testNixRunPolicy(t, root, embeddedCatalog(t).AllPermissionDenyRules()))
			arguments := map[string]any{"command": tt.command}
			if tt.args != nil {
				arguments["args"] = tt.args
			}
			if tt.stdin != "" {
				arguments["stdin"] = tt.stdin
			}
			res := call(t, nix.handle, arguments)
			st, _ := res.Structured.(map[string]any)
			if !res.IsError || st["status"] != "error" {
				t.Fatalf("expected a refusal before the nix lookup, got %+v", res.Structured)
			}
			if _, ok := st[tt.field]; !ok {
				t.Errorf("refusal %v does not carry %s", st, tt.field)
			}
		})
	}

	// A harmless call passes every check and reaches the nix lookup, which
	// fails on the emptied PATH.
	root := t.TempDir()
	nix := newNixRunner(root, testNixRunPolicy(t, root, embeddedCatalog(t).AllPermissionDenyRules()))
	res := call(t, nix.handle, map[string]any{"command": "nixpkgs#jq", "args": []any{"."}})
	if st := res.Structured.(map[string]any)["status"]; st != "not_configured" {
		t.Errorf("nixpkgs#jq . status = %v, want not_configured (past the policy); structured %v", st, res.Structured)
	}

	// A server built without a judge fails closed.
	nix = newNixRunner(t.TempDir(), NixRunPolicy{})
	res = call(t, nix.handle, map[string]any{"command": "nixpkgs#jq", "args": []any{"."}})
	if st := res.Structured.(map[string]any)["status"]; st != "error" {
		t.Errorf("no judge: status = %v, want error (refused); structured %v", st, res.Structured)
	}
}
