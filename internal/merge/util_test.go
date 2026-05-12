package merge

import (
	"slices"
	"testing"
)

func TestUnionStrings_BothEmpty(t *testing.T) {
	got := unionStrings(nil, nil)
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
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
