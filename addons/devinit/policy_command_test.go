package devinit

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
)

// TestRenderPolicySARIFEmitsSARIF is the red→green guard for F-CAP-21.3-1 at the
// command surface: `policy check --sarif` must emit a SARIF 2.1.0 document, not
// the bare PolicyPosture JSON it used to marshal.
func TestRenderPolicySARIFEmitsSARIF(t *testing.T) {
	t.Parallel()

	posture := &sarif.PolicyPosture{
		RulesActive:      3,
		RulesTotal:       3,
		MonitorModeCount: 1,
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	if err := renderPolicySARIF(cmd, posture, ""); err != nil {
		t.Fatalf("renderPolicySARIF: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if parsed["$schema"] != sarif.SchemaURL {
		t.Errorf("$schema = %v, want %q", parsed["$schema"], sarif.SchemaURL)
	}
	if parsed["version"] != "2.1.0" {
		t.Errorf("version = %v, want \"2.1.0\"", parsed["version"])
	}
	runs, ok := parsed["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("expected one run, got %v", parsed["runs"])
	}
	run := runs[0].(map[string]any)
	tool, ok := run["tool"].(map[string]any)
	if !ok {
		t.Fatalf("run missing tool")
	}
	if _, ok := tool["driver"].(map[string]any); !ok {
		t.Fatalf("tool missing driver")
	}
	if _, ok := run["results"].([]any); !ok {
		t.Fatalf("run missing results array")
	}
}
