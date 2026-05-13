package surgery

import (
	"testing"
)

func TestMarkdownInsertSection_NewSection(t *testing.T) {
	existing := []byte(`# My Document

Some preamble content.

<!-- END GENERATED SECTION -->

## Footer
`)
	content := []byte("This is my tool section content.")

	result, err := MarkdownInsertSection(existing, "mytool", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)

	// The new section should appear before the END marker.
	wantContains := "<!-- gdev:mytool -->\nThis is my tool section content.\n<!-- /gdev:mytool -->\n<!-- END GENERATED SECTION -->"
	if !containsStr(got, wantContains) {
		t.Errorf("expected result to contain:\n%s\n\ngot:\n%s", wantContains, got)
	}

	// Preamble should be preserved.
	if !containsStr(got, "Some preamble content.") {
		t.Error("preamble content was lost")
	}

	// Footer should be preserved.
	if !containsStr(got, "## Footer") {
		t.Error("footer content was lost")
	}
}

func TestMarkdownInsertSection_ReplacesExisting(t *testing.T) {
	existing := []byte(`# My Document

<!-- gdev:mytool -->
Old content here.
<!-- /gdev:mytool -->

<!-- END GENERATED SECTION -->
`)
	content := []byte("New replacement content.")

	result, err := MarkdownInsertSection(existing, "mytool", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)

	if containsStr(got, "Old content here.") {
		t.Error("old content should have been replaced")
	}
	if !containsStr(got, "New replacement content.") {
		t.Error("new content should be present")
	}

	wantContains := "<!-- gdev:mytool -->\nNew replacement content.\n<!-- /gdev:mytool -->"
	if !containsStr(got, wantContains) {
		t.Errorf("expected markers wrapping new content:\n%s\n\ngot:\n%s", wantContains, got)
	}
}

func TestMarkdownInsertSection_NoEndMarker(t *testing.T) {
	existing := []byte(`# My Document

Some content with no end marker.
`)

	_, err := MarkdownInsertSection(existing, "mytool", []byte("content"))
	if err == nil {
		t.Fatal("expected error when no END GENERATED SECTION marker present")
	}

	wantErr := `cannot insert section "mytool"`
	if !containsStr(err.Error(), wantErr) {
		t.Errorf("expected error containing %q, got: %v", wantErr, err)
	}
}

func TestMarkdownInsertSection_ContentTrimmed(t *testing.T) {
	existing := []byte("<!-- END GENERATED SECTION -->\n")
	content := []byte("\n  padded content  \n\n")

	result, err := MarkdownInsertSection(existing, "test", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	wantContains := "<!-- gdev:test -->\npadded content\n<!-- /gdev:test -->"
	if !containsStr(got, wantContains) {
		t.Errorf("content should be trimmed; got:\n%s", got)
	}
}

func TestMarkdownRemoveSection_RemovesExisting(t *testing.T) {
	existing := []byte(`# My Document

<!-- gdev:mytool -->
Tool content here.
<!-- /gdev:mytool -->

<!-- END GENERATED SECTION -->
`)

	result, err := MarkdownRemoveSection(existing, "mytool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if containsStr(got, "gdev:mytool") {
		t.Error("section markers should have been removed")
	}
	if containsStr(got, "Tool content here.") {
		t.Error("section content should have been removed")
	}
	if !containsStr(got, "<!-- END GENERATED SECTION -->") {
		t.Error("END marker should be preserved")
	}
}

func TestMarkdownRemoveSection_MissingSectionReturnsUnchanged(t *testing.T) {
	existing := []byte(`# My Document

Some content.

<!-- END GENERATED SECTION -->
`)

	result, err := MarkdownRemoveSection(existing, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(existing) {
		t.Errorf("expected unchanged content for missing section\ngot:\n%s", string(result))
	}
}

func TestMarkdownRemoveSection_PreservesOtherSections(t *testing.T) {
	existing := []byte(`<!-- gdev:tool-a -->
Tool A content.
<!-- /gdev:tool-a -->
<!-- gdev:tool-b -->
Tool B content.
<!-- /gdev:tool-b -->
<!-- END GENERATED SECTION -->
`)

	result, err := MarkdownRemoveSection(existing, "tool-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if containsStr(got, "tool-a") {
		t.Error("tool-a section should have been removed")
	}
	if !containsStr(got, "<!-- gdev:tool-b -->") {
		t.Error("tool-b section should be preserved")
	}
	if !containsStr(got, "Tool B content.") {
		t.Error("tool-b content should be preserved")
	}
}

func TestMarkdownRemoveSection_MissingCloseMarker(t *testing.T) {
	// If the close marker is missing, the function should return unchanged content.
	existing := []byte(`<!-- gdev:mytool -->
Content without a close marker.
<!-- END GENERATED SECTION -->
`)

	result, err := MarkdownRemoveSection(existing, "mytool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(existing) {
		t.Errorf("expected unchanged content when close marker is missing\ngot:\n%s", string(result))
	}
}

func TestMarkdownHasSection_True(t *testing.T) {
	content := []byte(`<!-- gdev:mytool -->
Some content.
<!-- /gdev:mytool -->
`)

	if !MarkdownHasSection(content, "mytool") {
		t.Error("expected HasSection to return true for existing section")
	}
}

func TestMarkdownHasSection_False(t *testing.T) {
	content := []byte(`# Document with no sections

Just some plain markdown.
`)

	if MarkdownHasSection(content, "mytool") {
		t.Error("expected HasSection to return false for missing section")
	}
}

func TestMarkdownHasSection_DifferentSection(t *testing.T) {
	content := []byte(`<!-- gdev:other-tool -->
Content.
<!-- /gdev:other-tool -->
`)

	if MarkdownHasSection(content, "mytool") {
		t.Error("expected HasSection to return false when a different section exists")
	}
}

func TestMarkdownInsertSection_MultipleSections(t *testing.T) {
	existing := []byte("<!-- END GENERATED SECTION -->\n")

	// Insert first section.
	result, err := MarkdownInsertSection(existing, "tool-a", []byte("Content A"))
	if err != nil {
		t.Fatalf("insert tool-a: %v", err)
	}

	// Insert second section.
	result, err = MarkdownInsertSection(result, "tool-b", []byte("Content B"))
	if err != nil {
		t.Fatalf("insert tool-b: %v", err)
	}

	got := string(result)
	if !containsStr(got, "Content A") {
		t.Error("Content A should be present")
	}
	if !containsStr(got, "Content B") {
		t.Error("Content B should be present")
	}
	if !containsStr(got, "<!-- gdev:tool-a -->") {
		t.Error("tool-a markers should be present")
	}
	if !containsStr(got, "<!-- gdev:tool-b -->") {
		t.Error("tool-b markers should be present")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
