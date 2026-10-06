//go:build unix

package devinit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestCheckPrerequisites_BelowFloorIsMissing: the init/join gate in front of
// auto-setup uses doctor's floors, so a devenv below its minimum counts as
// missing and is reported as below it, not OK.
func TestCheckPrerequisites_BelowFloorIsMissing(t *testing.T) {
	bin := t.TempDir()
	for name, out := range map[string]string{
		"nix":    "nix (Nix) 2.28.0",
		"devenv": "devenv 1.4.1 (x86_64-linux)",
		"direnv": "2.34.0",
		"git":    "git version 2.47.1",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\necho '"+out+"'\n"), 0o755); err != nil { //nolint:gosec // test executable
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)

	result := CheckPrerequisites(context.Background())
	if !result.HasMissing() {
		t.Errorf("HasMissing() = false with devenv 1.4.1 below the %s floor: %+v", types.MinDevenv, result.Tools)
	}
	var buf bytes.Buffer
	result.PrintReport(&buf)
	if want := "devenv     below minimum " + types.MinDevenv + " (1.4.1)"; !strings.Contains(buf.String(), want) {
		t.Errorf("PrintReport missing %q:\n%s", want, buf.String())
	}
}
