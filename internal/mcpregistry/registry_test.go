package mcpregistry

import (
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

func testDef(name string, cat McpCategory) McpServerDefinition {
	return McpServerDefinition{
		Name:            name,
		DisplayName:     name + " Server",
		Category:        cat,
		Description:     "Test server " + name,
		Command:         "/usr/bin/" + name,
		Args:            []string{"--mode", "test"},
		Transport:       TransportStdio,
		ProtocolVersion: "2024-11-05",
		ComplianceGrade: ComplianceStandard,
		Source:          SourceBuiltin,
		InstallMethod:   InstallManual,
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	def := McpServerDefinition{
		Name:            "context7",
		DisplayName:     "Context7",
		Category:        CategoryDocumentation,
		Description:     "Library documentation lookups",
		Command:         "npx",
		Args:            []string{"-y", "@upstash/context7-mcp@latest"},
		Env:             map[string]string{"NODE_ENV": "production"},
		RequiredEnv:     []string{"CONTEXT7_TOKEN"},
		Transport:       TransportStdio,
		ProtocolVersion: "2024-11-05",
		Capabilities: McpCapabilities{
			Tools:     true,
			ToolCount: 2,
		},
		ComplianceGrade: ComplianceStandard,
		Source:          SourceCatalog,
		ToolRegName:     "context7",
		InstallMethod:   InstallNpmGlobal,
		PackageName:     "@upstash/context7-mcp",
	}

	if err := r.Register(def); err != nil {
		t.Fatalf("Register() returned unexpected error: %v", err)
	}

	got, ok := r.ByName("context7")
	if !ok {
		t.Fatal("ByName() returned false for registered server")
	}
	if got.Name != "context7" {
		t.Errorf("Name = %q, want %q", got.Name, "context7")
	}
	if got.DisplayName != "Context7" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Context7")
	}
	if got.Category != CategoryDocumentation {
		t.Errorf("Category = %q, want %q", got.Category, CategoryDocumentation)
	}
	if got.Transport != TransportStdio {
		t.Errorf("Transport = %q, want %q", got.Transport, TransportStdio)
	}
	if got.ComplianceGrade != ComplianceStandard {
		t.Errorf("ComplianceGrade = %v, want %v", got.ComplianceGrade, ComplianceStandard)
	}
	if got.Source != SourceCatalog {
		t.Errorf("Source = %q, want %q", got.Source, SourceCatalog)
	}
	if got.InstallMethod != InstallNpmGlobal {
		t.Errorf("InstallMethod = %v, want %v", got.InstallMethod, InstallNpmGlobal)
	}
	if !got.Capabilities.Tools {
		t.Error("Capabilities.Tools = false, want true")
	}
	if got.Capabilities.ToolCount != 2 {
		t.Errorf("Capabilities.ToolCount = %d, want 2", got.Capabilities.ToolCount)
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	def := testDef("dup-server", CategorySecurity)

	if err := r.Register(def); err != nil {
		t.Fatalf("first Register() returned unexpected error: %v", err)
	}

	err := r.Register(def)
	if err == nil {
		t.Fatal("second Register() with same name returned nil, want error")
	}
}

func TestRegistry_MustRegister_Panics(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	def := testDef("panic-server", CategoryAgent)
	r.MustRegister(def)

	defer func() {
		if rec := recover(); rec == nil {
			t.Fatal("MustRegister with duplicate did not panic")
		}
	}()
	r.MustRegister(def)
}

func TestRegistry_All_SortedByName(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.MustRegister(testDef("zebra", CategoryAgent))
	r.MustRegister(testDef("alpha", CategorySecurity))
	r.MustRegister(testDef("middle", CategoryDocumentation))

	list := r.All()
	if len(list) != 3 {
		t.Fatalf("All() returned %d items, want 3", len(list))
	}
	want := []string{"alpha", "middle", "zebra"}
	for i, def := range list {
		if def.Name != want[i] {
			t.Errorf("All()[%d].Name = %q, want %q", i, def.Name, want[i])
		}
	}
}

func TestRegistry_ByCategory(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.MustRegister(testDef("sec-a", CategorySecurity))
	r.MustRegister(testDef("sec-b", CategorySecurity))
	r.MustRegister(testDef("doc-a", CategoryDocumentation))
	r.MustRegister(testDef("agent-a", CategoryAgent))

	tests := []struct {
		name     string
		cat      McpCategory
		wantLen  int
		wantName string
	}{
		{"security", CategorySecurity, 2, "sec-a"},
		{"documentation", CategoryDocumentation, 1, "doc-a"},
		{"agent", CategoryAgent, 1, "agent-a"},
		{"infrastructure-empty", CategoryInfrastructure, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := r.ByCategory(tt.cat)
			if len(result) != tt.wantLen {
				t.Errorf("ByCategory(%q) returned %d items, want %d", tt.cat, len(result), tt.wantLen)
			}
			if tt.wantLen > 0 && result[0].Name != tt.wantName {
				t.Errorf("ByCategory(%q)[0].Name = %q, want %q", tt.cat, result[0].Name, tt.wantName)
			}
		})
	}
}

func TestRegistry_AllHealthy(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.MustRegister(testDef("healthy-a", CategorySecurity))
	r.MustRegister(testDef("healthy-b", CategoryAgent))
	r.MustRegister(testDef("sick", CategoryDocumentation))
	r.MustRegister(testDef("unknown", CategoryIntegration))

	now := time.Now()
	r.SetHealth("healthy-a", &HealthResult{
		ServerHealth: &mcphealth.ServerHealth{Name: "healthy-a", Status: mcphealth.StatusHealthy},
		CheckedAt:    now,
	})
	r.SetHealth("healthy-b", &HealthResult{
		ServerHealth: &mcphealth.ServerHealth{Name: "healthy-b", Status: mcphealth.StatusHealthy},
		CheckedAt:    now,
	})
	r.SetHealth("sick", &HealthResult{
		ServerHealth: &mcphealth.ServerHealth{Name: "sick", Status: mcphealth.StatusUnreachable},
		CheckedAt:    now,
	})
	// "unknown" has no health result

	result := r.AllHealthy()
	if len(result) != 2 {
		t.Fatalf("AllHealthy() returned %d items, want 2", len(result))
	}
	if result[0].Name != "healthy-a" {
		t.Errorf("AllHealthy()[0].Name = %q, want %q", result[0].Name, "healthy-a")
	}
	if result[1].Name != "healthy-b" {
		t.Errorf("AllHealthy()[1].Name = %q, want %q", result[1].Name, "healthy-b")
	}
}

func TestRegistry_SetHealth_GetHealth(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.MustRegister(testDef("srv", CategorySecurity))

	_, ok := r.GetHealth("srv")
	if ok {
		t.Fatal("GetHealth() returned true before SetHealth")
	}

	now := time.Now()
	hr := &HealthResult{
		ServerHealth: &mcphealth.ServerHealth{
			Name:       "srv",
			Status:     mcphealth.StatusHealthy,
			ToolCount:  5,
			ResponseMs: 42,
		},
		CheckedAt: now,
		Stale:     false,
	}
	r.SetHealth("srv", hr)

	got, ok := r.GetHealth("srv")
	if !ok {
		t.Fatal("GetHealth() returned false after SetHealth")
	}
	if got.Status != mcphealth.StatusHealthy {
		t.Errorf("Status = %q, want %q", got.Status, mcphealth.StatusHealthy)
	}
	if got.ToolCount != 5 {
		t.Errorf("ToolCount = %d, want 5", got.ToolCount)
	}
	if got.ResponseMs != 42 {
		t.Errorf("ResponseMs = %d, want 42", got.ResponseMs)
	}
	if got.Stale {
		t.Error("Stale = true, want false")
	}
	if !got.CheckedAt.Equal(now) {
		t.Errorf("CheckedAt = %v, want %v", got.CheckedAt, now)
	}
}

func TestRegistry_Names(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.MustRegister(testDef("charlie", CategoryAgent))
	r.MustRegister(testDef("alpha", CategorySecurity))
	r.MustRegister(testDef("bravo", CategoryDocumentation))

	names := r.Names()
	want := []string{"alpha", "bravo", "charlie"}
	if len(names) != len(want) {
		t.Fatalf("Names() returned %d items, want %d", len(names), len(want))
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestRegistry_Count(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if r.Count() != 0 {
		t.Errorf("Count() on empty registry = %d, want 0", r.Count())
	}

	r.MustRegister(testDef("one", CategorySecurity))
	r.MustRegister(testDef("two", CategoryAgent))
	r.MustRegister(testDef("three", CategoryDocumentation))

	if r.Count() != 3 {
		t.Errorf("Count() = %d, want 3", r.Count())
	}
}

func TestRegistry_ByName_NotFound(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.MustRegister(testDef("exists", CategorySecurity))

	_, ok := r.ByName("nonexistent")
	if ok {
		t.Error("ByName() returned true for nonexistent server")
	}
}
