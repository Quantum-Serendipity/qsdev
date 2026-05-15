package generate_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestWriteFiles_CreatesNewFiles(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "a.yaml", Content: []byte("key: value\n"), Mode: 0644},
		{Path: "b.json", Content: []byte(`{"k":"v"}`), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Created != 2 {
		t.Errorf("Created = %d, want 2", result.Created)
	}
	for _, fr := range result.Files {
		if fr.Action != generate.ActionCreated {
			t.Errorf("file %s: action = %v, want created", fr.Path, fr.Action)
		}
	}

	// Verify files exist on disk.
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(dir, file.Path))
		if err != nil {
			t.Errorf("ReadFile %s: %v", file.Path, err)
			continue
		}
		if string(data) != string(file.Content) {
			t.Errorf("%s content = %q, want %q", file.Path, data, file.Content)
		}
	}
}

func TestWriteFiles_UpdatesExistingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create existing file.
	existing := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(existing, []byte("old: content\n"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	files := []types.GeneratedFile{
		{Path: "config.yaml", Content: []byte("new: content\n"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
	if result.Files[0].Action != generate.ActionUpdated {
		t.Errorf("action = %v, want updated", result.Files[0].Action)
	}

	data, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "new: content\n" {
		t.Errorf("content = %q, want %q", data, "new: content\n")
	}
}

func TestWriteFiles_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "should-not-exist.yaml", Content: []byte("key: value\n"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		DryRun:       true,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}

	// File should NOT exist on disk.
	path := filepath.Join(dir, "should-not-exist.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to not exist in dry-run mode, but it does")
	}
}

func TestWriteFiles_RejectsAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	absPath := "/etc/passwd"
	if runtime.GOOS == "windows" {
		absPath = `C:\Windows\System32\bad.txt`
	}
	files := []types.GeneratedFile{
		{Path: absPath, Content: []byte("bad"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if result.Files[0].Action != generate.ActionFailed {
		t.Errorf("action = %v, want failed", result.Files[0].Action)
	}
	if !strings.Contains(result.Files[0].Error.Error(), "relative") {
		t.Errorf("error = %q, want mention of 'relative'", result.Files[0].Error)
	}
}

func TestWriteFiles_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "../escape/file.txt", Content: []byte("bad"), Mode: 0644},
		{Path: "sub/../../escape.txt", Content: []byte("bad"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Failed != 2 {
		t.Errorf("Failed = %d, want 2", result.Failed)
	}
	for _, fr := range result.Files {
		if !strings.Contains(fr.Error.Error(), "traversal") {
			t.Errorf("error = %q, want mention of 'traversal'", fr.Error)
		}
	}
}

func TestWriteFiles_DefaultModeIs0644(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix file permission bits")
	}
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "default-mode.txt", Content: []byte("content"), Mode: 0},
	}

	_, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "default-mode.txt"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("mode = %o, want %o", info.Mode().Perm(), 0644)
	}
}

func TestWriteFiles_ValidationFailureContinuesToNextFile(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "bad.json", Content: []byte("{invalid json}"), Mode: 0644},
		{Path: "good.yaml", Content: []byte("key: value\n"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot: dir,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}

	// Good file should exist.
	if _, err := os.Stat(filepath.Join(dir, "good.yaml")); err != nil {
		t.Errorf("good.yaml should exist: %v", err)
	}

	// Bad file should NOT exist.
	if _, err := os.Stat(filepath.Join(dir, "bad.json")); !os.IsNotExist(err) {
		t.Errorf("bad.json should not exist")
	}
}

func TestWriteFiles_SkipValidateBypassesValidation(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "bad.json", Content: []byte("{invalid json}"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0 (validation should be skipped)", result.Failed)
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}
}

func TestWriteFiles_CreatesNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "deep/nested/dir/file.yaml", Content: []byte("key: value\n"), Mode: 0644},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot:  dir,
		SkipValidate: true,
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}

	data, err := os.ReadFile(filepath.Join(dir, "deep/nested/dir/file.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "key: value\n" {
		t.Errorf("content = %q, want %q", data, "key: value\n")
	}
}

func TestWriteResult_Summary(t *testing.T) {
	r := generate.WriteResult{
		Created: 3,
		Updated: 1,
		Skipped: 0,
		Failed:  0,
	}
	expected := "Created 3, updated 1, skipped 0, failed 0"
	if got := r.Summary(); got != expected {
		t.Errorf("Summary() = %q, want %q", got, expected)
	}
}

func TestWriteResult_HasFailures(t *testing.T) {
	tests := []struct {
		name   string
		failed int
		want   bool
	}{
		{"no failures", 0, false},
		{"one failure", 1, true},
		{"many failures", 5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := generate.WriteResult{Failed: tt.failed}
			if got := r.HasFailures(); got != tt.want {
				t.Errorf("HasFailures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteFiles_RelativeProjectRootRejected(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "file.txt", Content: []byte("content"), Mode: 0644},
	}
	_, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot: "relative/path",
	})
	if err == nil {
		t.Fatal("expected error for relative project root")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("error = %q, want mention of 'absolute'", err)
	}
}

func TestWriteFiles_NonexistentProjectRootRejected(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "file.txt", Content: []byte("content"), Mode: 0644},
	}
	_, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot: "/nonexistent/path/that/does/not/exist",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent project root")
	}
}

func TestPreviewFiles_ProducesReadableOutput(t *testing.T) {
	dir := t.TempDir()

	// Create an existing file so one shows as "update".
	existing := filepath.Join(dir, "existing.yaml")
	if err := os.WriteFile(existing, []byte("old"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	files := []types.GeneratedFile{
		{Path: "new-file.yaml", Content: []byte("key: value\n")},
		{Path: "existing.yaml", Content: []byte("key: new-value\n")},
	}

	output := generate.PreviewFiles(files, nil, dir)

	if !strings.Contains(output, "new-file.yaml") {
		t.Errorf("preview should contain 'new-file.yaml', got:\n%s", output)
	}
	if !strings.Contains(output, "existing.yaml") {
		t.Errorf("preview should contain 'existing.yaml', got:\n%s", output)
	}
	if !strings.Contains(output, "create") {
		t.Errorf("preview should contain 'create', got:\n%s", output)
	}
	if !strings.Contains(output, "update") {
		t.Errorf("preview should contain 'update', got:\n%s", output)
	}
}

func TestPreviewFiles_EmptyList(t *testing.T) {
	output := generate.PreviewFiles(nil, nil, "/tmp")
	if !strings.Contains(output, "No files") {
		t.Errorf("expected 'No files' message for empty list, got: %q", output)
	}
}

func TestValidateFiles_ReturnsAllResults(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "good.yaml", Content: []byte("key: value\n")},
		{Path: "bad.json", Content: []byte("{invalid}")},
		{Path: "readme.md", Content: []byte("# Title")},
	}

	results := generate.ValidateFiles(files)
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	// YAML should be valid.
	if !results[0].Valid {
		t.Errorf("good.yaml: expected valid, got error: %v", results[0].Error)
	}

	// JSON should be invalid.
	if results[1].Valid {
		t.Error("bad.json: expected invalid")
	}

	// Markdown should be skipped.
	if !results[2].Skipped {
		t.Error("readme.md: expected skipped")
	}
}

func TestFileAction_String(t *testing.T) {
	tests := []struct {
		action   generate.FileAction
		expected string
	}{
		{generate.ActionCreated, "created"},
		{generate.ActionUpdated, "updated"},
		{generate.ActionSkipped, "skipped"},
		{generate.ActionFailed, "failed"},
		{generate.FileAction(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.action.String(); got != tt.expected {
			t.Errorf("FileAction(%d).String() = %q, want %q", int(tt.action), got, tt.expected)
		}
	}
}

func TestWriteFiles_FileSkipValidationFlag(t *testing.T) {
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: "bad.json", Content: []byte("{invalid}"), Mode: 0644, SkipValidation: true},
	}

	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot: dir,
		// SkipValidate is false, but per-file SkipValidation is true.
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0 (per-file SkipValidation should bypass)", result.Failed)
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}
}
