package perfbudget

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// Payload is one hook input of the measurement corpus: the JSON document
// Claude Code writes to a hook's stdin for a call of Tool at Event.
type Payload struct {
	Name  string
	Event string
	Tool  string
	// Large marks a payload far past an ordinary call (several KiB of
	// command, up to the hook command cap), which the in-binary hook's
	// large budget covers.
	Large bool
	// input and response build tool_input and tool_response; input takes
	// the project directory, since Claude Code passes absolute file paths.
	input    func(dir string) map[string]any
	response map[string]any
}

// Input returns the payload's tool_input for a session in project dir.
func (p Payload) Input(dir string) map[string]any {
	return p.input(dir)
}

// JSON returns the hook input document for a session in project dir.
func (p Payload) JSON(dir string) ([]byte, error) {
	doc := map[string]any{
		"session_id":      "perfbudget",
		"transcript_path": filepath.Join(dir, ".perfbudget-transcript.jsonl"),
		"cwd":             dir,
		"permission_mode": "default",
		"hook_event_name": p.Event,
		"tool_name":       p.Tool,
		"tool_input":      p.Input(dir),
	}
	if p.response != nil {
		doc["tool_response"] = p.response
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encoding payload %s: %w", p.Name, err)
	}
	return data, nil
}

const (
	eventPreToolUse  = "PreToolUse"
	eventPostToolUse = "PostToolUse"
	// heredocBytes is the size of the large heredoc command: near, but
	// under, the hook command cap (hookio.MaxCommandBytes).
	heredocBytes = 62 << 10
)

// PayloadNamed returns the corpus payload called name.
func PayloadNamed(name string) (Payload, bool) {
	corpus := Payloads()
	i := slices.IndexFunc(corpus, func(p Payload) bool { return p.Name == name })
	if i < 0 {
		return Payload{}, false
	}
	return corpus[i], true
}

// Payloads returns the hook payload corpus: small and large shell commands
// for every shell tool, a destructive command a hook decides offline, file
// tool calls and a PostToolUse call. Hooks only inspect a payload; nothing
// runs its command. It is generated rather than committed, so the large
// payloads stay reviewable.
func Payloads() []Payload {
	return []Payload{
		pre("bash_ls", "Bash", command("ls -la")),
		pre("bash_git_go_chain", "Bash", command("git status --short && go build ./... && go test ./... && git diff --stat")),
		pre("bash_40seg", "Bash", command(segments(40))),
		large(pre("bash_200seg", "Bash", command(segments(200)))),
		large(pre("bash_800seg", "Bash", command(segments(800)))),
		large(pre("bash_heredoc_62k", "Bash", command(heredoc(heredocBytes)))),
		pre("bash_destructive", "Bash", command("rm -rf ~/")),
		pre("powershell", "PowerShell", command("Get-ChildItem -Force | Select-Object -First 20")),
		pre("monitor", "Monitor", func(string) map[string]any {
			return map[string]any{"command": "tail -f build.log", "description": "watch the build log"}
		}),
		pre("read", "Read", func(dir string) map[string]any {
			return map[string]any{"file_path": filepath.Join(dir, "internal", "app", "app.go")}
		}),
		pre("edit_2k", "Edit", func(dir string) map[string]any {
			return map[string]any{
				"file_path":  filepath.Join(dir, "internal", "app", "app.go"),
				"old_string": goSource("Before", 1<<10),
				"new_string": goSource("After", 1<<10),
			}
		}),
		pre("write", "Write", func(dir string) map[string]any {
			return map[string]any{
				"file_path": filepath.Join(dir, "internal", "app", "notes.md"),
				"content":   "# Notes\n\nThe app package wires the commands together.\n",
			}
		}),
		pre("grep", "Grep", func(dir string) map[string]any {
			return map[string]any{"pattern": "func main", "path": dir, "output_mode": "files_with_matches"}
		}),
		{
			Name: "post_bash", Event: eventPostToolUse, Tool: "Bash",
			input:    command("go test ./..."),
			response: map[string]any{"stdout": "ok  \texample.com/app\t0.012s\n", "stderr": "", "interrupted": false},
		},
	}
}

func pre(name, tool string, input func(string) map[string]any) Payload {
	return Payload{Name: name, Event: eventPreToolUse, Tool: tool, input: input}
}

func large(p Payload) Payload {
	p.Large = true
	return p
}

func command(c string) func(string) map[string]any {
	return func(string) map[string]any {
		return map[string]any{"command": c, "description": "perfbudget payload"}
	}
}

// segments returns a command of n simple commands joined by "&&", about 44
// bytes each.
func segments(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("grep -n needle%03d ./internal/pkg%03d/file.go", i, i)
	}
	return strings.Join(parts, " && ")
}

// heredoc returns a command of exactly size bytes that writes a heredoc.
func heredoc(size int) string {
	const (
		head = "cat > notes.md <<'EOF'\n"
		tail = "\nEOF"
		line = "the quick brown fox jumps over the lazy dog 0123456789\n"
	)
	n := size - len(head) - len(tail)
	return head + strings.Repeat(line, n/len(line)+1)[:n] + tail
}

// goSource returns about size bytes of Go source declaring functions named
// after prefix.
func goSource(prefix string, size int) string {
	var b strings.Builder
	for i := 0; b.Len() < size; i++ {
		fmt.Fprintf(&b, "func %s%03d(x int) int {\n\treturn x * %d\n}\n\n", prefix, i, i)
	}
	return b.String()
}
