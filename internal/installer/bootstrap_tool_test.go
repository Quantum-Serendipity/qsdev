package installer_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

// TestBootstrapToolInstallCmd_MissingEntry covers F110: a tool the catalog
// does not pin is refused rather than installed at whatever release the
// registry serves.
func TestBootstrapToolInstallCmd_MissingEntry(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	_, err = installer.BootstrapToolInstallCmd(cat, "no-such-tool", time.Now())
	if !errors.Is(err, installer.ErrBootstrapPin) {
		t.Fatalf("error = %v, want %v", err, installer.ErrBootstrapPin)
	}
}
