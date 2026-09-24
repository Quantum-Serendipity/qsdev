package merge

import (
	"slices"
	"strings"
	"testing"
)

func TestUnionStrings_BothEmpty(t *testing.T) {
	got := unionStrings(nil, nil)
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestUnionAndDiffStrings_NeverNil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  []string
	}{
		{"union_nil_nil", unionStrings(nil, nil)},
		{"union_empty_empty", unionStrings([]string{}, []string{})},
		{"diff_nil_nil", diffStrings(nil, nil)},
		{"diff_all_removed", diffStrings([]string{"a"}, []string{"a"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == nil {
				t.Error("got nil slice; want non-nil empty slice (marshals as [] not null)")
			}
		})
	}
}

func TestMergeSettings_EmptyAllowStaysArray(t *testing.T) {
	t.Parallel()
	ours := []byte(`{"permissions":{"allow":[],"deny":["Bash(rm -rf:*)"]}}`)
	theirs := []byte(`{"permissions":{"allow":[],"deny":["Bash(rm -rf:*)"]}}`)
	got, err := MergeSettings(ours, theirs, ours)
	if err != nil {
		t.Fatalf("MergeSettings: %v", err)
	}
	if strings.Contains(string(got), "null") {
		t.Errorf("merged settings contain null:\n%s", got)
	}
	if !strings.Contains(string(got), `"allow": []`) {
		t.Errorf("merged settings missing empty allow array:\n%s", got)
	}
}

func TestUnionStrings_OneEmpty(t *testing.T) {
	t.Run("a empty", func(t *testing.T) {
		got := unionStrings(nil, []string{"x", "y"})
		want := []string{"x", "y"}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
	t.Run("b empty", func(t *testing.T) {
		got := unionStrings([]string{"x", "y"}, nil)
		want := []string{"x", "y"}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestUnionStrings_NoDuplicates(t *testing.T) {
	got := unionStrings([]string{"a", "b"}, []string{"c", "d"})
	want := []string{"a", "b", "c", "d"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUnionStrings_WithDuplicates(t *testing.T) {
	got := unionStrings([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	want := []string{"a", "b", "c", "d"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUnionStrings_AllDuplicates(t *testing.T) {
	got := unionStrings([]string{"a", "b"}, []string{"a", "b"})
	want := []string{"a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUnionStrings_DuplicatesWithinA(t *testing.T) {
	got := unionStrings([]string{"a", "a", "b"}, []string{"c"})
	want := []string{"a", "b", "c"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDiffStrings_NoOverlap(t *testing.T) {
	got := diffStrings([]string{"a", "b"}, []string{"c", "d"})
	want := []string{"a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDiffStrings_FullOverlap(t *testing.T) {
	got := diffStrings([]string{"a", "b"}, []string{"a", "b"})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestDiffStrings_PartialOverlap(t *testing.T) {
	got := diffStrings([]string{"a", "b", "c"}, []string{"b"})
	want := []string{"a", "c"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDiffStrings_EmptyInputs(t *testing.T) {
	t.Run("both empty", func(t *testing.T) {
		got := diffStrings(nil, nil)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})
	t.Run("a empty", func(t *testing.T) {
		got := diffStrings(nil, []string{"a"})
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})
	t.Run("b empty", func(t *testing.T) {
		got := diffStrings([]string{"a", "b"}, nil)
		want := []string{"a", "b"}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}
