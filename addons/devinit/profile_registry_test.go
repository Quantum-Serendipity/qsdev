package devinit_test

import (
	"sync"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devinit"
)

func TestProjectProfileRegistry_RegisterAndGet(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	p := devinit.ExportProfile{
		Description: "test profile",
		Languages: []devinit.ExportLanguageSpec{
			{Name: "go", Version: "1.24"},
		},
		Direnv: true,
	}
	if err := r.Register("test-profile", p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := r.Get("test-profile")
	if !ok {
		t.Fatal("Get returned not found for registered profile")
	}
	if got.Description != "test profile" {
		t.Errorf("Description = %q, want %q", got.Description, "test profile")
	}
	if len(got.Languages) != 1 || got.Languages[0].Name != "go" {
		t.Errorf("Languages = %v, want [{Name:go Version:1.24}]", got.Languages)
	}
}

func TestProjectProfileRegistry_GetNotFound(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get returned found for nonexistent profile")
	}
}

func TestProjectProfileRegistry_DuplicateRegistration(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	p := devinit.ExportProfile{Description: "dup"}
	if err := r.Register("dup", p); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	if err := r.Register("dup", p); err == nil {
		t.Error("second Register should return error for duplicate")
	}
}

func TestProjectProfileRegistry_ListPreservesInsertionOrder(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	_ = r.Register("charlie", devinit.ExportProfile{Description: "C"})
	_ = r.Register("alpha", devinit.ExportProfile{Description: "A"})
	_ = r.Register("bravo", devinit.ExportProfile{Description: "B"})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("List length = %d, want 3", len(list))
	}

	// Insertion order, not alphabetical.
	want := []string{"charlie", "alpha", "bravo"}
	for i, s := range list {
		if s.Name != want[i] {
			t.Errorf("List[%d].Name = %q, want %q", i, s.Name, want[i])
		}
	}
}

func TestProjectProfileRegistry_ListIncludesDescription(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	_ = r.Register("test", devinit.ExportProfile{Description: "my description"})

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("List length = %d, want 1", len(list))
	}
	if list[0].Description != "my description" {
		t.Errorf("Description = %q, want %q", list[0].Description, "my description")
	}
}

func TestProjectProfileRegistry_Names(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	_ = r.Register("first", devinit.ExportProfile{})
	_ = r.Register("second", devinit.ExportProfile{})
	_ = r.Register("third", devinit.ExportProfile{})

	names := r.Names()
	want := []string{"first", "second", "third"}
	if len(names) != len(want) {
		t.Fatalf("Names length = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestProjectProfileRegistry_ConcurrentAccess(t *testing.T) {
	r := devinit.ExportNewProjectProfileRegistry()

	var wg sync.WaitGroup
	// Concurrent writers.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := "profile-" + string(rune('A'+n%26)) + string(rune('0'+n/26))
			_ = r.Register(name, devinit.ExportProfile{Description: name})
		}(i)
	}

	// Concurrent readers.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.List()
			_ = r.Names()
			_, _ = r.Get("profile-A0")
		}()
	}

	wg.Wait()

	// Verify no panic occurred (the test completing is the assertion).
	// Also verify the registry is internally consistent.
	names := r.Names()
	list := r.List()
	if len(names) != len(list) {
		t.Errorf("Names/List length mismatch: %d vs %d", len(names), len(list))
	}
}
