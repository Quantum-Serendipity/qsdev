package update

import (
	"strings"
	"testing"
)

func TestComputeUnifiedDiff_Identical(t *testing.T) {
	content := []byte("line1\nline2\nline3\n")
	diff, err := ComputeUnifiedDiff(content, content, "a.txt", "b.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff for identical content, got:\n%s", diff)
	}
}

func TestComputeUnifiedDiff_SingleLineChange(t *testing.T) {
	old := []byte("line1\nline2\nline3\n")
	new := []byte("line1\nchanged\nline3\n")
	diff, err := ComputeUnifiedDiff(old, new, "a.txt", "b.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "-line2") {
		t.Errorf("expected diff to contain '-line2', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+changed") {
		t.Errorf("expected diff to contain '+changed', got:\n%s", diff)
	}
	if !strings.Contains(diff, "@@") {
		t.Errorf("expected diff to contain @@ header, got:\n%s", diff)
	}
}

func TestComputeUnifiedDiff_MultiLineChanges(t *testing.T) {
	old := []byte("a\nb\nc\nd\ne\nf\ng\nh\n")
	new := []byte("a\nB\nc\nd\ne\nF\ng\nh\n")
	diff, err := ComputeUnifiedDiff(old, new, "old.nix", "new.nix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "-b") {
		t.Errorf("expected diff to contain '-b', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+B") {
		t.Errorf("expected diff to contain '+B', got:\n%s", diff)
	}
	if !strings.Contains(diff, "-f") {
		t.Errorf("expected diff to contain '-f', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+F") {
		t.Errorf("expected diff to contain '+F', got:\n%s", diff)
	}
}

func TestComputeUnifiedDiff_EmptyOld(t *testing.T) {
	old := []byte("")
	new := []byte("line1\nline2\n")
	diff, err := ComputeUnifiedDiff(old, new, "a.txt", "b.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "+line1") {
		t.Errorf("expected diff to show additions, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+line2") {
		t.Errorf("expected diff to show additions, got:\n%s", diff)
	}
}

func TestComputeUnifiedDiff_EmptyNew(t *testing.T) {
	old := []byte("line1\nline2\n")
	new := []byte("")
	diff, err := ComputeUnifiedDiff(old, new, "a.txt", "b.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "-line1") {
		t.Errorf("expected diff to show deletions, got:\n%s", diff)
	}
	if !strings.Contains(diff, "-line2") {
		t.Errorf("expected diff to show deletions, got:\n%s", diff)
	}
}

func TestComputeUnifiedDiff_FileNames(t *testing.T) {
	old := []byte("old\n")
	new := []byte("new\n")
	diff, err := ComputeUnifiedDiff(old, new, "devenv.nix", "devenv.nix.new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diff, "--- devenv.nix") {
		t.Errorf("expected diff to contain '--- devenv.nix', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ devenv.nix.new") {
		t.Errorf("expected diff to contain '+++ devenv.nix.new', got:\n%s", diff)
	}
}
