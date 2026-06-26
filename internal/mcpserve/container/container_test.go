package container

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// ---- gateway-decision derivation -----------------------------------------

func TestNeedsGatewayTier(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tier aiframework.EnforcementTier
		want bool
	}{
		{aiframework.TierKernel, false},  // native sandbox
		{aiframework.TierHook, false},    // native pre-tool hook
		{aiframework.TierPolicy, true},   // config-only
		{aiframework.TierAdvisory, true}, // instructions-only
		{aiframework.TierExternal, true}, // external isolation
	}
	for _, c := range cases {
		if got := NeedsGatewayTier(c.tier); got != c.want {
			t.Errorf("NeedsGatewayTier(%s) = %v, want %v", c.tier, got, c.want)
		}
		p := FrameworkProfile{Tier: c.tier}
		if got := p.NeedsGateway(); got != c.want {
			t.Errorf("FrameworkProfile{%s}.NeedsGateway() = %v, want %v", c.tier, got, c.want)
		}
	}
}

func TestProfileForKnownAndUnknown(t *testing.T) {
	t.Parallel()
	// Known: Claude Code is hook-tier and must not need the gateway.
	if p := ProfileFor(aiframework.ClaudeCode); p.NeedsGateway() {
		t.Errorf("ClaudeCode profile NeedsGateway = true, want false (tier %s)", p.Tier)
	}
	// Known: Cursor lacks native hooks and must need the gateway.
	if p := ProfileFor(aiframework.Cursor); !p.NeedsGateway() {
		t.Errorf("Cursor profile NeedsGateway = false, want true (tier %s)", p.Tier)
	}
	// Unknown ids default conservatively to advisory => need the gateway.
	if p := ProfileFor(aiframework.FrameworkID("brand-new-tool")); !p.NeedsGateway() {
		t.Errorf("unknown framework profile NeedsGateway = false, want true (tier %s)", p.Tier)
	}
}

// ---- deploy-mode parsing & standalone validation -------------------------

func TestParseDeployMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		flag    string
		env     string
		want    DeployMode
		wantErr bool
	}{
		{"default", "", "", DeployNative, false},
		{"flag wins", "gateway", "standalone", DeployGateway, false},
		{"env fallback", "", "standalone", DeployStandalone, false},
		{"case insensitive", "Gateway", "", DeployGateway, false},
		{"invalid", "proxy", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseDeployMode(c.flag, c.env)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ParseDeployMode(%q,%q) err = nil, want error", c.flag, c.env)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDeployMode(%q,%q) unexpected err: %v", c.flag, c.env, err)
			}
			if got != c.want {
				t.Errorf("ParseDeployMode(%q,%q) = %q, want %q", c.flag, c.env, got, c.want)
			}
		})
	}
}

func TestStandaloneProjectRoot(t *testing.T) {
	t.Parallel()

	// Explicit flag wins.
	if got, err := StandaloneProjectRoot("/srv/app", nil); err != nil || got != "/srv/app" {
		t.Fatalf("flag root = (%q,%v), want (/srv/app,nil)", got, err)
	}

	// Env fallback when flag is empty.
	env := func(k string) string {
		if k == envProjectRoot {
			return "/mnt/proj"
		}
		return ""
	}
	if got, err := StandaloneProjectRoot("", env); err != nil || got != "/mnt/proj" {
		t.Fatalf("env root = (%q,%v), want (/mnt/proj,nil)", got, err)
	}

	// Neither source: must refuse CWD detection with the sentinel error.
	if _, err := StandaloneProjectRoot("", func(string) string { return "" }); !errors.Is(err, ErrStandaloneRootRequired) {
		t.Fatalf("missing root err = %v, want ErrStandaloneRootRequired", err)
	}
	// Nil getenv with empty flag also errors.
	if _, err := StandaloneProjectRoot("", nil); !errors.Is(err, ErrStandaloneRootRequired) {
		t.Fatalf("nil getenv err = %v, want ErrStandaloneRootRequired", err)
	}
}

// ---- container-config generation -----------------------------------------

func TestGenerateGatewayForHooklessFramework(t *testing.T) {
	t.Parallel()

	gen := ContainerConfigGenerator{}
	art, err := gen.Generate(GenerateOptions{
		ProjectRoot: "/srv/project",
		Frameworks: []FrameworkProfile{
			{ID: aiframework.Cursor, Tier: aiframework.TierAdvisory},
			{ID: aiframework.ClaudeCode, Tier: aiframework.TierHook, NativeHooks: true}, // must NOT trigger the gateway
		},
		Port: 9000,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !art.NeedsGateway {
		t.Fatal("NeedsGateway = false, want true (Cursor lacks native hooks)")
	}
	if len(art.GatewayFrameworks) != 1 || art.GatewayFrameworks[0] != aiframework.Cursor {
		t.Fatalf("GatewayFrameworks = %v, want [cursor] (Claude Code is hook-tier)", art.GatewayFrameworks)
	}
	if art.EnvVars[envDeployMode] != string(DeployGateway) {
		t.Errorf("env %s = %q, want gateway", envDeployMode, art.EnvVars[envDeployMode])
	}

	// The compose fragment must be valid YAML that round-trips.
	var parsed struct {
		Services map[string]struct {
			Image       string            `yaml:"image"`
			Command     []string          `yaml:"command"`
			Ports       []string          `yaml:"ports"`
			Volumes     []string          `yaml:"volumes"`
			Environment map[string]string `yaml:"environment"`
			SecurityOpt []string          `yaml:"security_opt"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(art.ComposeYAML), &parsed); err != nil {
		t.Fatalf("compose YAML did not parse: %v\n%s", err, art.ComposeYAML)
	}
	svc, ok := parsed.Services[DefaultServiceName]
	if !ok {
		t.Fatalf("compose missing service %q; services=%v", DefaultServiceName, parsed.Services)
	}
	if svc.Image != DefaultImage {
		t.Errorf("service image = %q, want %q", svc.Image, DefaultImage)
	}
	if svc.Environment[envDeployMode] != string(DeployGateway) {
		t.Errorf("service env %s = %q, want gateway", envDeployMode, svc.Environment[envDeployMode])
	}
	if !containsStr(svc.Ports, "9000:9000") {
		t.Errorf("service ports = %v, want 9000:9000", svc.Ports)
	}
	if !containsStr(svc.Volumes, "/srv/project:"+ContainerWorkspace+":ro") {
		t.Errorf("service volumes = %v, want read-only project mount", svc.Volumes)
	}

	// R18: the ENTRYPOINT already supplies `/qsdev mcp serve`, and docker APPENDS
	// command to it — so command must carry ONLY flags. A leading `mcp`/`serve`
	// would double the subcommand.
	if len(svc.Command) == 0 || !strings.HasPrefix(svc.Command[0], "-") {
		t.Fatalf("command must start with a flag (ENTRYPOINT supplies `mcp serve`); got %v", svc.Command)
	}
	for _, a := range svc.Command {
		if a == "mcp" || a == "serve" {
			t.Errorf("command carries leading subcommand %q (doubled argv vs ENTRYPOINT); got %v", a, svc.Command)
		}
	}
	// R3: the generated gateway is the secure mTLS path — it must bind a reachable
	// address and point at the mounted mTLS material.
	if !commandBindsAll(svc.Command) {
		t.Errorf("command missing --bind %s; got %v", GatewayBindAll, svc.Command)
	}
	for env, want := range map[string]string{
		envTLSCert:     ContainerTLSCert,
		envTLSKey:      ContainerTLSKey,
		envTLSClientCA: ContainerTLSClientCA,
	} {
		if got := svc.Environment[env]; got != want {
			t.Errorf("service env %s = %q, want %q (fail-closed mTLS path)", env, got, want)
		}
	}
	if !containsStr(svc.Volumes, "${QSDEV_TLS_CERT:-./tls/server.crt}:"+ContainerTLSCert+":ro") {
		t.Errorf("service volumes = %v, want a read-only server-cert mount", svc.Volumes)
	}
	// R19: hardening — security_opt no-new-privileges must be rendered.
	if !containsStr(svc.SecurityOpt, noNewPrivileges) {
		t.Errorf("service security_opt = %v, want %q", svc.SecurityOpt, noNewPrivileges)
	}

	// The .mcp.json entry must point at the container's mTLS HTTP port over https.
	var mcp struct {
		Servers map[string]MCPServerConfig `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(art.MCPJSON), &mcp); err != nil {
		t.Fatalf(".mcp.json did not parse: %v\n%s", err, art.MCPJSON)
	}
	entry, ok := mcp.Servers[DefaultMCPServerName]
	if !ok {
		t.Fatalf(".mcp.json missing server %q; got %v", DefaultMCPServerName, mcp.Servers)
	}
	if entry.Type != "http" || entry.URL != "https://localhost:9000/mcp" {
		t.Errorf(".mcp.json entry = %+v, want https://localhost:9000/mcp", entry)
	}
}

func TestGenerateNoGatewayForHookOnlyProject(t *testing.T) {
	t.Parallel()

	gen := ContainerConfigGenerator{}
	art, err := gen.Generate(GenerateOptions{
		Frameworks: []FrameworkProfile{
			{ID: aiframework.ClaudeCode, Tier: aiframework.TierHook, NativeHooks: true},
			{ID: aiframework.Codex, Tier: aiframework.TierKernel, NativeHooks: true},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if art.NeedsGateway {
		t.Errorf("NeedsGateway = true, want false (both frameworks enforce natively)")
	}
	if art.ComposeYAML != "" || art.MCPJSON != "" {
		t.Errorf("expected no compose/.mcp.json output for an all-native project; got compose=%q mcp=%q",
			art.ComposeYAML, art.MCPJSON)
	}
}

// ---- gateway 4-layer enforcement -----------------------------------------

// authedCtx builds a ToolCallContext for an agent in a given category.
func authedCtx(agent, category string) *spi.ToolCallContext {
	return &spi.ToolCallContext{AgentID: agent, Category: category}
}

func TestGatewayInterceptorBlocksAndAllows(t *testing.T) {
	t.Parallel()

	chain := GatewayChain(GatewayOptions{
		AllowedAgents: []string{"trusted-agent"},
		RequireAuth:   true,
	})

	ran := false
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		ran = true
		return &spi.ToolResult{Text: "ok"}, nil
	}

	// Disallowed agent: short-circuit, handler never runs.
	res, err := chain.Execute(context.Background(), authedCtx("intruder", "general"),
		&spi.ToolRequest{Name: "qsdev_status"}, final)
	if err != nil {
		t.Fatalf("Execute(intruder): %v", err)
	}
	if ran {
		t.Error("handler ran for an unauthenticated agent")
	}
	if res == nil || !res.IsError || !strings.Contains(res.Text, "authorization failed") {
		t.Fatalf("intruder result = %+v, want authz-failure IsError", res)
	}

	// Allowed agent: passes through to the handler.
	ran = false
	res, err = chain.Execute(context.Background(), authedCtx("trusted-agent", "general"),
		&spi.ToolRequest{Name: "qsdev_status"}, final)
	if err != nil {
		t.Fatalf("Execute(trusted): %v", err)
	}
	if !ran {
		t.Error("handler did not run for an allowed agent")
	}
	if res == nil || res.IsError || res.Text != "ok" {
		t.Fatalf("trusted result = %+v, want ok", res)
	}
}

func TestGatewayChainAppliesRateLimits(t *testing.T) {
	t.Parallel()

	clk := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	chain := GatewayChain(GatewayOptions{
		// One token, no refill, frozen clock => the second call is starved.
		Limits: middleware.CategoryLimits{Default: middleware.Limit{Rate: 0, Burst: 1, Concurrency: 0}},
		Clock:  func() time.Time { return clk },
	})

	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return &spi.ToolResult{Text: "ok"}, nil
	}
	cc := authedCtx("agent", "general")
	req := &spi.ToolRequest{Name: "qsdev_status"}

	first, err := chain.Execute(context.Background(), cc, req, final)
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	if first.IsError {
		t.Fatalf("first call rate-limited unexpectedly: %+v", first)
	}

	second, err := chain.Execute(context.Background(), cc, req, final)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if second == nil || !second.IsError || !strings.Contains(second.Text, "rate limit") {
		t.Fatalf("second call result = %+v, want rate-limit IsError", second)
	}
}

func TestGatewayChainReusedGuardrailRuns(t *testing.T) {
	t.Parallel()

	// A deny policy on the "general" category proves the REUSED Guardrail layer
	// still runs inside the gateway chain (auth disabled so it cannot interfere).
	chain := GatewayChain(GatewayOptions{
		Policy: &middleware.Policy{
			ByToolType: map[string]middleware.Verdict{"general": middleware.VerdictDeny},
			Default:    middleware.VerdictAllow,
		},
	})

	ran := false
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		ran = true
		return &spi.ToolResult{Text: "ok"}, nil
	}
	res, err := chain.Execute(context.Background(), authedCtx("agent", "general"),
		&spi.ToolRequest{Name: "qsdev_status"}, final)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if ran {
		t.Error("handler ran despite a deny policy")
	}
	if res == nil || !res.IsError || !strings.Contains(res.Text, "denied by policy") {
		t.Fatalf("result = %+v, want guardrail deny", res)
	}
}

func containsStr(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// commandBindsAll reports whether cmd requests a 0.0.0.0 bind in either the
// "--bind=0.0.0.0" or the "--bind","0.0.0.0" form.
func commandBindsAll(cmd []string) bool {
	want := GatewayBindAll
	for i, a := range cmd {
		if a == "--bind="+want {
			return true
		}
		if a == "--bind" && i+1 < len(cmd) && cmd[i+1] == want {
			return true
		}
	}
	return false
}
