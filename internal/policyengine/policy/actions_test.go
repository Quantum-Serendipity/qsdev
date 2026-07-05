package policy

import "testing"

// TestPromptAction_FailsClosed is the regression guard for F-CAP-21.1-2: the
// `prompt` action used to be inverted — on a TTY it returned Prompt/exit 0
// (ALLOW), letting the gated call through. A prompt verdict must never resolve
// to a silent allow: it is fail-closed unless the policy explicitly opts into
// an allow default via default_on_timeout.
func TestPromptAction_FailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		defaultOnTimeout string
		wantAction       ActionType
		wantExit         int
	}{
		{name: "unset default blocks", defaultOnTimeout: "", wantAction: Block, wantExit: 2},
		{name: "deny default blocks", defaultOnTimeout: "deny", wantAction: Block, wantExit: 2},
		{name: "block default blocks", defaultOnTimeout: "block", wantAction: Block, wantExit: 2},
		{name: "explicit allow proceeds", defaultOnTimeout: "allow", wantAction: Prompt, wantExit: 0},
		{name: "explicit proceed proceeds", defaultOnTimeout: "proceed", wantAction: Prompt, wantExit: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule := &PolicyRule{
				ID:     "PROMPT-001",
				Action: Action{Type: Prompt, Message: "confirm", DefaultOnTimeout: tt.defaultOnTimeout},
			}
			decision := promptAction{}.Execute(rule, &EvalContext{})

			if decision.Action != tt.wantAction {
				t.Errorf("Action = %q, want %q", decision.Action, tt.wantAction)
			}
			if decision.ExitCode != tt.wantExit {
				t.Errorf("ExitCode = %d, want %d", decision.ExitCode, tt.wantExit)
			}
			// A prompt verdict must never both look like an allow (exit 0) while
			// leaving the caller believing it was gated.
			if tt.wantExit == 2 && decision.Action == Prompt {
				t.Error("prompt verdict resolved to Prompt with a blocking exit code")
			}
		})
	}
}

// TestEvaluate_PromptDoesNotAllow proves the fix through the evaluator: a rule
// whose action is `prompt` (with the default, non-interactive resolution) does
// NOT allow the call — it produces a blocking decision.
func TestEvaluate_PromptDoesNotAllow(t *testing.T) {
	t.Parallel()

	rule := makeRule("PROMPT-EVAL", EnforceAlways, High, ToolMatch, Prompt)
	set := compileTestPolicy(t, makePolicy(rule))

	decision := Evaluate(set, &EvalContext{ToolName: "Bash"})

	if decision.ExitCode != 2 {
		t.Errorf("prompt verdict should be fail-closed (exit 2), got exit %d (action %q)",
			decision.ExitCode, decision.Action)
	}
	if decision.Action == Prompt && decision.ExitCode == 0 {
		t.Error("prompt verdict resolved to a silent allow in the non-interactive path")
	}
}
