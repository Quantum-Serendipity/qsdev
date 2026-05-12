package profile

import (
	"testing"
)

func TestProfileRegistry_RegisterAndGet(t *testing.T) {
	r := NewProfileRegistry()

	p := &InfraProfile{Name: "test-profile", Description: "A test profile"}
	if err := r.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := r.Get("test-profile")
	if !ok {
		t.Fatal("Get returned not found for registered profile")
	}
	if got.Name != "test-profile" {
		t.Errorf("Got name %q, want test-profile", got.Name)
	}
}

func TestProfileRegistry_GetNotFound(t *testing.T) {
	r := NewProfileRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get returned found for nonexistent profile")
	}
}

func TestProfileRegistry_DuplicateRegistration(t *testing.T) {
	r := NewProfileRegistry()

	p := &InfraProfile{Name: "dup"}
	if err := r.Register(p); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	if err := r.Register(p); err == nil {
		t.Error("second Register should return error for duplicate")
	}
}

func TestProfileRegistry_ListSorted(t *testing.T) {
	r := NewProfileRegistry()

	_ = r.Register(&InfraProfile{Name: "charlie"})
	_ = r.Register(&InfraProfile{Name: "alpha"})
	_ = r.Register(&InfraProfile{Name: "bravo"})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("List length = %d, want 3", len(list))
	}

	want := []string{"alpha", "bravo", "charlie"}
	for i, p := range list {
		if p.Name != want[i] {
			t.Errorf("List[%d].Name = %q, want %q", i, p.Name, want[i])
		}
	}
}

func TestDefaultProfileRegistry_BuiltinsRegistered(t *testing.T) {
	r := DefaultProfileRegistry()

	builtins := []string{"consulting-default", "startup-github", "enterprise"}
	for _, name := range builtins {
		p, ok := r.Get(name)
		if !ok {
			t.Errorf("built-in profile %q not found in DefaultProfileRegistry", name)
			continue
		}
		if p.Name != name {
			t.Errorf("profile name = %q, want %q", p.Name, name)
		}
	}

	list := r.List()
	if len(list) != len(builtins) {
		t.Errorf("List length = %d, want %d", len(list), len(builtins))
	}
}
