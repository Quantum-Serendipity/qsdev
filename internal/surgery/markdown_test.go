package surgery

import (
	"strings"
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
	wantContains := "<!-- qsdev:mytool -->\nThis is my tool section content.\n<!-- /qsdev:mytool -->\n<!-- END GENERATED SECTION -->"
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

<!-- qsdev:mytool -->
Old content here.
<!-- /qsdev:mytool -->

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

	wantContains := "<!-- qsdev:mytool -->\nNew replacement content.\n<!-- /qsdev:mytool -->"
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
	wantContains := "<!-- qsdev:test -->\npadded content\n<!-- /qsdev:test -->"
	if !containsStr(got, wantContains) {
		t.Errorf("content should be trimmed; got:\n%s", got)
	}
}

func TestMarkdownRemoveSection_RemovesExisting(t *testing.T) {
	existing := []byte(`# My Document

<!-- qsdev:mytool -->
Tool content here.
<!-- /qsdev:mytool -->

<!-- END GENERATED SECTION -->
`)

	result, err := MarkdownRemoveSection(existing, "mytool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if containsStr(got, "qsdev:mytool") {
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
	existing := []byte(`<!-- qsdev:tool-a -->
Tool A content.
<!-- /qsdev:tool-a -->
<!-- qsdev:tool-b -->
Tool B content.
<!-- /qsdev:tool-b -->
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
	if !containsStr(got, "<!-- qsdev:tool-b -->") {
		t.Error("tool-b section should be preserved")
	}
	if !containsStr(got, "Tool B content.") {
		t.Error("tool-b content should be preserved")
	}
}

func TestMarkdownRemoveSection_MissingCloseMarker(t *testing.T) {
	// If the close marker is missing, the function should return unchanged content.
	existing := []byte(`<!-- qsdev:mytool -->
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
	content := []byte(`<!-- qsdev:mytool -->
Some content.
<!-- /qsdev:mytool -->
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
	content := []byte(`<!-- qsdev:other-tool -->
Content.
<!-- /qsdev:other-tool -->
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
	if !containsStr(got, "<!-- qsdev:tool-a -->") {
		t.Error("tool-a markers should be present")
	}
	if !containsStr(got, "<!-- qsdev:tool-b -->") {
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

func TestMarkdownInsertSection_Idempotent(t *testing.T) {
	existing := []byte("<!-- END GENERATED SECTION -->\n")

	result, err := MarkdownInsertSection(existing, "test-section", []byte("hello"))
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	result, err = MarkdownInsertSection(result, "test-section", []byte("hello"))
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}

	got := string(result)
	marker := "<!-- qsdev:test-section -->"
	count := strings.Count(got, marker)
	if count != 1 {
		t.Errorf("expected exactly 1 instance of section marker, got %d\nfull output:\n%s", count, got)
	}
}
