package sandbox

import (
	"context"
	"errors"
	"testing"
)

type mockBackend struct {
	name         string
	tier         DegradationTier
	availableErr error
	runResult    *SandboxResult
	runErr       error
}

func (m *mockBackend) Name() string          { return m.name }
func (m *mockBackend) Available() error      { return m.availableErr }
func (m *mockBackend) Tier() DegradationTier { return m.tier }
func (m *mockBackend) RunHook(_ context.Context, _ *SandboxConfig) (*SandboxResult, error) {
	return m.runResult, m.runErr
}

var _ SandboxBackend = (*mockBackend)(nil)

func TestBackendRegistry_Select_StrongestAvailable(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&mockBackend{name: "weak", tier: TierSystemdRun})
	r.Register(&mockBackend{name: "strong", tier: TierFull})
	r.Register(&mockBackend{name: "medium", tier: TierBwrapWithoutLandlock})

	b, err := r.Select()
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if b.Name() != "strong" {
		t.Errorf("Select() = %q, want %q", b.Name(), "strong")
	}
}

func TestBackendRegistry_Select_SkipsUnavailable(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&mockBackend{
		name:         "unavailable",
		tier:         TierFull,
		availableErr: errors.New("missing bwrap"),
	})
	r.Register(&mockBackend{name: "available", tier: TierSystemdRun})

	b, err := r.Select()
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if b.Name() != "available" {
		t.Errorf("Select() = %q, want %q", b.Name(), "available")
	}
}

func TestBackendRegistry_Select_FallsBackToUnsandboxed(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&mockBackend{
		name:         "unavailable",
		tier:         TierFull,
		availableErr: errors.New("not supported"),
	})

	b, err := r.Select()
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if b.Tier() != TierUnsandboxed {
		t.Errorf("Select() tier = %v, want %v", b.Tier(), TierUnsandboxed)
	}
}

func TestBackendRegistry_Select_EmptyRegistry(t *testing.T) {
	t.Parallel()
	r := NewRegistry()

	b, err := r.Select()
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if b.Tier() != TierUnsandboxed {
		t.Errorf("Select() tier = %v, want %v", b.Tier(), TierUnsandboxed)
	}
}

func TestBackendRegistry_SelectByName(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&mockBackend{name: "bubblewrap", tier: TierFull})
	r.Register(&mockBackend{name: "systemd-run", tier: TierSystemdRun})

	b, err := r.SelectByName("bubblewrap")
	if err != nil {
		t.Fatalf("SelectByName() error = %v", err)
	}
	if b.Name() != "bubblewrap" {
		t.Errorf("SelectByName() = %q, want %q", b.Name(), "bubblewrap")
	}
}

func TestBackendRegistry_SelectByName_NotRegistered(t *testing.T) {
	t.Parallel()
	r := NewRegistry()

	_, err := r.SelectByName("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered backend")
	}
}

func TestBackendRegistry_SelectByName_Unavailable(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&mockBackend{
		name:         "broken",
		tier:         TierFull,
		availableErr: errors.New("kernel too old"),
	})

	_, err := r.SelectByName("broken")
	if err == nil {
		t.Fatal("expected error for unavailable backend")
	}
}

func TestBackendRegistry_List(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&mockBackend{name: "a", tier: TierFull})
	r.Register(&mockBackend{
		name:         "b",
		tier:         TierSystemdRun,
		availableErr: errors.New("missing"),
	})

	statuses := r.List()
	if len(statuses) != 2 {
		t.Fatalf("List() returned %d, want 2", len(statuses))
	}
	if !statuses[0].Available {
		t.Error("first backend should be available")
	}
	if statuses[1].Available {
		t.Error("second backend should not be available")
	}
	if statuses[1].Error == nil {
		t.Error("second backend should have error")
	}
}
