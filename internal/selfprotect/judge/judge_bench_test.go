package judge

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/perfbudget"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

// evaluateBenchPayloads are the corpus payloads BenchmarkSelfprotectEvaluate
// times: an ordinary command, a long chain and an ordinary edit.
var evaluateBenchPayloads = []string{"bash_ls", "bash_200seg", "edit_2k"}

// TestEvaluateBenchPayloads keeps BenchmarkSelfprotectEvaluate meaningful:
// its payloads are in the corpus, decode as the selfprotect hook decodes
// them and are allowed, so the benchmark times a full evaluation rather
// than an early deny.
func TestEvaluateBenchPayloads(t *testing.T) {
	t.Parallel()
	dir := benchProject(t)
	for _, name := range evaluateBenchPayloads {
		if d, denied := Evaluate(benchCall(t, dir, name).context()); denied {
			t.Errorf("%s: denied %+v, want allowed", name, d)
		}
	}
}

// BenchmarkSelfprotectEvaluate times Evaluate on a decoded corpus payload,
// as `qsdev selfprotect` judges one: a fresh context per call, so nothing
// memoized carries over. Run it with `make bench`.
func BenchmarkSelfprotectEvaluate(b *testing.B) {
	dir := benchProject(b)
	for _, name := range evaluateBenchPayloads {
		call := benchCall(b, dir, name)
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				Evaluate(call.context())
			}
		})
	}
}

// benchProject returns a project directory holding the file the corpus's
// file payloads name.
func benchProject(tb testing.TB) string {
	tb.Helper()
	dir := filepath.Join(tb.TempDir(), "project")
	app := filepath.Join(dir, "internal", "app", "app.go")
	if err := os.MkdirAll(filepath.Dir(app), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(app, []byte("package app\n"), 0o644); err != nil {
		tb.Fatal(err)
	}
	return dir
}

// decodedCall is a corpus payload decoded as the selfprotect hook decodes it.
type decodedCall struct {
	call      *hookio.ToolCall
	input     hookio.ToolInput
	canonical string
}

// benchCall decodes the named corpus payload for a session in dir.
func benchCall(tb testing.TB, dir, name string) decodedCall {
	tb.Helper()
	p, ok := perfbudget.PayloadNamed(name)
	if !ok {
		tb.Fatalf("no payload %q in the perfbudget corpus", name)
	}
	data, err := p.JSON(dir)
	if err != nil {
		tb.Fatal(err)
	}
	call, err := hookio.ParseToolCall(context.Background(), bytes.NewReader(data))
	if err != nil {
		tb.Fatalf("%s: %v", name, err)
	}
	input, err := hookio.ParseInput(call.ToolName, call.ToolInput)
	if err != nil {
		tb.Fatalf("%s: %v", name, err)
	}
	d := decodedCall{call: call, input: input}
	if input.FilePath != "" {
		if d.canonical, err = canon.Canonicalize(input.FilePath); err != nil {
			tb.Fatalf("%s: canonicalizing %s: %v", name, input.FilePath, err)
		}
	}
	return d
}

// context returns a fresh evaluation context for the call.
func (d decodedCall) context() *rules.EvalContext {
	ctx := &rules.EvalContext{
		ToolName:      d.call.ToolName,
		FilePath:      d.input.FilePath,
		CanonicalPath: d.canonical,
		Command:       d.input.Command,
		Content:       d.input.EditedContent(),
		CWD:           d.call.CWD,
		ToolInput:     d.call.ToolInput,
	}
	if d.call.ToolName == "Edit" {
		ctx.Edits = []rules.TextEdit{{OldString: d.input.OldString, NewString: d.input.NewString, ReplaceAll: d.input.ReplaceAll}}
	}
	return ctx
}
