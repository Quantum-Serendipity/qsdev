package state

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOrgOverlayRoundTrip pins the record of the approved org overlay: none
// before it is saved, the saved path (also "" for no overlay) after, and an
// error for a record that does not parse.
func TestOrgOverlayRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if p, ok, err := LoadOrgOverlay(root); err != nil || ok || p != "" {
		t.Fatalf("LoadOrgOverlay before saving = %q, %v, %v; want none", p, ok, err)
	}
	for _, want := range []string{"/etc/org/defaults.yaml", ""} {
		if err := SaveOrgOverlay(root, want); err != nil {
			t.Fatal(err)
		}
		if p, ok, err := LoadOrgOverlay(root); err != nil || !ok || p != want {
			t.Errorf("LoadOrgOverlay = %q, %v, %v; want %q recorded", p, ok, err, want)
		}
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(OrgOverlayFile())), []byte("path: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadOrgOverlay(root); err == nil {
		t.Error("LoadOrgOverlay of a malformed record succeeded, want an error")
	}
}
