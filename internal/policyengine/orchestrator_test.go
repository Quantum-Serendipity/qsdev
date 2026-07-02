package policyengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/risk"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
)

type mockPolicyEvaluator struct {
	evaluateFunc func(*policy.EvalContext) policy.PolicyDecision
	denyRules    []policy.DenyRule
	rules        []policy.CompiledRule
}

func (m *mockPolicyEvaluator) Evaluate(ctx *policy.EvalContext) policy.PolicyDecision {
	if m.evaluateFunc != nil {
		return m.evaluateFunc(ctx)
	}
	return policy.PolicyDecision{}
}

func (m *mockPolicyEvaluator) FilePathDenyRules() []policy.DenyRule {
	return m.denyRules
}

func (m *mockPolicyEvaluator) CurrentRules() []policy.CompiledRule {
	return m.rules
}

type mockRiskScorer struct {
	scorePackageFunc func(*risk.PackageInfo) risk.PackageScore
	scoreAllFunc     func([]risk.PackageInfo) risk.DependencyHealth
}

func (m *mockRiskScorer) ScorePackage(info *risk.PackageInfo) risk.PackageScore {
	if m.scorePackageFunc != nil {
		return m.scorePackageFunc(info)
	}
	return risk.PackageScore{}
}

func (m *mockRiskScorer) ScoreAll(packages []risk.PackageInfo) risk.DependencyHealth {
	if m.scoreAllFunc != nil {
		return m.scoreAllFunc(packages)
	}
	return risk.DependencyHealth{}
}

type mockTrustEvaluator struct {
	checkAccessFunc    func(string, json.RawMessage, []policy.DenyRule) (bool, string)
	applyHardeningFunc func(string, trust.TrustTier, string) string
	scoreServerFunc    func(*trust.McpServerInfo) trust.TrustScore
}

func (m *mockTrustEvaluator) CheckAccess(toolName string, toolArgs json.RawMessage, denyRules []policy.DenyRule) (bool, string) {
	if m.checkAccessFunc != nil {
		return m.checkAccessFunc(toolName, toolArgs, denyRules)
	}
	return false, ""
}

func (m *mockTrustEvaluator) ApplyHardening(serverName string, tier trust.TrustTier, output string) string {
	if m.applyHardeningFunc != nil {
		return m.applyHardeningFunc(serverName, tier, output)
	}
	return output
}

func (m *mockTrustEvaluator) ScoreServer(info *trust.McpServerInfo) trust.TrustScore {
	if m.scoreServerFunc != nil {
		return m.scoreServerFunc(info)
	}
	return trust.TrustScore{Tier: trust.Tier3Fallback}
}

func newAllowPolicyEvaluator() *mockPolicyEvaluator {
	return &mockPolicyEvaluator{
		evaluateFunc: func(_ *policy.EvalContext) policy.PolicyDecision {
			return policy.PolicyDecision{Action: "", ExitCode: 0}
		},
	}
}

func TestRunPreToolUse_EnforceAlwaysBlock(t *testing.T) {
	t.Parallel()

	pe := &mockPolicyEvaluator{
		evaluateFunc: func(ctx *policy.EvalContext) policy.PolicyDecision {
			if ctx.TierFilter == policy.EnforceAlwaysOnly {
				return policy.PolicyDecision{Action: policy.Block, ExitCode: 2, Message: "blocked by enforce_always"}
			}
			return policy.PolicyDecision{}
		},
	}

	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, &mockTrustEvaluator{})
	ctx := &policy.EvalContext{ToolName: "Bash"}

	code := orch.RunPreToolUse(ctx)
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}
}

func TestRunPreToolUse_ConfusedDeputyBlock(t *testing.T) {
	t.Parallel()

	pe := newAllowPolicyEvaluator()
	pe.denyRules = []policy.DenyRule{{Pattern: "/secret/*", Type: "path"}}

	te := &mockTrustEvaluator{
		checkAccessFunc: func(_ string, _ json.RawMessage, _ []policy.DenyRule) (bool, string) {
			return true, "confused deputy: access denied"
		},
	}

	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, te)
	ctx := &policy.EvalContext{
		ToolName:  "mcp__filesystem__read_file",
		ToolInput: json.RawMessage(`{"path": "/secret/key.pem"}`),
	}

	code := orch.RunPreToolUse(ctx)
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}
}

func TestRunPreToolUse_AllAllow(t *testing.T) {
	t.Parallel()

	pe := newAllowPolicyEvaluator()
	te := &mockTrustEvaluator{
		checkAccessFunc: func(_ string, _ json.RawMessage, _ []policy.DenyRule) (bool, string) {
			return false, ""
		},
	}

	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, te)
	ctx := &policy.EvalContext{
		ToolName:  "mcp__context7__get",
		ToolInput: json.RawMessage(`{}`),
	}

	code := orch.RunPreToolUse(ctx)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunPreToolUse_PolicyPanic(t *testing.T) {
	t.Parallel()

	pe := &mockPolicyEvaluator{
		evaluateFunc: func(_ *policy.EvalContext) policy.PolicyDecision {
			panic("simulated policy engine explosion")
		},
	}

	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, &mockTrustEvaluator{})
	ctx := &policy.EvalContext{ToolName: "Bash"}

	code := orch.RunPreToolUse(ctx)
	if code != 2 {
		t.Errorf("expected exit code 2 on panic (fail-closed), got %d", code)
	}
}

func TestRunPreToolUse_TrustPanic(t *testing.T) {
	t.Parallel()

	pe := newAllowPolicyEvaluator()
	te := &mockTrustEvaluator{
		checkAccessFunc: func(_ string, _ json.RawMessage, _ []policy.DenyRule) (bool, string) {
			panic("simulated trust engine explosion")
		},
	}

	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, te)
	ctx := &policy.EvalContext{
		ToolName:  "mcp__filesystem__read_file",
		ToolInput: json.RawMessage(`{"path": "/etc/shadow"}`),
	}

	code := orch.RunPreToolUse(ctx)
	if code != 2 {
		t.Errorf("expected exit code 2 on trust panic (fail-closed), got %d", code)
	}
}

func TestRunPreToolUse_NonMCPTool(t *testing.T) {
	t.Parallel()

	pe := newAllowPolicyEvaluator()

	trustCalled := false
	te := &mockTrustEvaluator{
		checkAccessFunc: func(_ string, _ json.RawMessage, _ []policy.DenyRule) (bool, string) {
			trustCalled = true
			return true, "should not reach"
		},
	}

	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, te)
	ctx := &policy.EvalContext{ToolName: "Bash"}

	code := orch.RunPreToolUse(ctx)
	if code != 0 {
		t.Errorf("expected exit code 0 for non-MCP tool, got %d", code)
	}
	if trustCalled {
		t.Error("trust CheckAccess should not be called for non-MCP tools")
	}
}

func TestRunPostToolUse_AppliesHardening(t *testing.T) {
	t.Parallel()

	te := &mockTrustEvaluator{
		applyHardeningFunc: func(serverName string, tier trust.TrustTier, output string) string {
			return "[hardened:" + serverName + "]" + output
		},
		scoreServerFunc: func(info *trust.McpServerInfo) trust.TrustScore {
			return trust.TrustScore{ServerName: info.Name, Tier: trust.Tier3Fallback}
		},
	}

	orch := NewSecurityOrchestrator(newAllowPolicyEvaluator(), &mockRiskScorer{}, te)
	ctx := &policy.EvalContext{ToolName: "mcp__context7__get-library-docs"}

	output, code := orch.RunPostToolUse(ctx, "raw library content")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	expected := "[hardened:context7]raw library content"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestRunPostToolUse_NonMCP(t *testing.T) {
	t.Parallel()

	hardeningCalled := false
	te := &mockTrustEvaluator{
		applyHardeningFunc: func(_ string, _ trust.TrustTier, output string) string {
			hardeningCalled = true
			return "modified"
		},
	}

	orch := NewSecurityOrchestrator(newAllowPolicyEvaluator(), &mockRiskScorer{}, te)
	ctx := &policy.EvalContext{ToolName: "Bash"}

	output, code := orch.RunPostToolUse(ctx, "original output")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if output != "original output" {
		t.Errorf("expected passthrough for non-MCP tool, got %q", output)
	}
	if hardeningCalled {
		t.Error("hardening should not be called for non-MCP tools")
	}
}

func TestPostureSnapshot(t *testing.T) {
	t.Parallel()

	enabled := true
	rules := []policy.CompiledRule{
		{Rule: policy.PolicyRule{ID: "R1", Category: "self-protection", BypassTier: policy.EnforceAlways, Enabled: &enabled}},
		{Rule: policy.PolicyRule{ID: "R2", Category: "config-guard", BypassTier: policy.Session, Enabled: &enabled, MonitorMode: true}},
		{Rule: policy.PolicyRule{ID: "R3", Category: "integrity", BypassTier: policy.Command, Enabled: &enabled}},
	}

	pe := &mockPolicyEvaluator{rules: rules}
	orch := NewSecurityOrchestrator(pe, &mockRiskScorer{}, &mockTrustEvaluator{})

	posture, riskPosture, trustPosture := orch.PostureSnapshot()

	if posture == nil {
		t.Fatal("expected non-nil PolicyPosture")
		return
	}
	if posture.RulesTotal != 3 {
		t.Errorf("expected RulesTotal=3, got %d", posture.RulesTotal)
	}
	if posture.RulesActive != 3 {
		t.Errorf("expected RulesActive=3, got %d", posture.RulesActive)
	}
	if posture.MonitorModeCount != 1 {
		t.Errorf("expected MonitorModeCount=1, got %d", posture.MonitorModeCount)
	}
	if posture.BypassTierSummary["enforce_always"] != 1 {
		t.Errorf("expected 1 enforce_always rule, got %d", posture.BypassTierSummary["enforce_always"])
	}
	if posture.BypassTierSummary["session"] != 1 {
		t.Errorf("expected 1 session rule, got %d", posture.BypassTierSummary["session"])
	}
	if posture.BypassTierSummary["command"] != 1 {
		t.Errorf("expected 1 command rule, got %d", posture.BypassTierSummary["command"])
	}
	if !posture.CategoryCoverage["self-protection"] {
		t.Error("expected self-protection in category coverage")
	}
	if !posture.CategoryCoverage["config-guard"] {
		t.Error("expected config-guard in category coverage")
	}
	if !posture.CategoryCoverage["integrity"] {
		t.Error("expected integrity in category coverage")
	}
	if riskPosture != nil {
		t.Error("expected nil PackageRiskPosture")
	}
	if trustPosture != nil {
		t.Error("expected nil McpTrustPosture")
	}
}

// TestProductionAdaptersSatisfyInterfaces is the compile-time guard for
// F-CAP-21.6-2 / F-CAP-21.2-1: the production trust adapter and risk scorer must
// satisfy the orchestrator interfaces so both can be wired non-nil. If either
// stops satisfying its interface, this file fails to compile.
func TestProductionAdaptersSatisfyInterfaces(t *testing.T) {
	t.Parallel()

	var _ McpTrustEvaluator = (*TrustAdapter)(nil)
	var _ McpTrustEvaluator = NewTrustAdapter(trust.NewMcpTrustEngine(""))
	var _ PackageRiskScorer = risk.NewScorer()
}

// TestRunPreToolUse_ConfusedDeputyRealCompiledRule is the end-to-end red→green
// guard for F-CAP-21.2-1 + F-CAP-21.6-1 + F-CAP-21.6-2: a REAL compiled deny rule
// (emitted as Type "denied_path" by the compiler, normalized to "path" by the
// engine) must fire the confused-deputy check through the production-wired
// orchestrator (real PolicyEngine + real TrustAdapter), blocking an MCP tool that
// launders a write into a denied path.
func TestRunPreToolUse_ConfusedDeputyRealCompiledRule(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "id_rsa")
	if err := os.WriteFile(secretFile, []byte("key"), 0o600); err != nil {
		t.Fatalf("writing secret file: %v", err)
	}

	policyYAML := fmt.Sprintf(`apiVersion: qsdev/v1
kind: SecurityPolicy
metadata:
  name: deputy-test
  version: "1.0.0"
rules:
  - id: CG-DEPUTY
    category: config-guard
    name: Deny credential path
    severity: high
    bypass_tier: session
    conditions:
      type: denied_path_check
      pattern: %q
    action:
      type: block
      message: "denied"
`, tmpDir+"/*")

	policyFile := filepath.Join(tmpDir, "policy.yaml")
	if err := os.WriteFile(policyFile, []byte(policyYAML), 0o644); err != nil {
		t.Fatalf("writing policy file: %v", err)
	}

	engine, err := policy.NewPolicyEngine([]string{policyFile}, nil, policy.EngineOptions{})
	if err != nil {
		t.Fatalf("creating policy engine: %v", err)
	}

	// The compiled rule came from denied_path_check; FilePathDenyRules must have
	// normalized its type to "path" for the trust layer to act on it.
	denyRules := engine.FilePathDenyRules()
	if len(denyRules) != 1 || denyRules[0].Type != "path" {
		t.Fatalf("expected one normalized \"path\" deny rule, got %+v", denyRules)
	}

	adapter := NewTrustAdapter(trust.NewMcpTrustEngine(filepath.Join(tmpDir, "trust.yaml")))
	orch := NewSecurityOrchestrator(engine, risk.NewScorer(), adapter)

	args, _ := json.Marshal(map[string]string{"path": secretFile})
	ctx := &policy.EvalContext{
		ToolName:  "mcp__filesystem__write_file",
		ToolInput: args,
	}

	code := orch.RunPreToolUse(ctx)
	if code != 2 {
		t.Errorf("expected confused-deputy block (exit 2) via real compiled deny rule, got %d", code)
	}
}

// TestRunPostToolUse_ProductionAdapterHardens confirms the production trust
// adapter actually transforms MCP output (F-CAP-21.8-1): a fallback-tier server's
// output is hardened (framed) rather than passed through untouched.
func TestRunPostToolUse_ProductionAdapterHardens(t *testing.T) {
	t.Parallel()

	adapter := NewTrustAdapter(trust.NewMcpTrustEngine(""))
	orch := NewSecurityOrchestrator(newAllowPolicyEvaluator(), risk.NewScorer(), adapter)

	ctx := &policy.EvalContext{ToolName: "mcp__untrusted__fetch"}
	out, code := orch.RunPostToolUse(ctx, "raw content")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if out == "raw content" {
		t.Error("expected production adapter to harden MCP output, got passthrough")
	}
	if want := "<qsdev:data"; !strings.Contains(out, want) {
		t.Errorf("expected hardened output to contain %q, got %q", want, out)
	}
}

func TestExtractServerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"mcp__context7__get", "context7"},
		{"mcp__github__edit", "github"},
		{"mcp__context7__get-library-docs", "context7"},
		{"mcp__filesystem__read_file", "filesystem"},
		{"mcp__serveronly", "serveronly"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := extractServerName(tt.input)
			if got != tt.want {
				t.Errorf("extractServerName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
