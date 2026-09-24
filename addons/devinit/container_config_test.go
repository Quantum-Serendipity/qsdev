package devinit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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

func assertNoCompose(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, container.ComposeFileName)); !os.IsNotExist(err) {
		t.Errorf("expected %s to be absent, stat err = %v", container.ComposeFileName, err)
	}
}

func TestPlanGatewayCompose(t *testing.T) {
	registerCursor(t)
	tests := []struct {
		name     string
		cursor   bool
		answers  types.WizardAnswers
		opts     UpdateOptions
		wantFile bool
		wantHold bool
	}{
		{name: "skip-container holds any existing fragment", cursor: true, opts: UpdateOptions{SkipContainer: true}, wantHold: true},
		{name: "all-native project generates nothing", answers: types.WizardAnswers{ClaudeCode: true}},
		{name: "hookless framework generates the fragment", cursor: true, wantFile: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.cursor {
				writeCursorMarker(t, dir)
			}
			var buf bytes.Buffer
			g := planGatewayCompose(&buf, dir, tt.answers, tt.opts)
			if g.hold != tt.wantHold {
				t.Errorf("hold = %v, want %v", g.hold, tt.wantHold)
			}
			if (g.file != nil) != tt.wantFile {
				t.Fatalf("file = %v, want present=%v", g.file, tt.wantFile)
			}
			if buf.Len() != 0 {
				t.Errorf("unexpected warning output: %q", buf.String())
			}
			assertNoCompose(t, dir) // planning never writes

			var hints bytes.Buffer
			g.printHints(&hints)
			if !tt.wantFile {
				if hints.Len() != 0 {
					t.Errorf("no gateway needed, but hints printed: %q", hints.String())
				}
				return
			}
			f := g.file
			if f.Path != container.ComposeFileName || f.Strategy != types.ThreeWayMerge || f.Owner != "" {
				t.Errorf("file = {Path:%q Strategy:%v Owner:%q}, want {%q ThreeWayMerge \"\"}",
					f.Path, f.Strategy, f.Owner, container.ComposeFileName)
			}
			if strings.Contains(string(f.Content), dir) {
				t.Errorf("committed fragment embeds this checkout's path %q:\n%s", dir, f.Content)
			}
			if !strings.Contains(string(f.Content), ".:"+container.ContainerWorkspace+":ro") {
				t.Errorf("fragment does not mount the project relative to itself:\n%s", f.Content)
			}
			if !strings.Contains(hints.String(), container.DefaultMCPServerName) {
				t.Errorf("hints lack the .mcp.json entry: %q", hints.String())
			}
		})
	}
}

func TestGatewayComposeKeepHeld(t *testing.T) {
	t.Parallel()
	orphans := []FileUpdatePlan{
		{Path: container.ComposeFileName, Action: UpdateActionRemove},
		{Path: "other.txt", Action: UpdateActionRemove},
	}
	tests := []struct {
		name string
		hold bool
		want []string
	}{
		{"held fragment is not an orphan", true, []string{"other.txt"}},
		{"unheld fragment follows orphan cleanup", false, []string{container.ComposeFileName, "other.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := gatewayCompose{hold: tt.hold}.keepHeld(slices.Clone(orphans))
			var paths []string
			for _, fp := range got {
				paths = append(paths, fp.Path)
			}
			if !slices.Equal(paths, tt.want) {
				t.Errorf("paths = %v, want %v", paths, tt.want)
			}
		})
	}
}

// runGatewayUpdate runs `qsdev update` with opts in dir.
func runGatewayUpdate(t *testing.T, dir string, opts UpdateOptions) string {
	t.Helper()
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := runUpdate(cmd, opts); err != nil {
		t.Fatalf("update: %v\n%s", err, buf.String())
	}
	return buf.String()
}

// TestUpdate_GatewayComposeLifecycle is the F064 regression: the gateway
// fragment goes through update's plan and state pipeline, so it is tracked,
// user edits survive later updates, --skip-container leaves it alone, it is
// removed once no framework needs it, and teardown removes it.
func TestUpdate_GatewayComposeLifecycle(t *testing.T) {
	registerCursor(t)
	dir := initLifecycleProject(t)
	writeCursorMarker(t, dir)
	composePath := filepath.Join(dir, container.ComposeFileName)

	// Dry run plans the file without writing it.
	out := runGatewayUpdate(t, dir, UpdateOptions{DryRun: true})
	if !strings.Contains(out, container.ComposeFileName) || !strings.Contains(out, container.DefaultMCPServerName) {
		t.Errorf("dry run does not preview the fragment and hint:\n%s", out)
	}
	assertNoCompose(t, dir)

	// First update creates and tracks it.
	runGatewayUpdate(t, dir, UpdateOptions{})
	fs, ok := loadProjectState(t, dir).Files[container.ComposeFileName]
	if !ok {
		t.Fatalf("%s not recorded in state", container.ComposeFileName)
	}
	if fs.Strategy != types.ThreeWayMerge {
		t.Errorf("recorded strategy = %v, want ThreeWayMerge", fs.Strategy)
	}

	// A user edit survives the next update.
	edited := strings.Replace(readProjectFile(t, dir, container.ComposeFileName),
		fmt.Sprintf("%d:%d", container.DefaultGatewayPort, container.DefaultGatewayPort),
		fmt.Sprintf("9000:%d", container.DefaultGatewayPort), 1)
	edited = "# local port override\n" + edited
	if err := os.WriteFile(composePath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	runGatewayUpdate(t, dir, UpdateOptions{})
	got := readProjectFile(t, dir, container.ComposeFileName)
	for _, want := range []string{"9000:", "# local port override"} {
		if !strings.Contains(got, want) {
			t.Errorf("update lost the user's %q:\n%s", want, got)
		}
	}

	// --skip-container leaves the file and its state entry alone, also once
	// no framework needs the gateway.
	if err := os.RemoveAll(filepath.Join(dir, ".cursor")); err != nil {
		t.Fatal(err)
	}
	runGatewayUpdate(t, dir, UpdateOptions{SkipContainer: true})
	if readProjectFile(t, dir, container.ComposeFileName) != got {
		t.Error("--skip-container changed the fragment")
	}
	if _, ok := loadProjectState(t, dir).Files[container.ComposeFileName]; !ok {
		t.Error("--skip-container untracked the fragment")
	}

	// Once the fragment is back to generated content, it is removed as an
	// orphan when no framework needs the gateway any more.
	writeCursorMarker(t, dir)
	if err := os.Remove(composePath); err != nil {
		t.Fatal(err)
	}
	runGatewayUpdate(t, dir, UpdateOptions{Force: true}) // recreate the deleted file
	if err := os.RemoveAll(filepath.Join(dir, ".cursor")); err != nil {
		t.Fatal(err)
	}
	runGatewayUpdate(t, dir, UpdateOptions{})
	assertNoCompose(t, dir)
	if _, ok := loadProjectState(t, dir).Files[container.ComposeFileName]; ok {
		t.Error("removed fragment is still tracked")
	}
}

// TestUpdate_GatewayComposeRefusesSymlinkEscape verifies the gateway compose
// fragment is not written through a committed symlink that resolves outside
// the project (F455): the fragment goes through the update pipeline's
// root-confined writer.
func TestUpdate_GatewayComposeRefusesSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	registerCursor(t)
	dir := initLifecycleProject(t)
	writeCursorMarker(t, dir)
	outside := filepath.Join(t.TempDir(), "compose.yaml")
	if err := os.WriteFile(outside, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, container.ComposeFileName)); err != nil {
		t.Fatal(err)
	}

	t.Setenv("QSDEV_SKIP_SETUP", "1")
	t.Chdir(dir)
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	_ = runUpdate(cmd, UpdateOptions{Force: true}) // the refused write may fail the update

	if data, _ := os.ReadFile(outside); string(data) != "services: {}\n" {
		t.Errorf("file outside the project was rewritten: %q\n%s", data, buf.String())
	}
}

// TestTeardown_RemovesGatewayCompose checks teardown removes the tracked,
// unmodified gateway fragment.
func TestTeardown_RemovesGatewayCompose(t *testing.T) {
	registerCursor(t)
	dir := initLifecycleProject(t)
	writeCursorMarker(t, dir)
	runGatewayUpdate(t, dir, UpdateOptions{})
	if _, err := os.Stat(filepath.Join(dir, container.ComposeFileName)); err != nil {
		t.Fatalf("fragment not generated: %v", err)
	}
	if out, err := runLifecycleCmd(t, dir, teardownCmd(), "--force"); err != nil {
		t.Fatalf("teardown: %v\n%s", err, out)
	}
	assertNoCompose(t, dir)
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
