package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func makePolicy(rules ...PolicyRule) *SecurityPolicy {
	return &SecurityPolicy{
		APIVersion: "qsdev/v1",
		Kind:       "SecurityPolicy",
		Metadata:   PolicyMetadata{Name: "test"},
		Rules:      rules,
	}
}

func makeRule(id string, tier BypassTier, sev Severity, condType ConditionType, actionType ActionType) PolicyRule {
	return PolicyRule{
		ID:         id,
		Category:   "test",
		Name:       id,
		Severity:   sev,
		BypassTier: tier,
		Conditions: Condition{Type: condType, ToolName: "Bash"},
		Action:     Action{Type: actionType, Message: "rule " + id + " fired"},
	}
}

func compileTestPolicy(t *testing.T, sp *SecurityPolicy) *CompiledPolicySet {
	t.Helper()
	set, err := Compile(sp)
	if err != nil {
		t.Fatalf("Compile returned unexpected error: %v", err)
	}
	return set
}

// panicCondition is a test double that panics when evaluated, used to test
// fail-closed behavior and short-circuit logic.
type panicCondition struct{}

func (c *panicCondition) Evaluate(_ *EvalContext) (bool, error) {
	panic("test panic")
}

// errorCondition is a test double that returns an error when evaluated.
type errorCondition struct{}

func (c *errorCondition) Evaluate(_ *EvalContext) (bool, error) {
	return false, fmt.Errorf("test error")
}

// alwaysTrueCondition is a test double that always matches.
type alwaysTrueCondition struct{}

func (c *alwaysTrueCondition) Evaluate(_ *EvalContext) (bool, error) {
	return true, nil
}

func TestEvaluate_EnforceAlwaysBlocks(t *testing.T) {
	t.Parallel()

	rule := makeRule("EA-001", EnforceAlways, Critical, ToolMatch, Block)
	sp := makePolicy(rule)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{
		ToolName:         "Bash",
		SessionOverrides: []string{"EA-001"}, // override should have no effect
	}

	decision := Evaluate(set, ctx)
	if decision.Action != Block {
		t.Errorf("expected Block action, got %q", decision.Action)
	}
	if decision.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", decision.ExitCode)
	}
	if decision.RuleID != "EA-001" {
		t.Errorf("expected rule ID %q, got %q", "EA-001", decision.RuleID)
	}
}

func TestEvaluate_SessionBypass(t *testing.T) {
	t.Parallel()

	rule := makeRule("SESS-001", Session, High, ToolMatch, Block)
	sp := makePolicy(rule)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{
		ToolName:         "Bash",
		SessionOverrides: []string{"SESS-001"},
	}

	decision := Evaluate(set, ctx)
	if decision.Action == Block {
		t.Error("session-tier rule should be bypassed with matching override, but got Block")
	}
	if decision.ExitCode != 0 {
		t.Errorf("expected exit code 0 when bypassed, got %d", decision.ExitCode)
	}
}

func TestEvaluate_CommandBypass(t *testing.T) {
	t.Parallel()

	rule := makeRule("CMD-001", Command, Low, ToolMatch, Block)
	sp := makePolicy(rule)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{
		ToolName:         "Bash",
		SessionOverrides: []string{"CMD-001"},
	}

	decision := Evaluate(set, ctx)
	if decision.Action == Block {
		t.Error("command-tier rule should be bypassed with matching override, but got Block")
	}
	if decision.ExitCode != 0 {
		t.Errorf("expected exit code 0 when bypassed, got %d", decision.ExitCode)
	}
}

func TestEvaluate_MonitorModeNoBlock(t *testing.T) {
	t.Parallel()

	rule := makeRule("MON-001", Session, Medium, ToolMatch, Block)
	rule.MonitorMode = true
	sp := makePolicy(rule)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{
		ToolName: "Bash",
	}

	decision := Evaluate(set, ctx)
	if decision.Action == Block {
		t.Error("monitor_mode block rule should NOT produce a Block decision")
	}
	if decision.ExitCode != 0 {
		t.Errorf("expected exit code 0 for monitor_mode, got %d", decision.ExitCode)
	}
	if len(decision.Findings) == 0 {
		t.Error("expected at least one finding for monitor_mode rule")
	}

	found := false
	for _, f := range decision.Findings {
		if f.RuleID == "MON-001" && f.Monitor {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding with RuleID MON-001 and Monitor=true")
	}
}

func TestEvaluate_BlockShortCircuits(t *testing.T) {
	t.Parallel()

	// Build a compiled set manually: first rule blocks, second panics.
	// If short-circuit works, the panic never fires.
	blockRule := PolicyRule{
		ID: "SHORT-001", Category: "test", Name: "blocker",
		Severity: Critical, BypassTier: EnforceAlways,
		Action: Action{Type: Block, Message: "short-circuit block"},
	}
	panicRule := PolicyRule{
		ID: "SHORT-002", Category: "test", Name: "panicker",
		Severity: Critical, BypassTier: EnforceAlways,
		Action: Action{Type: Block, Message: "should never reach"},
	}

	set := &CompiledPolicySet{
		Rules: []CompiledRule{
			{Rule: blockRule, Condition: &alwaysTrueCondition{}, Action: ResolveAction(Block)},
			{Rule: panicRule, Condition: &panicCondition{}, Action: ResolveAction(Block)},
		},
		ToolIndex: map[string][]*CompiledRule{},
	}
	// Point ToolIndex at our rules.
	set.ToolIndex["Bash"] = []*CompiledRule{&set.Rules[0], &set.Rules[1]}

	ctx := &EvalContext{ToolName: "Bash"}
	decision := Evaluate(set, ctx)

	if decision.Action != Block {
		t.Errorf("expected Block from first rule, got %q", decision.Action)
	}
	if decision.RuleID != "SHORT-001" {
		t.Errorf("expected rule ID SHORT-001, got %q", decision.RuleID)
	}
}

func TestEvaluate_WarnAccumulatesFindings(t *testing.T) {
	t.Parallel()

	sp := makePolicy(
		makeRule("WARN-001", Command, Low, ToolMatch, Warn),
		makeRule("WARN-002", Command, Low, ToolMatch, Warn),
		makeRule("WARN-003", Command, Low, ToolMatch, Warn),
	)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{ToolName: "Bash"}
	decision := Evaluate(set, ctx)

	if decision.Action == Block {
		t.Error("warn rules should not produce Block")
	}
	if decision.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", decision.ExitCode)
	}
	if len(decision.Findings) != 3 {
		t.Errorf("expected 3 findings, got %d", len(decision.Findings))
	}

	seen := make(map[string]bool)
	for _, f := range decision.Findings {
		seen[f.RuleID] = true
	}
	for _, id := range []string{"WARN-001", "WARN-002", "WARN-003"} {
		if !seen[id] {
			t.Errorf("missing finding for rule %s", id)
		}
	}
}

func TestEvaluate_FailClosed_Panic(t *testing.T) {
	t.Parallel()

	rule := PolicyRule{
		ID: "PANIC-001", Category: "test", Name: "panicker",
		Severity: Critical, BypassTier: EnforceAlways,
		Action: Action{Type: Block, Message: "panic test"},
	}

	set := &CompiledPolicySet{
		Rules: []CompiledRule{
			{Rule: rule, Condition: &panicCondition{}, Action: ResolveAction(Block)},
		},
		ToolIndex: map[string][]*CompiledRule{},
	}
	set.ToolIndex["Bash"] = []*CompiledRule{&set.Rules[0]}

	ctx := &EvalContext{ToolName: "Bash"}
	decision := Evaluate(set, ctx)

	if decision.Action != Block {
		t.Errorf("expected Block for panicking condition (fail-closed), got %q", decision.Action)
	}
	if decision.ExitCode != 2 {
		t.Errorf("expected exit code 2 for panic, got %d", decision.ExitCode)
	}
	if decision.Message == "" {
		t.Error("expected non-empty message for panic recovery")
	}
}

func TestEvaluate_FailClosed_Error(t *testing.T) {
	t.Parallel()

	rule := PolicyRule{
		ID: "ERR-001", Category: "test", Name: "error-maker",
		Severity: Critical, BypassTier: EnforceAlways,
		Action: Action{Type: Block, Message: "error test"},
	}

	set := &CompiledPolicySet{
		Rules: []CompiledRule{
			{Rule: rule, Condition: &errorCondition{}, Action: ResolveAction(Block)},
		},
		ToolIndex: map[string][]*CompiledRule{},
	}
	set.ToolIndex["Bash"] = []*CompiledRule{&set.Rules[0]}

	ctx := &EvalContext{ToolName: "Bash"}
	decision := Evaluate(set, ctx)

	if decision.Action != Block {
		t.Errorf("expected Block for erroring condition (fail-closed), got %q", decision.Action)
	}
	if decision.ExitCode != 2 {
		t.Errorf("expected exit code 2 for error, got %d", decision.ExitCode)
	}
	if decision.Err == nil {
		t.Error("expected non-nil Err for condition error")
	}
}

func TestEvaluate_TierFilter_EnforceAlwaysOnly(t *testing.T) {
	t.Parallel()

	sp := makePolicy(
		makeRule("EA-F1", EnforceAlways, Critical, ToolMatch, Block),
		makeRule("SESS-F1", Session, High, ToolMatch, Warn),
		makeRule("CMD-F1", Command, Low, ToolMatch, Warn),
	)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{
		ToolName:   "Bash",
		TierFilter: EnforceAlwaysOnly,
	}
	decision := Evaluate(set, ctx)

	if decision.Action != Block {
		t.Errorf("expected enforce_always rule to fire, got action %q", decision.Action)
	}
	if decision.RuleID != "EA-F1" {
		t.Errorf("expected rule EA-F1, got %q", decision.RuleID)
	}
	// No findings from session/command rules.
	for _, f := range decision.Findings {
		if f.RuleID == "SESS-F1" || f.RuleID == "CMD-F1" {
			t.Errorf("session/command rule %s should have been filtered out", f.RuleID)
		}
	}
}

func TestEvaluate_TierFilter_SessionCommandOnly(t *testing.T) {
	t.Parallel()

	sp := makePolicy(
		makeRule("EA-F2", EnforceAlways, Critical, ToolMatch, Block),
		makeRule("SESS-F2", Session, High, ToolMatch, Warn),
	)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{
		ToolName:   "Bash",
		TierFilter: SessionCommandOnly,
	}
	decision := Evaluate(set, ctx)

	// enforce_always rule should be skipped, session rule should fire.
	if decision.Action == Block {
		t.Error("enforce_always rule should be skipped with SessionCommandOnly filter")
	}
	if len(decision.Findings) == 0 {
		t.Error("expected findings from session-tier warn rule")
	}

	found := false
	for _, f := range decision.Findings {
		if f.RuleID == "SESS-F2" {
			found = true
		}
		if f.RuleID == "EA-F2" {
			t.Error("enforce_always rule EA-F2 should have been filtered out")
		}
	}
	if !found {
		t.Error("expected finding from session rule SESS-F2")
	}
}

func TestEvaluate_NoMatchAllowed(t *testing.T) {
	t.Parallel()

	sp := makePolicy(
		makeRule("NO-MATCH", EnforceAlways, Critical, ToolMatch, Block),
	)
	set := compileTestPolicy(t, sp)

	// Use a tool name that does not match the rule's ToolMatch("Bash").
	ctx := &EvalContext{ToolName: "Read"}
	decision := Evaluate(set, ctx)

	if decision.Action == Block {
		t.Error("expected no block when no rules match")
	}
	if decision.ExitCode != 0 {
		t.Errorf("expected exit code 0 when no rules match, got %d", decision.ExitCode)
	}
	if len(decision.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(decision.Findings))
	}
}

func TestEvaluate_EvaluationOrder(t *testing.T) {
	t.Parallel()

	// enforce_always/critical should evaluate before session/low.
	// Both target the same tool. The enforce_always block should win.
	sp := makePolicy(
		makeRule("ORDER-LO", Session, Low, ToolMatch, Warn),
		makeRule("ORDER-HI", EnforceAlways, Critical, ToolMatch, Block),
	)
	set := compileTestPolicy(t, sp)

	ctx := &EvalContext{ToolName: "Bash"}
	decision := Evaluate(set, ctx)

	if decision.Action != Block {
		t.Errorf("expected Block from enforce_always/critical rule, got %q", decision.Action)
	}
	if decision.RuleID != "ORDER-HI" {
		t.Errorf("expected first firing rule to be ORDER-HI, got %q", decision.RuleID)
	}
}

func BenchmarkEvaluate8Rules(b *testing.B) {
	tools := []string{"Bash", "Edit", "Read", "Write", "Grep", "Glob", "LS", "MCP"}
	var rules []PolicyRule
	for i, tool := range tools {
		rules = append(rules, PolicyRule{
			ID:         fmt.Sprintf("BENCH-%03d", i),
			Category:   "bench",
			Name:       fmt.Sprintf("bench rule %d", i),
			Severity:   Low,
			BypassTier: Command,
			Conditions: Condition{Type: ToolMatch, ToolName: tool},
			Action:     Action{Type: Warn, Message: "bench warning"},
		})
	}

	sp := makePolicy(rules...)
	set, err := Compile(sp)
	if err != nil {
		b.Fatalf("Compile: %v", err)
	}

	ctx := &EvalContext{ToolName: "Bash"}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		Evaluate(set, ctx)
	}
}

func TestPolicyEngine_EndToEnd(t *testing.T) {
	t.Parallel()

	engine, err := NewPolicyEngine(
		[]string{"testdata/valid-policy.yaml"},
		nil,
		EngineOptions{},
	)
	if err != nil {
		t.Fatalf("NewPolicyEngine: %v", err)
	}

	// SP-001 blocks Edit on .claude/settings.json.
	ctx := &EvalContext{
		ToolName:  "Edit",
		ToolInput: json.RawMessage(`{"file_path": "/project/.claude/settings.json"}`),
		FilePath:  "/project/.claude/settings.json",
	}

	decision := engine.Evaluate(ctx)
	if decision.Action != Block {
		t.Errorf("expected Block from SP-001, got %q", decision.Action)
	}
	if decision.RuleID != "SP-001" {
		t.Errorf("expected rule ID SP-001, got %q", decision.RuleID)
	}
}

func TestPolicyEngine_WithSessionBypass(t *testing.T) {
	t.Parallel()

	state := &StaticSessionStateReader{
		Overrides: []string{"CG-001"},
	}

	engine, err := NewPolicyEngine(
		[]string{"testdata/valid-policy.yaml"},
		state,
		EngineOptions{},
	)
	if err != nil {
		t.Fatalf("NewPolicyEngine: %v", err)
	}

	// CG-001 blocks access to .ssh paths, but we have a session bypass.
	ctx := &EvalContext{
		ToolName: "Read",
		FilePath: "/home/user/.ssh/id_rsa",
	}

	decision := engine.Evaluate(ctx)
	if decision.Action == Block && decision.RuleID == "CG-001" {
		t.Error("CG-001 should be bypassed with session override, but got Block")
	}
}

func TestSessionState_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	overrides := []string{"RULE-A", "RULE-B", "RULE-C"}

	if err := SaveSessionOverrides(path, overrides); err != nil {
		t.Fatalf("SaveSessionOverrides: %v", err)
	}

	reader := NewFileSessionStateReader(path)
	got := reader.SessionOverrides()

	if len(got) != len(overrides) {
		t.Fatalf("round-trip length: got %d, want %d", len(got), len(overrides))
	}
	for i := range overrides {
		if got[i] != overrides[i] {
			t.Errorf("round-trip index %d: got %q, want %q", i, got[i], overrides[i])
		}
	}

	if err := ClearSessionOverrides(path); err != nil {
		t.Fatalf("ClearSessionOverrides: %v", err)
	}

	postClear := reader.SessionOverrides()
	if len(postClear) != 0 {
		t.Errorf("expected empty overrides after clear, got %v", postClear)
	}
}

// TestSaveSessionOverrides_AtomicReplace covers F199: the state file is
// replaced atomically (parent directory created, no temp files left behind)
// and an overwrite fully replaces the previous overrides.
func TestSaveSessionOverrides_AtomicReplace(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", ".qsdev")
	path := filepath.Join(dir, "session-state.json")

	if err := SaveSessionOverrides(path, []string{"RULE-A", "RULE-B"}); err != nil {
		t.Fatalf("SaveSessionOverrides: %v", err)
	}
	if err := SaveSessionOverrides(path, []string{"RULE-C"}); err != nil {
		t.Fatalf("SaveSessionOverrides overwrite: %v", err)
	}

	if got := NewFileSessionStateReader(path).SessionOverrides(); len(got) != 1 || got[0] != "RULE-C" {
		t.Errorf("overrides after overwrite = %v, want [RULE-C]", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading state dir: %v", err)
	}
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("state dir holds %v, want only the state file", names)
	}
}

// TestEvaluate_MonitorModeReportsOnlyMatches pins that a monitor-mode block
// rule reports a finding only when its conditions match the call, like any
// other rule; it must not report a violation for every call to the tool.
func TestEvaluate_MonitorModeReportsOnlyMatches(t *testing.T) {
	t.Parallel()

	rule := makeRule("MON-002", Session, Medium, All, Block)
	rule.Conditions = Condition{Type: All, Conditions: []Condition{
		{Type: ToolMatch, ToolName: "Bash"},
		{Type: CommandMatch, Pattern: "curl"},
	}}
	rule.MonitorMode = true
	set := compileTestPolicy(t, makePolicy(rule))

	tests := []struct {
		name        string
		command     string
		wantFinding bool
	}{
		{name: "matching call reported", command: "curl https://example.com", wantFinding: true},
		{name: "non-matching call not reported", command: "ls -la", wantFinding: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decision := Evaluate(set, &EvalContext{ToolName: "Bash", Command: tt.command})
			if decision.ExitCode != 0 {
				t.Fatalf("monitor-mode rule blocked: exit %d", decision.ExitCode)
			}
			if got := len(decision.Findings) > 0; got != tt.wantFinding {
				t.Errorf("findings = %+v, want finding: %v", decision.Findings, tt.wantFinding)
			}
		})
	}
}

// TestEvaluate_PromptAllowDoesNotShadowBlock pins that a prompt rule resolving
// to allow does not end evaluation: a later-sorted block rule that matches the
// same call still blocks. Otherwise any rule (for example one an overlay adds)
// with a more severe or more tool-specific allow-by-default prompt would
// neutralize an enforce_always block rule.
func TestEvaluate_PromptAllowDoesNotShadowBlock(t *testing.T) {
	t.Parallel()

	block := makeRule("BLOCK-001", EnforceAlways, High, CommandMatch, Block)
	block.Conditions = Condition{Type: CommandMatch, Pattern: "curl"}

	tests := []struct {
		name   string
		prompt PolicyRule
	}{
		{name: "more severe prompt", prompt: makeRule("PROMPT-SEV", EnforceAlways, Critical, ToolMatch, Prompt)},
		{name: "tool-specific prompt of equal severity", prompt: makeRule("PROMPT-TOOL", EnforceAlways, High, ToolMatch, Prompt)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.prompt.Action.DefaultOnTimeout = "allow"
			set := compileTestPolicy(t, makePolicy(block, tt.prompt))

			blocked := Evaluate(set, &EvalContext{ToolName: "Bash", Command: "curl https://example.com"})
			if blocked.ExitCode != 2 || blocked.RuleID != "BLOCK-001" {
				t.Errorf("decision = %s/%d by %q, want block by BLOCK-001", blocked.Action, blocked.ExitCode, blocked.RuleID)
			}

			allowed := Evaluate(set, &EvalContext{ToolName: "Bash", Command: "ls"})
			if allowed.ExitCode != 0 || allowed.Action != Prompt || allowed.RuleID != tt.prompt.ID {
				t.Errorf("decision = %s/%d by %q, want allowed prompt by %s", allowed.Action, allowed.ExitCode, allowed.RuleID, tt.prompt.ID)
			}
		})
	}
}
