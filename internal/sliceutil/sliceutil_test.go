package sliceutil

import (
	"testing"
)

func TestDedup(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{[]string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{[]string{"a"}, []string{"a"}},
		{nil, []string{}},
		{[]string{}, []string{}},
		{[]string{"x", "y", "z"}, []string{"x", "y", "z"}},
	}
	for _, tt := range tests {
		got := Dedup(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("Dedup(%v) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Dedup(%v)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		slice []string
		val   string
		want  []string
	}{
		{[]string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{[]string{"a", "b", "a"}, "a", []string{"b"}},
		{[]string{"a", "b", "c"}, "d", []string{"a", "b", "c"}},
		{nil, "a", []string{}},
		{[]string{}, "a", []string{}},
	}
	for _, tt := range tests {
		got := Remove(tt.slice, tt.val)
		if len(got) != len(tt.want) {
			t.Errorf("Remove(%v, %q) = %v, want %v", tt.slice, tt.val, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Remove(%v, %q)[%d] = %q, want %q", tt.slice, tt.val, i, got[i], tt.want[i])
			}
		}
	}
}
