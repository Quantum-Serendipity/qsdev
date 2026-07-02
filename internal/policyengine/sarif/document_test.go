package sarif

import (
	"encoding/json"
	"testing"
)

// TestBuildLogEmitsValidSARIF is the red→green guard for F-CAP-21.3-1: the
// --sarif path must emit a real SARIF 2.1.0 document ($schema, version "2.1.0",
// runs[].tool.driver.rules, results[]) rather than a bare PolicyPosture struct.
func TestBuildLogEmitsValidSARIF(t *testing.T) {
	t.Parallel()

	findings := []SarifResult{
		{
			RuleID:           "qsdev/policy/MCP-002",
			Level:            "error",
			Message:          "confused deputy blocked",
			ArtifactURI:      ".mcp.json",
			SecuritySeverity: 9.0,
			PartialFingerprints: map[string]string{
				"ruleId": "qsdev/policy/MCP-002",
			},
		},
	}

	log := BuildLog("qsdev", "1.2.3", "https://github.com/Quantum-Serendipity/qsdev", findings)

	// Round-trip through JSON so we validate the serialized shape a SARIF
	// consumer (GitHub code scanning) would actually ingest.
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshaling SARIF: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshaling SARIF: %v", err)
	}

	if parsed["$schema"] != SchemaURL {
		t.Errorf("$schema = %v, want %q", parsed["$schema"], SchemaURL)
	}
	if parsed["version"] != "2.1.0" {
		t.Errorf("version = %v, want \"2.1.0\"", parsed["version"])
	}

	runs, ok := parsed["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("expected exactly one run, got %v", parsed["runs"])
	}

	run := runs[0].(map[string]any)
	tool, ok := run["tool"].(map[string]any)
	if !ok {
		t.Fatalf("run missing tool: %v", run)
	}
	driver, ok := tool["driver"].(map[string]any)
	if !ok {
		t.Fatalf("tool missing driver: %v", tool)
	}
	if driver["name"] != "qsdev" {
		t.Errorf("driver.name = %v, want \"qsdev\"", driver["name"])
	}
	rules, ok := driver["rules"].([]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("driver.rules empty or missing: %v", driver["rules"])
	}
	if len(rules) != len(AllRules) {
		t.Errorf("driver.rules count = %d, want %d (full catalog)", len(rules), len(AllRules))
	}

	results, ok := run["results"].([]any)
	if !ok {
		t.Fatalf("run missing results array: %v", run)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	res := results[0].(map[string]any)
	if res["ruleId"] != "qsdev/policy/MCP-002" {
		t.Errorf("result.ruleId = %v, want qsdev/policy/MCP-002", res["ruleId"])
	}
	if res["level"] != "error" {
		t.Errorf("result.level = %v, want error", res["level"])
	}
	msg, ok := res["message"].(map[string]any)
	if !ok || msg["text"] != "confused deputy blocked" {
		t.Errorf("result.message.text = %v, want \"confused deputy blocked\"", res["message"])
	}
}

// TestBuildLogEmptyFindingsStillValid confirms that with no findings the results
// array is present (non-nil) so the document still conforms to SARIF 2.1.0.
func TestBuildLogEmptyFindingsStillValid(t *testing.T) {
	t.Parallel()

	log := BuildLog("qsdev", "", "https://example.com", nil)

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshaling SARIF: %v", err)
	}
	if got := log.Runs[0].Results; got == nil {
		t.Error("results must be a non-nil array even with no findings")
	}
	// The serialized results must be [], not null.
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	run := parsed["runs"].([]any)[0].(map[string]any)
	if _, ok := run["results"].([]any); !ok {
		t.Errorf("results serialized as %v, want []", run["results"])
	}
}
