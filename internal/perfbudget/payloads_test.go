package perfbudget

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
)

// hookDoc is the part of a hook input document the corpus tests read.
type hookDoc struct {
	Event     string         `json:"hook_event_name"`
	Tool      string         `json:"tool_name"`
	Cwd       string         `json:"cwd"`
	Input     map[string]any `json:"tool_input"`
	Response  map[string]any `json:"tool_response"`
	SessionID string         `json:"session_id"`
}

// corpusDocs decodes every payload's document for project dir, by name.
func corpusDocs(t *testing.T, dir string) map[string]hookDoc {
	t.Helper()
	docs := map[string]hookDoc{}
	for _, p := range Payloads() {
		if p.Name == "" || p.Tool == "" || p.Event == "" {
			t.Errorf("payload %+v lacks a name, tool or event", p)
		}
		if _, dup := docs[p.Name]; dup {
			t.Errorf("duplicate payload %q", p.Name)
		}
		data, err := p.JSON(dir)
		if err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
		var d hookDoc
		if err := json.Unmarshal(data, &d); err != nil {
			t.Fatalf("%s: not valid JSON: %v", p.Name, err)
		}
		if d.Event != p.Event || d.Tool != p.Tool || d.Cwd != dir || d.Input == nil || d.SessionID == "" {
			t.Errorf("%s: document %+v does not match the payload", p.Name, d)
		}
		if p.Event == eventPostToolUse && d.Response == nil {
			t.Errorf("%s: PostToolUse payload has no tool_response", p.Name)
		}
		if !maps.EqualFunc(d.Input, p.Input(dir), func(a, b any) bool { return a == b }) {
			t.Errorf("%s: tool_input %v is not Input(dir) %v", p.Name, d.Input, p.Input(dir))
		}
		docs[p.Name] = d
	}
	return docs
}

func TestPayloads(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	docs := corpusDocs(t, dir)

	want := []string{
		"bash_ls", "bash_git_go_chain", "bash_40seg", "bash_200seg", "bash_800seg", "bash_heredoc_62k",
		"bash_destructive", "powershell", "monitor", "read", "edit_2k", "write", "grep", "post_bash",
	}
	if got := slices.Sorted(maps.Keys(docs)); !slices.Equal(got, slices.Sorted(slices.Values(want))) {
		t.Errorf("corpus = %q, want %q", got, want)
	}
	// Large is set on exactly the payloads far past an ordinary call.
	wantLarge := []string{"bash_200seg", "bash_800seg", "bash_heredoc_62k"}
	for _, p := range Payloads() {
		if p.Large != slices.Contains(wantLarge, p.Name) {
			t.Errorf("%s: Large = %v", p.Name, p.Large)
		}
	}

	tools, events := map[string]bool{}, map[string]bool{}
	for _, d := range docs {
		tools[d.Tool], events[d.Event] = true, true
	}
	for _, tool := range append(slices.Clone(cmdscan.ShellTools), "Read", "Edit", "Write", "Grep") {
		if !tools[tool] {
			t.Errorf("no payload for tool %s", tool)
		}
	}
	for _, event := range []string{eventPreToolUse, eventPostToolUse} {
		if !events[event] {
			t.Errorf("no payload for event %s", event)
		}
	}
	for _, name := range []string{"read", "edit_2k", "write"} {
		if fp, _ := docs[name].Input["file_path"].(string); !strings.HasPrefix(fp, dir) {
			t.Errorf("%s: file_path %q is not inside the project", name, fp)
		}
	}
}

func TestPayloadCommands(t *testing.T) {
	t.Parallel()
	docs := corpusDocs(t, t.TempDir())
	command := func(name string) string {
		t.Helper()
		c, _ := docs[name].Input["command"].(string)
		if c == "" {
			t.Fatalf("payload %q has no command", name)
		}
		return c
	}
	tests := []struct {
		name     string
		segments int
		min, max int
	}{
		{"bash_40seg", 40, 1 << 10, 2 << 10},
		{"bash_200seg", 200, 8 << 10, 10 << 10},
		{"bash_800seg", 800, 32 << 10, hookio.MaxCommandBytes},
	}
	for _, tt := range tests {
		c := command(tt.name)
		if n := strings.Count(c, " && ") + 1; n != tt.segments {
			t.Errorf("%s: %d segments, want %d", tt.name, n, tt.segments)
		}
		if len(c) < tt.min || len(c) > tt.max {
			t.Errorf("%s: %d bytes, want %d..%d", tt.name, len(c), tt.min, tt.max)
		}
	}
	if c := command("bash_heredoc_62k"); len(c) != 62<<10 || len(c) >= hookio.MaxCommandBytes ||
		!strings.Contains(c, "<<'EOF'\n") || !strings.HasSuffix(c, "\nEOF") {
		t.Errorf("heredoc: %d bytes, want a 62 KiB heredoc under the %d-byte command cap", len(c), hookio.MaxCommandBytes)
	}
	if c := command("bash_ls"); len(c) > 64 {
		t.Errorf("bash_ls command %q is not a small call", c)
	}
	if c := command("bash_destructive"); !strings.Contains(c, "rm -rf") {
		t.Errorf("bash_destructive command %q is not a destructive call", c)
	}
	edit := docs["edit_2k"].Input
	old, _ := edit["old_string"].(string)
	repl, _ := edit["new_string"].(string)
	if n := len(old) + len(repl); n < 2<<10 || n > 3<<10 {
		t.Errorf("edit_2k carries %d bytes of strings, want about 2 KiB", n)
	}
}

func TestPayloadNamed(t *testing.T) {
	t.Parallel()
	for _, want := range Payloads() {
		got, ok := PayloadNamed(want.Name)
		if !ok || got.Name != want.Name || got.Tool != want.Tool || got.Event != want.Event || got.Large != want.Large {
			t.Errorf("PayloadNamed(%q) = %+v, %v; want the corpus payload %+v", want.Name, got, ok, want)
		}
	}
	for _, name := range []string{"", "bash_9000seg", "BASH_LS"} {
		if got, ok := PayloadNamed(name); ok {
			t.Errorf("PayloadNamed(%q) = %+v, want no payload", name, got)
		}
	}
}
