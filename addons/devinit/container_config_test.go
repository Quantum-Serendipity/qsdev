package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// registerCursorOnce registers the Cursor adapter into the shared registry a
// single time for this test binary. Cursor is a gateway-tier framework, so it is
// what lets these tests reach the NeedsGateway code path. It only Applies when the
// project carries a .cursor/rules marker, so marker-less projects are unaffected.
var registerCursorOnce sync.Once

func registerCursor(t *testing.T) {
	t.Helper()
	registerCursorOnce.Do(func() {
		if err := spi.DefaultRegistry().Register(cursor.New()); err != nil {
			t.Fatalf("register cursor adapter: %v", err)
		}
	})
}

// writeCursorMarker creates the .cursor/rules marker directory so cursor.Applies
// reports true for dir.
func writeCursorMarker(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir cursor marker: %v", err)
	}
}

// captureCmd returns a cobra.Command whose stdout and stderr are captured in one
// buffer (combined is sufficient for substring assertions).
func captureCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	return cmd, &buf
}

func assertNoCompose(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, gatewayComposePath)); !os.IsNotExist(err) {
		t.Errorf("expected %s to be absent, stat err = %v", gatewayComposePath, err)
	}
}

func TestMaybeGenerateContainerConfig_SkipContainer(t *testing.T) {
	dir := t.TempDir()
	cmd, buf := captureCmd()
	maybeGenerateContainerConfig(cmd, dir, types.WizardAnswers{}, UpdateOptions{SkipContainer: true})
	if buf.Len() != 0 {
		t.Errorf("--skip-container should produce no output, got: %q", buf.String())
	}
	assertNoCompose(t, dir)
}

func TestMaybeGenerateContainerConfig_NoGatewayNeeded(t *testing.T) {
	// Claude Code is a native-hook (TierHook) framework, so no gateway is needed:
	// the integration must stay silent and write nothing.
	dir := t.TempDir()
	cmd, buf := captureCmd()
	maybeGenerateContainerConfig(cmd, dir, types.WizardAnswers{ClaudeCode: true}, UpdateOptions{})
	if buf.Len() != 0 {
		t.Errorf("an all-native project should stay silent, got: %q", buf.String())
	}
	assertNoCompose(t, dir)
}

func TestMaybeGenerateContainerConfig_DryRun(t *testing.T) {
	registerCursor(t)
	dir := t.TempDir()
	writeCursorMarker(t, dir)

	cmd, buf := captureCmd()
	maybeGenerateContainerConfig(cmd, dir, types.WizardAnswers{}, UpdateOptions{DryRun: true})

	out := buf.String()
	if !strings.Contains(out, "[dry-run] would write "+gatewayComposePath) {
		t.Errorf("dry-run should announce the intended write, got: %q", out)
	}
	assertNoCompose(t, dir) // dry-run must not actually write
}

func TestMaybeGenerateContainerConfig_WritesCompose(t *testing.T) {
	registerCursor(t)
	dir := t.TempDir()
	writeCursorMarker(t, dir)

	cmd, buf := captureCmd()
	maybeGenerateContainerConfig(cmd, dir, types.WizardAnswers{}, UpdateOptions{})

	if out := buf.String(); !strings.Contains(out, "wrote "+gatewayComposePath) {
		t.Errorf("expected a 'wrote' message, got: %q", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, gatewayComposePath))
	if err != nil {
		t.Fatalf("compose file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("compose file is empty")
	}
}

func TestDetectGatewayProfiles(t *testing.T) {
	registerCursor(t)

	assertProfileIDs := func(t *testing.T, profiles []container.FrameworkProfile, want ...aiframework.FrameworkID) {
		t.Helper()
		got := make(map[aiframework.FrameworkID]bool)
		for _, p := range profiles {
			got[p.ID] = true
		}
		if len(got) != len(want) {
			t.Errorf("profile count = %d, want %d (got %v)", len(got), len(want), profiles)
		}
		for _, id := range want {
			if !got[id] {
				t.Errorf("missing profile %q (got %v)", id, profiles)
			}
		}
	}

	t.Run("answers contribute the recorded framework", func(t *testing.T) {
		dir := t.TempDir() // no markers
		assertProfileIDs(t, detectGatewayProfiles(dir, types.WizardAnswers{ClaudeCode: true}), aiframework.ClaudeCode)
	})

	t.Run("no frameworks yields no profiles", func(t *testing.T) {
		dir := t.TempDir()
		if got := detectGatewayProfiles(dir, types.WizardAnswers{}); len(got) != 0 {
			t.Errorf("expected no profiles, got %v", got)
		}
	})

	t.Run("registry detects a framework via its project marker", func(t *testing.T) {
		dir := t.TempDir()
		writeCursorMarker(t, dir)
		assertProfileIDs(t, detectGatewayProfiles(dir, types.WizardAnswers{}), aiframework.Cursor)
	})

	t.Run("registry and answers combine without duplicates", func(t *testing.T) {
		dir := t.TempDir()
		writeCursorMarker(t, dir)
		assertProfileIDs(t,
			detectGatewayProfiles(dir, types.WizardAnswers{ClaudeCode: true}),
			aiframework.Cursor, aiframework.ClaudeCode)
	})
}
