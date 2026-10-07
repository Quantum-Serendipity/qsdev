//go:build unix

package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestRunSingleCheck_StaleSelfOnPath is the E3 case: a CLI printing an old
// version first on PATH, where the selfprotect hook would run it, fails its
// required check instead of passing a lookup.
func TestRunSingleCheck_StaleSelfOnPath(t *testing.T) {
	app := branding.Get().AppName
	bin := t.TempDir()
	script := "#!/bin/sh\necho '" + app + " version 0.1.0'\n"
	if err := os.WriteFile(filepath.Join(bin, app), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Chdir(t.TempDir())

	checks := withSelfCheck(requireBinaries(nil, []string{app}, HookRequiredBy, "linux", ""), ">= 0.7.10", "/opt/q/"+app, "linux", "")
	st := runSingleCheck(context.Background(), checks[0], &sysinfo.OSInfo{OS: "linux"})
	if !st.Installed || st.Version != "0.1.0" || st.VersionOK || !st.NeedsSetup() {
		t.Errorf("status = %+v; want the stale CLI found at version 0.1.0 and failing the constraint", st)
	}
}
