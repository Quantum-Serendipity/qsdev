package surgery

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONAddMCPServer_EmptyContent(t *testing.T) {
	entry := json.RawMessage(`{"command": "npx", "args": ["server"]}`)

	result, err := JSONAddMCPServer(nil, "my-server", entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)

	// Should contain the server.
	if !strings.Contains(got, `"my-server"`) {
		t.Error("server name should be present")
	}
	if !strings.Contains(got, `"mcpServers"`) {
		t.Error("mcpServers key should be present")
	}

	// Should be valid JSON.
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Should end with a trailing newline.
	if !strings.HasSuffix(got, "\n") {
		t.Error("result should end with a newline")
	}
}

func TestJSONAddMCPServer_EmptyWhitespaceContent(t *testing.T) {
	entry := json.RawMessage(`{"command": "test"}`)

	result, err := JSONAddMCPServer([]byte("  \n  "), "my-server", entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := doc["mcpServers"]; !ok {
		t.Error("mcpServers key should exist")
	}
}

func TestJSONAddMCPServer_ExistingContent(t *testing.T) {
	existing := []byte(`{
  "mcpServers": {
    "existing-server": {
      "command": "npx",
      "args": ["old-server"]
    }
  }
}`)
	entry := json.RawMessage(`{"command": "npx", "args": ["new-server"]}`)

	result, err := JSONAddMCPServer(existing, "new-server", entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)

	// Both servers should be present.
	if !strings.Contains(got, `"existing-server"`) {
		t.Error("existing server should be preserved")
	}
	if !strings.Contains(got, `"new-server"`) {
		t.Error("new server should be present")
	}

	// Validate JSON structure.
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if len(doc.MCPServers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(doc.MCPServers))
	}
}

func TestJSONAddMCPServer_ReplacesExisting(t *testing.T) {
	existing := []byte(`{
  "mcpServers": {
    "my-server": {
      "command": "old-command"
    }
  }
}`)
	entry := json.RawMessage(`{"command": "new-command", "args": ["--flag"]}`)

	result, err := JSONAddMCPServer(existing, "my-server", entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if len(doc.MCPServers) != 1 {
		t.Errorf("expected 1 server after replacement, got %d", len(doc.MCPServers))
	}

	got := string(result)
	if strings.Contains(got, "old-command") {
		t.Error("old entry should have been replaced")
	}
	if !strings.Contains(got, "new-command") {
		t.Error("new entry should be present")
	}
}

func TestJSONAddMCPServer_PreservesOtherKeys(t *testing.T) {
	existing := []byte(`{
  "customKey": "customValue",
  "mcpServers": {}
}`)
	entry := json.RawMessage(`{"command": "test"}`)

	result, err := JSONAddMCPServer(existing, "server", entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := doc["customKey"]; !ok {
		t.Error("customKey should be preserved")
	}
}

func TestJSONAddMCPServer_InvalidExistingJSON(t *testing.T) {
	existing := []byte(`{invalid json}`)
	entry := json.RawMessage(`{"command": "test"}`)

	_, err := JSONAddMCPServer(existing, "server", entry)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONRemoveMCPServer_RemovesExisting(t *testing.T) {
	existing := []byte(`{
  "mcpServers": {
    "server-a": {"command": "a"},
    "server-b": {"command": "b"}
  }
}`)

	result, err := JSONRemoveMCPServer(existing, "server-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if strings.Contains(got, "server-a") {
		t.Error("server-a should have been removed")
	}
	if !strings.Contains(got, "server-b") {
		t.Error("server-b should be preserved")
	}

	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if len(doc.MCPServers) != 1 {
		t.Errorf("expected 1 server, got %d", len(doc.MCPServers))
	}
}

func TestJSONRemoveMCPServer_MissingServerReturnsUnchanged(t *testing.T) {
	existing := []byte(`{
  "mcpServers": {
    "server-a": {"command": "a"}
  }
}`)

	result, err := JSONRemoveMCPServer(existing, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(existing) {
		t.Error("content should be unchanged when removing nonexistent server")
	}
}

func TestJSONRemoveMCPServer_NoMCPServersKey(t *testing.T) {
	existing := []byte(`{
  "otherKey": "value"
}`)

	result, err := JSONRemoveMCPServer(existing, "server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(existing) {
		t.Error("content should be unchanged when mcpServers key is missing")
	}
}

func TestJSONRemoveMCPServer_LastServerProducesEmptyObject(t *testing.T) {
	existing := []byte(`{
  "mcpServers": {
    "only-server": {"command": "test"}
  }
}`)

	result, err := JSONRemoveMCPServer(existing, "only-server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(string(result))
	if got != "{}" {
		t.Errorf("expected empty object {}, got: %s", got)
	}
}

func TestJSONRemoveMCPServer_EmptyContent(t *testing.T) {
	result, err := JSONRemoveMCPServer(nil, "server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for empty input, got: %s", string(result))
	}
}

func TestJSONRemoveMCPServer_EmptyByteSlice(t *testing.T) {
	result, err := JSONRemoveMCPServer([]byte{}, "server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got: %s", string(result))
	}
}

func TestJSONHasMCPServer_True(t *testing.T) {
	content := []byte(`{
  "mcpServers": {
    "my-server": {"command": "test"}
  }
}`)

	if !JSONHasMCPServer(content, "my-server") {
		t.Error("expected HasMCPServer to return true")
	}
}

func TestJSONHasMCPServer_False(t *testing.T) {
	content := []byte(`{
  "mcpServers": {
    "other-server": {"command": "test"}
  }
}`)

	if JSONHasMCPServer(content, "my-server") {
		t.Error("expected HasMCPServer to return false for missing server")
	}
}

func TestJSONHasMCPServer_EmptyContent(t *testing.T) {
	if JSONHasMCPServer(nil, "server") {
		t.Error("expected false for nil content")
	}
	if JSONHasMCPServer([]byte{}, "server") {
		t.Error("expected false for empty content")
	}
}

func TestJSONHasMCPServer_NoMCPServersKey(t *testing.T) {
	content := []byte(`{"otherKey": "value"}`)
	if JSONHasMCPServer(content, "server") {
		t.Error("expected false when mcpServers key is absent")
	}
}

func TestJSONHasMCPServer_InvalidJSON(t *testing.T) {
	content := []byte(`{invalid}`)
	if JSONHasMCPServer(content, "server") {
		t.Error("expected false for invalid JSON")
	}
}

func TestJSONAddMCPServer_PreservesOtherKeysOnRemoveLastServer(t *testing.T) {
	existing := []byte(`{
  "customKey": "value",
  "mcpServers": {
    "only-server": {"command": "test"}
  }
}`)

	result, err := JSONRemoveMCPServer(existing, "only-server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := doc["customKey"]; !ok {
		t.Error("customKey should be preserved after removing last server")
	}
	if _, ok := doc["mcpServers"]; ok {
		t.Error("mcpServers key should be removed when empty")
	}
}

func TestJSONAddMCPServer_DeterministicSortedOutput(t *testing.T) {
	var content []byte
	var err error

	content, err = JSONAddMCPServer(content, "charlie", json.RawMessage(`{"command":"c"}`))
	if err != nil {
		t.Fatalf("adding charlie: %v", err)
	}
	content, err = JSONAddMCPServer(content, "alpha", json.RawMessage(`{"command":"a"}`))
	if err != nil {
		t.Fatalf("adding alpha: %v", err)
	}
	content, err = JSONAddMCPServer(content, "bravo", json.RawMessage(`{"command":"b"}`))
	if err != nil {
		t.Fatalf("adding bravo: %v", err)
	}

	got := string(content)

	alphaIdx := strings.Index(got, `"alpha"`)
	bravoIdx := strings.Index(got, `"bravo"`)
	charlieIdx := strings.Index(got, `"charlie"`)

	if alphaIdx == -1 || bravoIdx == -1 || charlieIdx == -1 {
		t.Fatalf("expected all three servers in output, got:\n%s", got)
	}

	if alphaIdx >= bravoIdx || bravoIdx >= charlieIdx {
		t.Errorf("expected alphabetical order (alpha < bravo < charlie), got positions alpha=%d bravo=%d charlie=%d\noutput:\n%s",
			alphaIdx, bravoIdx, charlieIdx, got)
	}

	for i := 0; i < 20; i++ {
		var repeat []byte
		repeat, err = JSONAddMCPServer(nil, "charlie", json.RawMessage(`{"command":"c"}`))
		if err != nil {
			t.Fatalf("iteration %d charlie: %v", i, err)
		}
		repeat, err = JSONAddMCPServer(repeat, "alpha", json.RawMessage(`{"command":"a"}`))
		if err != nil {
			t.Fatalf("iteration %d alpha: %v", i, err)
		}
		repeat, err = JSONAddMCPServer(repeat, "bravo", json.RawMessage(`{"command":"b"}`))
		if err != nil {
			t.Fatalf("iteration %d bravo: %v", i, err)
		}

		if string(repeat) != got {
			t.Errorf("iteration %d produced different output:\nexpected:\n%s\ngot:\n%s", i, got, string(repeat))
			break
		}
	}
}
