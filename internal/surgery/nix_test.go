package surgery

import (
	"strings"
	"testing"
)

func TestNixInsertSection_NewSection(t *testing.T) {
	existing := []byte(`{ pkgs, ... }:
{
  packages = [ pkgs.git ];
}
`)

	content := []byte("  languages.go.enable = true;")

	result, err := NixInsertSection(existing, "golang", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)

	// Section should appear before the closing brace.
	if !strings.Contains(got, "# --- golang ---") {
		t.Error("open marker should be present")
	}
	if !strings.Contains(got, "# --- end golang ---") {
		t.Error("close marker should be present")
	}
	if !strings.Contains(got, "languages.go.enable = true;") {
		t.Error("content should be present")
	}

	// The closing brace should still be present.
	if !strings.Contains(got, "}") {
		t.Error("closing brace should be preserved")
	}

	// The section should appear before the final closing brace.
	braceIdx := strings.LastIndex(got, "}")
	markerIdx := strings.Index(got, "# --- golang ---")
	if markerIdx > braceIdx {
		t.Error("section should be inserted before the closing brace")
	}
}

func TestNixInsertSection_ReplacesExisting(t *testing.T) {
	existing := []byte(`{ pkgs, ... }:
{
  packages = [ pkgs.git ];

  # --- golang ---
  languages.go.enable = true;
  # --- end golang ---
}
`)
	content := []byte("  languages.go.enable = false;\n  languages.go.package = pkgs.go;")

	result, err := NixInsertSection(existing, "golang", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)

	if strings.Contains(got, "languages.go.enable = true;") {
		t.Error("old content should have been replaced")
	}
	if !strings.Contains(got, "languages.go.enable = false;") {
		t.Error("new content should be present")
	}
	if !strings.Contains(got, "languages.go.package = pkgs.go;") {
		t.Error("second line of new content should be present")
	}
}

func TestNixInsertSection_NoClosingBrace(t *testing.T) {
	existing := []byte(`# This file has no closing brace
packages = [ pkgs.git ];
`)

	_, err := NixInsertSection(existing, "golang", []byte("content"))
	if err == nil {
		t.Fatal("expected error when no closing brace found")
	}

	if !strings.Contains(err.Error(), `cannot insert section "golang"`) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNixInsertSection_ContentTrailingNewlinesTrimmed(t *testing.T) {
	existing := []byte("{\n}\n")
	content := []byte("  foo = true;\n\n\n")

	result, err := NixInsertSection(existing, "test", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	// Content should have trailing newlines trimmed.
	expected := "# --- test ---\n  foo = true;\n"
	if !strings.Contains(got, expected) {
		t.Errorf("expected content with trimmed trailing newlines:\n%s\ngot:\n%s", expected, got)
	}
}

func TestNixRemoveSection_RemovesExisting(t *testing.T) {
	existing := []byte(`{ pkgs, ... }:
{
  packages = [ pkgs.git ];

  # --- golang ---
  languages.go.enable = true;
  # --- end golang ---
}
`)

	result, err := NixRemoveSection(existing, "golang")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if strings.Contains(got, "golang") {
		t.Error("section should have been removed")
	}
	if strings.Contains(got, "languages.go.enable") {
		t.Error("section content should have been removed")
	}
	if !strings.Contains(got, "packages = [ pkgs.git ];") {
		t.Error("other content should be preserved")
	}
}

func TestNixRemoveSection_MissingSectionReturnsUnchanged(t *testing.T) {
	existing := []byte(`{ pkgs, ... }:
{
  packages = [ pkgs.git ];
}
`)

	result, err := NixRemoveSection(existing, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(existing) {
		t.Errorf("expected unchanged content for missing section\ngot:\n%s", string(result))
	}
}

func TestNixRemoveSection_PreservesOtherSections(t *testing.T) {
	existing := []byte(`{
  # --- tool-a ---
  a = true;
  # --- end tool-a ---

  # --- tool-b ---
  b = true;
  # --- end tool-b ---
}
`)

	result, err := NixRemoveSection(existing, "tool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if strings.Contains(got, "tool-a") {
		t.Error("tool-a section should have been removed")
	}
	if !strings.Contains(got, "# --- tool-b ---") {
		t.Error("tool-b section should be preserved")
	}
	if !strings.Contains(got, "b = true;") {
		t.Error("tool-b content should be preserved")
	}
}

func TestNixRemoveSection_MissingCloseMarker(t *testing.T) {
	// If the close marker is missing, the function should return unchanged content.
	existing := []byte(`{
  # --- broken ---
  content without end marker
}
`)

	result, err := NixRemoveSection(existing, "broken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(existing) {
		t.Errorf("expected unchanged content when close marker is missing\ngot:\n%s", string(result))
	}
}

func TestNixHasSection_True(t *testing.T) {
	content := []byte(`{
  # --- golang ---
  languages.go.enable = true;
  # --- end golang ---
}
`)

	if !NixHasSection(content, "golang") {
		t.Error("expected HasSection to return true")
	}
}

func TestNixHasSection_False(t *testing.T) {
	content := []byte(`{
  packages = [ pkgs.git ];
}
`)

	if NixHasSection(content, "golang") {
		t.Error("expected HasSection to return false")
	}
}

func TestNixHasSection_DifferentSection(t *testing.T) {
	content := []byte(`{
  # --- python ---
  languages.python.enable = true;
  # --- end python ---
}
`)

	if NixHasSection(content, "golang") {
		t.Error("expected HasSection to return false for different section name")
	}
}

func TestNixInsertSection_MultipleSections(t *testing.T) {
	existing := []byte("{\n}\n")

	// Insert first section.
	result, err := NixInsertSection(existing, "tool-a", []byte("  a = true;"))
	if err != nil {
		t.Fatalf("insert tool-a: %v", err)
	}

	// Insert second section.
	result, err = NixInsertSection(result, "tool-b", []byte("  b = true;"))
	if err != nil {
		t.Fatalf("insert tool-b: %v", err)
	}

	got := string(result)
	if !strings.Contains(got, "a = true;") {
		t.Error("tool-a content should be present")
	}
	if !strings.Contains(got, "b = true;") {
		t.Error("tool-b content should be present")
	}
	if !strings.Contains(got, "# --- tool-a ---") {
		t.Error("tool-a markers should be present")
	}
	if !strings.Contains(got, "# --- tool-b ---") {
		t.Error("tool-b markers should be present")
	}
}

func TestNixInsertSection_Idempotent(t *testing.T) {
	existing := []byte("{ pkgs, ... }:\n{\n}\n")

	result, err := NixInsertSection(existing, "test-section", []byte("# some nix code"))
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	result, err = NixInsertSection(result, "test-section", []byte("# some nix code"))
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}

	got := string(result)
	if count := strings.Count(got, "# --- test-section ---"); count != 1 {
		t.Errorf("expected 1 open marker, got %d\nfull output:\n%s", count, got)
	}
	if count := strings.Count(got, "# --- end test-section ---"); count != 1 {
		t.Errorf("expected 1 end marker, got %d\nfull output:\n%s", count, got)
	}
}
