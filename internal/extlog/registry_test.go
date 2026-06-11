package extlog

import (
	"io"
	"sync"
	"testing"
	"time"
)

// fakeProvider implements LogProvider for testing.
type fakeProvider struct {
	name        string
	displayName string
	detected    bool
}

func (f *fakeProvider) Name() string        { return f.name }
func (f *fakeProvider) DisplayName() string { return f.displayName }

func (f *fakeProvider) Detect(_, _ string) bool { return f.detected }

func (f *fakeProvider) Discover(_, _ string, _ time.Time) ([]LogFile, error) {
	return nil, nil
}

func (f *fakeProvider) Parse(_ io.Reader, _ string) ([]LogEntry, error) {
	return nil, nil
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.All()) != 0 {
		t.Errorf("new registry has %d providers, want 0", len(r.All()))
	}
}

func TestRegistryRegisterAndAll(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(&fakeProvider{name: "npm", displayName: "npm"})
	r.Register(&fakeProvider{name: "nix", displayName: "Nix"})

	all := r.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d providers, want 2", len(all))
	}

	names := make(map[string]bool)
	for _, p := range all {
		names[p.Name()] = true
	}
	if !names["npm"] {
		t.Error("npm provider not found in All()")
	}
	if !names["nix"] {
		t.Error("nix provider not found in All()")
	}
}

func TestRegistryRegisterOverwrite(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(&fakeProvider{name: "npm", displayName: "npm v1"})
	r.Register(&fakeProvider{name: "npm", displayName: "npm v2"})

	all := r.All()
	if len(all) != 1 {
		t.Errorf("All() returned %d providers after overwrite, want 1", len(all))
	}
	if all[0].DisplayName() != "npm v2" {
		t.Errorf("DisplayName() = %q, want %q (overwrite failed)", all[0].DisplayName(), "npm v2")
	}
}

func TestRegistryByName(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(&fakeProvider{name: "devenv", displayName: "devenv"})

	tests := []struct {
		name   string
		lookup string
		wantOK bool
	}{
		{"found", "devenv", true},
		{"not found", "npm", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, ok := r.ByName(tt.lookup)
			if ok != tt.wantOK {
				t.Errorf("ByName(%q) ok = %v, want %v", tt.lookup, ok, tt.wantOK)
			}
			if tt.wantOK && p == nil {
				t.Errorf("ByName(%q) returned nil provider", tt.lookup)
			}
			if !tt.wantOK && p != nil {
				t.Errorf("ByName(%q) returned non-nil provider for missing name", tt.lookup)
			}
		})
	}
}

func TestRegistryDetectAll(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(&fakeProvider{name: "npm", detected: true})
	r.Register(&fakeProvider{name: "nix", detected: false})
	r.Register(&fakeProvider{name: "devenv", detected: true})

	detected := r.DetectAll("/project", "/home/user")

	if len(detected) != 2 {
		t.Errorf("DetectAll returned %d providers, want 2", len(detected))
	}

	names := make(map[string]bool)
	for _, p := range detected {
		names[p.Name()] = true
	}
	if !names["npm"] {
		t.Error("npm should be detected")
	}
	if names["nix"] {
		t.Error("nix should not be detected")
	}
	if !names["devenv"] {
		t.Error("devenv should be detected")
	}
}

func TestRegistryDetectAllEmpty(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	detected := r.DetectAll("/project", "/home/user")
	if len(detected) != 0 {
		t.Errorf("DetectAll on empty registry returned %d, want 0", len(detected))
	}
}

func TestRegistryDetectAllNoneMatch(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(&fakeProvider{name: "npm", detected: false})
	r.Register(&fakeProvider{name: "nix", detected: false})

	detected := r.DetectAll("/project", "/home/user")
	if len(detected) != 0 {
		t.Errorf("DetectAll with no matches returned %d, want 0", len(detected))
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	var wg sync.WaitGroup
	wg.Add(3)

	// Concurrent registrations.
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			r.Register(&fakeProvider{name: "provider-a", detected: true})
		}
	}()

	// Concurrent reads via All.
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = r.All()
		}
	}()

	// Concurrent reads via ByName.
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = r.ByName("provider-a")
		}
	}()

	wg.Wait()
}

func TestDefaultRegistrySingleton(t *testing.T) {
	// Not parallel: accesses package-level state.
	r1 := DefaultRegistry()
	r2 := DefaultRegistry()
	if r1 != r2 {
		t.Error("DefaultRegistry should return the same instance")
	}
}
