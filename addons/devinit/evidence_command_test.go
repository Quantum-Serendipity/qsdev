package devinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/evidence"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// executeEvidenceCmd runs the evidence command in the given directory and
// returns its stdout/stderr buffer and any error.
func executeEvidenceCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}

	cmd := evidenceCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// writeInitialized creates a minimally-initialized project (a .qsdev.yaml) so
// posture.Assess treats the directory as a qsdev project.
func writeInitialized(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte("version: 1\ntier: standard\n"), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}
}

// writeWeakConfig drops the exact artifacts from the E3 repro: an empty `{}`
// .claude/settings.json and a one-line comment devenv.nix — files that are
// PRESENT but enforce nothing.
func writeWeakConfig(t *testing.T, dir string) {
	t.Helper()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("writing settings.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), []byte("# devenv\n"), 0o644); err != nil {
		t.Fatalf("writing devenv.nix: %v", err)
	}
}

func controlStatus(t *testing.T, report *evidence.EvidenceReport, id string) evidence.ControlStatus {
	t.Helper()
	for _, cm := range report.Controls {
		if cm.ControlID == id {
			return cm.Status
		}
	}
	t.Fatalf("control %q not found in evidence report", id)
	return ""
}

func parseEvidenceJSON(t *testing.T, out string) *evidence.EvidenceReport {
	t.Helper()
	var er evidence.EvidenceReport
	if err := json.Unmarshal([]byte(out), &er); err != nil {
		t.Fatalf("unmarshaling evidence JSON: %v\noutput:\n%s", err, out)
	}
	return &er
}

// TestEvidenceCmd_PresentButUnenforcedNotAddressed is the E3 regression test.
//
// A project that merely CONTAINS an empty `{}` .claude/settings.json and a
// one-line devenv.nix (no tools enabled, no enforcement) must NOT have SOC2
// CC6.6 (System Boundary Protection) or CC8.2 (Configuration Management)
// reported as "Addressed". The old presence-based assessment marked both
// Addressed from file existence alone; the posture-driven assessment does not.
func TestEvidenceCmd_PresentButUnenforcedNotAddressed(t *testing.T) {
	dir := t.TempDir()
	writeInitialized(t, dir)
	writeWeakConfig(t, dir)

	out, err := executeEvidenceCmd(t, dir, "--framework", "soc2", "--format", "json")
	if err != nil {
		t.Fatalf("evidence command failed: %v\noutput:\n%s", err, out)
	}

	report := parseEvidenceJSON(t, out)

	for _, id := range []string{"CC6.6", "CC8.2"} {
		if got := controlStatus(t, report, id); got == evidence.StatusAddressed {
			t.Errorf("control %s reported %q for a present-but-unenforced config; want anything other than %q",
				id, got, evidence.StatusAddressed)
		}
	}

	// Concretely, with no enforcement these controls are not-addressed.
	if got := controlStatus(t, report, "CC6.6"); got != evidence.StatusNotAddressed {
		t.Errorf("CC6.6 = %q, want %q", got, evidence.StatusNotAddressed)
	}
	if got := controlStatus(t, report, "CC8.2"); got != evidence.StatusNotAddressed {
		t.Errorf("CC8.2 = %q, want %q", got, evidence.StatusNotAddressed)
	}
}

// TestEvidenceCmd_ExactE3ReproUninitialized covers the literal E3 directory:
// only settings.json + devenv.nix, with NO qsdev initialization. Like
// `qsdev status`, the evidence command refuses to grade an uninitialized
// project, so no control can be reported as "Addressed".
func TestEvidenceCmd_ExactE3ReproUninitialized(t *testing.T) {
	dir := t.TempDir()
	writeWeakConfig(t, dir) // no writeInitialized: no .qsdev.yaml, no state

	out, err := executeEvidenceCmd(t, dir, "--framework", "soc2", "--format", "json")
	if err == nil {
		t.Fatalf("expected an error for an uninitialized project, got nil\noutput:\n%s", out)
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != exitNotInitialized {
		t.Errorf("expected ExitError code %d, got %v", exitNotInitialized, err)
	}
	if bytes.Contains([]byte(out), []byte("Addressed")) {
		t.Errorf("uninitialized project must not emit any 'Addressed' status; output:\n%s", out)
	}
}

// TestEvidenceCmd_ParityWithPostureAssess kills F-CAP-28.4-3: the evidence
// command must use the same posture assessment as `qsdev status`. Every defense
// layer status embedded in (and derived by) the evidence report must equal the
// status produced by an independent posture.Assess call on the same project.
func TestEvidenceCmd_ParityWithPostureAssess(t *testing.T) {
	dir := t.TempDir()
	writeInitialized(t, dir)
	writeWeakConfig(t, dir)

	out, err := executeEvidenceCmd(t, dir, "--framework", "soc2", "--format", "json")
	if err != nil {
		t.Fatalf("evidence command failed: %v\noutput:\n%s", err, out)
	}
	report := parseEvidenceJSON(t, out)

	assessed, err := posture.Assess(dir, posture.AssessOptions{})
	if err != nil {
		t.Fatalf("posture.Assess failed: %v", err)
	}

	want := make(map[string]string, len(assessed.Defense.Layers))
	for _, l := range assessed.Defense.Layers {
		want[l.Name] = string(l.Status)
	}

	// The embedded posture report must match posture.Assess layer-for-layer.
	if report.Posture == nil {
		t.Fatal("evidence report is missing its embedded posture report")
	}
	if len(report.Posture.Defense.Layers) != len(want) {
		t.Fatalf("embedded posture has %d layers, posture.Assess has %d",
			len(report.Posture.Defense.Layers), len(want))
	}
	for _, l := range report.Posture.Defense.Layers {
		if got := string(l.Status); got != want[l.Name] {
			t.Errorf("layer %q: evidence posture status %q != posture.Assess status %q",
				l.Name, got, want[l.Name])
		}
	}

	// Each control's mapped-layer statuses must also equal posture.Assess.
	for _, cm := range report.Controls {
		for _, le := range cm.GdevLayers {
			expected, ok := want[le.LayerName]
			if !ok {
				t.Errorf("control %s references unknown layer %q", cm.ControlID, le.LayerName)
				continue
			}
			if le.Status != expected {
				t.Errorf("control %s layer %q: evidence status %q != posture.Assess status %q",
					cm.ControlID, le.LayerName, le.Status, expected)
			}
		}
	}
}

// TestEvidenceCmd_EnforcedLayerAddressed guards against a degenerate fix that
// always reports not-addressed. When a tool is actually enabled in project
// state (attach-guard), the layer it enforces (install-script-blocking) is
// enabled and its control (CC6.6) is legitimately Addressed.
func TestEvidenceCmd_EnforcedLayerAddressed(t *testing.T) {
	dir := t.TempDir()
	writeInitialized(t, dir)

	// Persist a real init-state manifest that enables attach-guard.
	st := types.GeneratedState{
		QsdevVersion: "0.8.0",
		EnabledTools: map[string]bool{"attach-guard": true},
		Files:        map[string]types.FileState{},
	}
	data, err := yaml.Marshal(&st)
	if err != nil {
		t.Fatalf("marshaling state: %v", err)
	}
	stateDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, ".qsdev-init-state.yaml"), data, 0o644); err != nil {
		t.Fatalf("writing state: %v", err)
	}

	out, err := executeEvidenceCmd(t, dir, "--framework", "soc2", "--format", "json")
	if err != nil {
		t.Fatalf("evidence command failed: %v\noutput:\n%s", err, out)
	}
	report := parseEvidenceJSON(t, out)

	if got := controlStatus(t, report, "CC6.6"); got != evidence.StatusAddressed {
		t.Errorf("CC6.6 = %q with attach-guard enabled, want %q", got, evidence.StatusAddressed)
	}
}
