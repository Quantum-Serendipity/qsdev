package ecosystem_test

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
)

func newTestRegistry(t *testing.T, modules ...*ecosystem.MockModule) *ecosystem.Registry {
	t.Helper()
	r := ecosystem.NewRegistry()
	for _, m := range modules {
		if err := r.Register(m); err != nil {
			t.Fatalf("Register(%q): %v", m.Name(), err)
		}
	}
	return r
}

func TestRegisterAndByName(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{NameVal: "go", DisplayNameVal: "Go", TierVal: 1},
		&ecosystem.MockModule{NameVal: "python", DisplayNameVal: "Python", TierVal: 1},
	)

	m, ok := r.ByName("go")
	if !ok {
		t.Fatal("ByName(go) not found")
	}
	if m.DisplayName() != "Go" {
		t.Errorf("DisplayName() = %q, want %q", m.DisplayName(), "Go")
	}

	m, ok = r.ByName("python")
	if !ok {
		t.Fatal("ByName(python) not found")
	}
	if m.DisplayName() != "Python" {
		t.Errorf("DisplayName() = %q, want %q", m.DisplayName(), "Python")
	}
}

func TestByNameNotFound(t *testing.T) {
	r := ecosystem.NewRegistry()
	_, ok := r.ByName("nonexistent")
	if ok {
		t.Error("ByName(nonexistent) should return false")
	}
}

func TestRegisterDuplicateReturnsError(t *testing.T) {
	r := ecosystem.NewRegistry()
	m := &ecosystem.MockModule{NameVal: "go"}
	if err := r.Register(m); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := r.Register(&ecosystem.MockModule{NameVal: "go"})
	if err == nil {
		t.Fatal("expected error on duplicate Register, got nil")
	}
	wantMsg := `ecosystem module "go" is already registered`
	if err.Error() != wantMsg {
		t.Errorf("error = %q, want %q", err.Error(), wantMsg)
	}
}

func TestAllSortedByTierThenName(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{NameVal: "terraform", TierVal: 2},
		&ecosystem.MockModule{NameVal: "go", TierVal: 1},
		&ecosystem.MockModule{NameVal: "python", TierVal: 1},
		&ecosystem.MockModule{NameVal: "zig", TierVal: 3},
		&ecosystem.MockModule{NameVal: "docker", TierVal: 2},
		&ecosystem.MockModule{NameVal: "javascript", TierVal: 1},
	)

	all := r.All()
	wantOrder := []string{"go", "javascript", "python", "docker", "terraform", "zig"}

	if len(all) != len(wantOrder) {
		t.Fatalf("All() returned %d modules, want %d", len(all), len(wantOrder))
	}
	for i, m := range all {
		if m.Name() != wantOrder[i] {
			t.Errorf("All()[%d].Name() = %q, want %q", i, m.Name(), wantOrder[i])
		}
	}
}

func TestByTierFiltering(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{NameVal: "go", TierVal: 1},
		&ecosystem.MockModule{NameVal: "python", TierVal: 1},
		&ecosystem.MockModule{NameVal: "docker", TierVal: 2},
		&ecosystem.MockModule{NameVal: "terraform", TierVal: 2},
		&ecosystem.MockModule{NameVal: "zig", TierVal: 3},
	)

	tier1 := r.ByTier(1)
	if len(tier1) != 2 {
		t.Fatalf("ByTier(1) returned %d modules, want 2", len(tier1))
	}
	if tier1[0].Name() != "go" || tier1[1].Name() != "python" {
		t.Errorf("ByTier(1) = [%q, %q], want [go, python]", tier1[0].Name(), tier1[1].Name())
	}

	tier2 := r.ByTier(2)
	if len(tier2) != 2 {
		t.Fatalf("ByTier(2) returned %d modules, want 2", len(tier2))
	}
	if tier2[0].Name() != "docker" || tier2[1].Name() != "terraform" {
		t.Errorf("ByTier(2) = [%q, %q], want [docker, terraform]", tier2[0].Name(), tier2[1].Name())
	}

	tier3 := r.ByTier(3)
	if len(tier3) != 1 || tier3[0].Name() != "zig" {
		t.Errorf("ByTier(3) unexpected: %v", tier3)
	}

	tier99 := r.ByTier(99)
	if len(tier99) != 0 {
		t.Errorf("ByTier(99) returned %d modules, want 0", len(tier99))
	}
}

func TestNamesSorted(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{NameVal: "python"},
		&ecosystem.MockModule{NameVal: "go"},
		&ecosystem.MockModule{NameVal: "docker"},
	)

	names := r.Names()
	want := []string{"docker", "go", "python"}
	if len(names) != len(want) {
		t.Fatalf("Names() returned %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestCount(t *testing.T) {
	r := ecosystem.NewRegistry()
	if r.Count() != 0 {
		t.Errorf("Count() = %d, want 0", r.Count())
	}

	_ = r.Register(&ecosystem.MockModule{NameVal: "a"})
	_ = r.Register(&ecosystem.MockModule{NameVal: "b"})
	if r.Count() != 2 {
		t.Errorf("Count() = %d, want 2", r.Count())
	}
}

func TestDetectAllDelegation(t *testing.T) {
	goMod := &ecosystem.MockModule{
		NameVal: "go",
		TierVal: 1,
		DetectResult: ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceCertain,
			Evidence:   []string{"go.mod"},
			SuggestedConfig: ecosystem.ModuleConfig{
				Version: "1.22",
			},
		},
	}
	python := &ecosystem.MockModule{
		NameVal: "python",
		TierVal: 1,
		DetectResult: ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		},
	}
	docker := &ecosystem.MockModule{
		NameVal: "docker",
		TierVal: 2,
		DetectResult: ecosystem.DetectionResult{
			Detected:   true,
			Confidence: ecosystem.ConfidenceCertain,
			Evidence:   []string{"Dockerfile"},
		},
	}

	r := newTestRegistry(t, goMod, python, docker)
	summary := r.DetectAll("/tmp/project")

	if len(summary.Results) != 3 {
		t.Fatalf("DetectAll returned %d results, want 3", len(summary.Results))
	}

	goResult := summary.Results["go"]
	if !goResult.Detected {
		t.Error("go should be detected")
	}

	pythonResult := summary.Results["python"]
	if pythonResult.Detected {
		t.Error("python should not be detected")
	}

	if !summary.Project.HasGoMod {
		t.Error("Project.HasGoMod should be true")
	}
	if summary.Project.GoVersion != "1.22" {
		t.Errorf("Project.GoVersion = %q, want %q", summary.Project.GoVersion, "1.22")
	}
	if !summary.Project.HasDockerfile {
		t.Error("Project.HasDockerfile should be true")
	}
	if !summary.Project.Ecosystems["go"] {
		t.Error("Ecosystems[go] should be true")
	}
	if !summary.Project.Ecosystems["docker"] {
		t.Error("Ecosystems[docker] should be true")
	}
	if summary.Project.Ecosystems["python"] {
		t.Error("Ecosystems[python] should be false")
	}
}

func TestDetectAllCallsDetectFn(t *testing.T) {
	var calledWith string
	m := &ecosystem.MockModule{
		NameVal: "custom",
		DetectFn: func(projectRoot string) ecosystem.DetectionResult {
			calledWith = projectRoot
			return ecosystem.DetectionResult{Detected: true, Confidence: ecosystem.ConfidenceCertain}
		},
	}

	r := newTestRegistry(t, m)
	r.DetectAll("/my/root")

	if calledWith != "/my/root" {
		t.Errorf("Detect called with %q, want %q", calledWith, "/my/root")
	}
}

func TestConcurrentRegistration(t *testing.T) {
	r := ecosystem.NewRegistry()
	const n = 100
	var wg sync.WaitGroup
	errs := make(chan error, n)

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			m := &ecosystem.MockModule{NameVal: fmt.Sprintf("mod-%03d", idx)}
			if err := r.Register(m); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error during concurrent registration: %v", err)
	}

	if r.Count() != n {
		t.Errorf("Count() = %d, want %d", r.Count(), n)
	}

	names := r.Names()
	if !sort.StringsAreSorted(names) {
		t.Error("Names() not sorted after concurrent registration")
	}
}

func TestConcurrentReadsDuringRegistration(t *testing.T) {
	r := ecosystem.NewRegistry()

	// Pre-populate some modules.
	for i := range 10 {
		_ = r.Register(&ecosystem.MockModule{NameVal: fmt.Sprintf("pre-%02d", i)})
	}

	var wg sync.WaitGroup
	const writers = 20
	const readers = 50

	// Launch concurrent writers.
	wg.Add(writers)
	for i := range writers {
		go func(idx int) {
			defer wg.Done()
			_ = r.Register(&ecosystem.MockModule{NameVal: fmt.Sprintf("new-%03d", idx)})
		}(i)
	}

	// Launch concurrent readers.
	wg.Add(readers)
	for range readers {
		go func() {
			defer wg.Done()
			_ = r.All()
			_ = r.Names()
			_ = r.Count()
			_, _ = r.ByName("pre-00")
			_ = r.ByTier(0)
		}()
	}

	wg.Wait()
	// If we get here without a race detector panic, concurrency is handled correctly.
}

func TestDefaultRegistryIsSingleton(t *testing.T) {
	r1 := ecosystem.DefaultRegistry()
	r2 := ecosystem.DefaultRegistry()
	if r1 != r2 {
		t.Error("DefaultRegistry() should return the same instance")
	}
}

func TestEmptyRegistryAll(t *testing.T) {
	r := ecosystem.NewRegistry()
	all := r.All()
	if len(all) != 0 {
		t.Errorf("All() on empty registry returned %d, want 0", len(all))
	}
}

func TestEmptyRegistryNames(t *testing.T) {
	r := ecosystem.NewRegistry()
	names := r.Names()
	if len(names) != 0 {
		t.Errorf("Names() on empty registry returned %d, want 0", len(names))
	}
}
