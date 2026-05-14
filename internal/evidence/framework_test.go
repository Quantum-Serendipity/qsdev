package evidence

import (
	"testing"
)

func TestNewFrameworkRegistry(t *testing.T) {
	r := NewFrameworkRegistry()
	if r == nil {
		t.Fatal("NewFrameworkRegistry returned nil")
	}
	if got := r.List(); len(got) != 0 {
		t.Errorf("new registry should be empty, got %d frameworks", len(got))
	}
}

func TestFrameworkRegistry_Register(t *testing.T) {
	r := NewFrameworkRegistry()
	fw := Framework{
		ID:          "test-fw",
		Name:        "Test Framework",
		Version:     "1.0",
		Description: "A test framework",
		Controls:    func() []ControlDefinition { return nil },
	}

	if err := r.Register(fw); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, ok := r.Get("test-fw")
	if !ok {
		t.Fatal("Get returned false for registered framework")
	}
	if got.ID != "test-fw" {
		t.Errorf("got ID %q, want %q", got.ID, "test-fw")
	}
	if got.Name != "Test Framework" {
		t.Errorf("got Name %q, want %q", got.Name, "Test Framework")
	}
}

func TestFrameworkRegistry_RegisterDuplicate(t *testing.T) {
	r := NewFrameworkRegistry()
	fw := Framework{
		ID:       "dup",
		Name:     "Duplicate",
		Controls: func() []ControlDefinition { return nil },
	}

	if err := r.Register(fw); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := r.Register(fw)
	if err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestFrameworkRegistry_GetMissing(t *testing.T) {
	r := NewFrameworkRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("Get should return false for missing framework")
	}
}

func TestFrameworkRegistry_List(t *testing.T) {
	r := NewFrameworkRegistry()

	_ = r.Register(Framework{
		ID:          "zz",
		Name:        "ZZ Framework",
		Version:     "2.0",
		Description: "Second",
		Controls:    func() []ControlDefinition { return nil },
	})
	_ = r.Register(Framework{
		ID:          "aa",
		Name:        "AA Framework",
		Version:     "1.0",
		Description: "First",
		Controls:    func() []ControlDefinition { return nil },
	})

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 frameworks, got %d", len(list))
	}

	// Should be sorted by ID.
	if list[0].ID != "aa" {
		t.Errorf("first framework should be 'aa', got %q", list[0].ID)
	}
	if list[1].ID != "zz" {
		t.Errorf("second framework should be 'zz', got %q", list[1].ID)
	}
}

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()
	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 frameworks in default registry, got %d", len(list))
	}

	// Verify all three are present.
	ids := make(map[string]bool)
	for _, f := range list {
		ids[f.ID] = true
	}
	for _, expected := range []string{"soc2", "hipaa", "asvs"} {
		if !ids[expected] {
			t.Errorf("default registry missing framework %q", expected)
		}
	}
}

func TestFrameworkRegistry_ListInfo(t *testing.T) {
	r := NewFrameworkRegistry()
	_ = r.Register(Framework{
		ID:          "test",
		Name:        "Test",
		Version:     "3.0",
		Description: "Desc",
		Controls:    func() []ControlDefinition { return nil },
	})

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	info := list[0]
	if info.ID != "test" || info.Name != "Test" || info.Version != "3.0" || info.Description != "Desc" {
		t.Errorf("FrameworkInfo fields mismatch: %+v", info)
	}
}
