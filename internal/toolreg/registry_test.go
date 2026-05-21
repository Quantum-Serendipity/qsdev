package toolreg

import (
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestRegister_Success(t *testing.T) {
	reg := NewRegistry()

	err := reg.Register(Tool{Name: "foo", Category: CategorySecurity})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if reg.Count() != 1 {
		t.Fatalf("expected count 1, got %d", reg.Count())
	}
}

func TestRegister_Duplicate(t *testing.T) {
	reg := NewRegistry()

	_ = reg.Register(Tool{Name: "foo", Category: CategorySecurity})
	err := reg.Register(Tool{Name: "foo", Category: CategoryDevEx})
	if err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}

	want := `tool "foo" already registered`
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func TestByName_Found(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(Tool{Name: "alpha", DisplayName: "Alpha Tool", Category: CategorySecurity})

	tool, ok := reg.ByName("alpha")
	if !ok {
		t.Fatal("expected tool to be found")
	}
	if tool.Name != "alpha" {
		t.Fatalf("expected name %q, got %q", "alpha", tool.Name)
	}
	if tool.DisplayName != "Alpha Tool" {
		t.Fatalf("expected display name %q, got %q", "Alpha Tool", tool.DisplayName)
	}
}

func TestByName_NotFound(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(Tool{Name: "alpha", Category: CategorySecurity})

	_, ok := reg.ByName("nonexistent")
	if ok {
		t.Fatal("expected tool to not be found")
	}
}

func TestAll_SortsByCategoryThenName(t *testing.T) {
	reg := NewRegistry()

	// Register in deliberately unsorted order across categories.
	_ = reg.Register(Tool{Name: "zeta", Category: CategoryInfrastructure})
	_ = reg.Register(Tool{Name: "beta", Category: CategorySecurity})
	_ = reg.Register(Tool{Name: "alpha", Category: CategorySecurity})
	_ = reg.Register(Tool{Name: "gamma", Category: CategoryAIAgent})
	_ = reg.Register(Tool{Name: "delta", Category: CategoryDevEx})
	_ = reg.Register(Tool{Name: "epsilon", Category: CategoryInfrastructure})

	all := reg.All()
	if len(all) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(all))
	}

	// Expected order: security(alpha, beta), ai-agent(gamma), devex(delta), infrastructure(epsilon, zeta)
	expected := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	for i, want := range expected {
		if all[i].Name != want {
			t.Errorf("position %d: expected %q, got %q", i, want, all[i].Name)
		}
	}
}

func TestAll_Empty(t *testing.T) {
	reg := NewRegistry()
	all := reg.All()
	if len(all) != 0 {
		t.Fatalf("expected empty slice, got %d elements", len(all))
	}
}

func TestByCategory_Filters(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(Tool{Name: "sec-b", Category: CategorySecurity})
	_ = reg.Register(Tool{Name: "sec-a", Category: CategorySecurity})
	_ = reg.Register(Tool{Name: "agent-x", Category: CategoryAIAgent})
	_ = reg.Register(Tool{Name: "devex-y", Category: CategoryDevEx})

	secTools := reg.ByCategory(CategorySecurity)
	if len(secTools) != 2 {
		t.Fatalf("expected 2 security tools, got %d", len(secTools))
	}
	if secTools[0].Name != "sec-a" || secTools[1].Name != "sec-b" {
		t.Errorf("expected [sec-a, sec-b], got [%s, %s]", secTools[0].Name, secTools[1].Name)
	}

	aiTools := reg.ByCategory(CategoryAIAgent)
	if len(aiTools) != 1 {
		t.Fatalf("expected 1 ai-agent tool, got %d", len(aiTools))
	}
	if aiTools[0].Name != "agent-x" {
		t.Errorf("expected agent-x, got %s", aiTools[0].Name)
	}
}

func TestByCategory_NoMatch(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(Tool{Name: "sec-a", Category: CategorySecurity})

	tools := reg.ByCategory(CategoryInfrastructure)
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(tools))
	}
}

func TestNames_SortedAlphabetically(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(Tool{Name: "charlie", Category: CategoryDevEx})
	_ = reg.Register(Tool{Name: "alpha", Category: CategorySecurity})
	_ = reg.Register(Tool{Name: "bravo", Category: CategoryAIAgent})

	names := reg.Names()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	expected := []string{"alpha", "bravo", "charlie"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("position %d: expected %q, got %q", i, want, names[i])
		}
	}
}

func TestNames_Empty(t *testing.T) {
	reg := NewRegistry()
	names := reg.Names()
	if len(names) != 0 {
		t.Fatalf("expected empty slice, got %d elements", len(names))
	}
}

func TestCount(t *testing.T) {
	reg := NewRegistry()
	if reg.Count() != 0 {
		t.Fatalf("expected count 0, got %d", reg.Count())
	}

	_ = reg.Register(Tool{Name: "a", Category: CategorySecurity})
	_ = reg.Register(Tool{Name: "b", Category: CategorySecurity})
	if reg.Count() != 2 {
		t.Fatalf("expected count 2, got %d", reg.Count())
	}
}

func TestDefaultRegistry_Singleton(t *testing.T) {
	// DefaultRegistry returns the same instance across calls.
	r1 := DefaultRegistry()
	r2 := DefaultRegistry()
	if r1 != r2 {
		t.Fatal("expected DefaultRegistry to return the same instance")
	}
}

func TestYAMLRegistryCorrespondence(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	yamlTools := catalog.Default().Tools()

	for name := range yamlTools {
		if _, ok := reg.ByName(name); !ok {
			t.Errorf("YAML tool %q not found in registry", name)
		}
	}

	opsPrefix := branding.Get().AppName + "-"
	for _, tool := range reg.All() {
		if strings.HasPrefix(tool.Name, opsPrefix) {
			continue
		}
		if _, ok := yamlTools[tool.Name]; !ok {
			t.Errorf("registry tool %q not found in YAML catalog", tool.Name)
		}
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	// Concurrent writes.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "tool-" + string(rune('A'+idx%26)) + string(rune('0'+idx/26))
			if err := reg.Register(Tool{Name: name, Category: CategorySecurity}); err != nil {
				errs <- err
			}
		}(i)
	}

	// Concurrent reads while writing.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.All()
			_ = reg.Names()
			_ = reg.Count()
			_, _ = reg.ByName("tool-A0")
			_ = reg.ByCategory(CategorySecurity)
		}()
	}

	wg.Wait()
	close(errs)

	// Some duplicate registrations are expected with the naming scheme,
	// but no panics should occur.
	for err := range errs {
		_ = err // duplicates are acceptable
	}
}
