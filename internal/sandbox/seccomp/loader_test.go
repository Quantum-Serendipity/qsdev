package seccomp

import "testing"

func TestLoadFilter_NoFilter(t *testing.T) {
	t.Parallel()
	_, err := LoadFilter()
	if err == nil {
		t.Skip("seccomp filter is available (Nix build), skipping no-filter test")
	}
}

func TestAvailable_ReflectsLoadFilter(t *testing.T) {
	t.Parallel()
	_, err := LoadFilter()
	got := Available()
	want := err == nil
	if got != want {
		t.Errorf("Available() = %v, LoadFilter error = %v", got, err)
	}
}
