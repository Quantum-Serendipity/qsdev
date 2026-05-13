package toolreg

import "testing"

func TestNewToolState(t *testing.T) {
	ts := NewToolState()
	if ts.Enabled == nil {
		t.Fatal("expected Enabled map to be initialized")
	}
	if len(ts.Enabled) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(ts.Enabled))
	}
}

func TestToolState_IsEnabled_Default(t *testing.T) {
	ts := NewToolState()
	if ts.IsEnabled("anything") {
		t.Fatal("expected unset tool to not be enabled")
	}
}

func TestToolState_IsEnabled_NilMap(t *testing.T) {
	ts := ToolState{Enabled: nil}
	if ts.IsEnabled("anything") {
		t.Fatal("expected false when Enabled map is nil")
	}
}

func TestToolState_Enable(t *testing.T) {
	ts := NewToolState()
	ts.Enable("my-tool")

	if !ts.IsEnabled("my-tool") {
		t.Fatal("expected tool to be enabled after Enable()")
	}
}

func TestToolState_Enable_NilMap(t *testing.T) {
	ts := ToolState{Enabled: nil}
	ts.Enable("my-tool")

	if ts.Enabled == nil {
		t.Fatal("expected Enable to initialize the map")
	}
	if !ts.IsEnabled("my-tool") {
		t.Fatal("expected tool to be enabled after Enable() on nil map")
	}
}

func TestToolState_Disable(t *testing.T) {
	ts := NewToolState()
	ts.Enable("my-tool")
	ts.Disable("my-tool")

	if ts.IsEnabled("my-tool") {
		t.Fatal("expected tool to be disabled after Disable()")
	}
	// Verify the key exists with value false (not deleted).
	val, exists := ts.Enabled["my-tool"]
	if !exists {
		t.Fatal("expected key to still exist in map after Disable()")
	}
	if val {
		t.Fatal("expected value to be false after Disable()")
	}
}

func TestToolState_Disable_NilMap(t *testing.T) {
	ts := ToolState{Enabled: nil}
	// Should not panic.
	ts.Disable("my-tool")

	// Map stays nil since there's nothing to disable.
	if ts.Enabled != nil {
		t.Fatal("expected nil map to remain nil after Disable()")
	}
}

func TestToolState_Disable_NeverEnabled(t *testing.T) {
	ts := NewToolState()
	ts.Disable("never-enabled")

	if ts.IsEnabled("never-enabled") {
		t.Fatal("expected tool to not be enabled")
	}
}

func TestToolState_MultipleTools(t *testing.T) {
	ts := NewToolState()
	ts.Enable("tool-a")
	ts.Enable("tool-b")
	ts.Enable("tool-c")
	ts.Disable("tool-b")

	if !ts.IsEnabled("tool-a") {
		t.Error("expected tool-a to be enabled")
	}
	if ts.IsEnabled("tool-b") {
		t.Error("expected tool-b to be disabled")
	}
	if !ts.IsEnabled("tool-c") {
		t.Error("expected tool-c to be enabled")
	}
}

func TestToolState_EnableTwice(t *testing.T) {
	ts := NewToolState()
	ts.Enable("tool")
	ts.Enable("tool")

	if !ts.IsEnabled("tool") {
		t.Fatal("expected tool to still be enabled after double Enable()")
	}
}

func TestToolState_DisableTwice(t *testing.T) {
	ts := NewToolState()
	ts.Enable("tool")
	ts.Disable("tool")
	ts.Disable("tool")

	if ts.IsEnabled("tool") {
		t.Fatal("expected tool to still be disabled after double Disable()")
	}
}
