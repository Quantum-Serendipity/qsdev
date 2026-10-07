package java

import "testing"

// TestMirrorOfExcept pins the generator's own guard: an allowlist id that is
// not a plain token would change which repositories the mirror matches
// (',' separates mirrorOf entries, '!' and '*' are operators), so it is
// dropped whatever config or answers validation let through.
func TestMirrorOfExcept(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		allowlist []string
		want      string
	}{
		{name: "nil", want: "*"},
		{name: "valid ids kept in order", allowlist: []string{"confluent", "ok.repo_1-x"}, want: "*,!confluent,!ok.repo_1-x"},
		{name: "duplicates dropped", allowlist: []string{"a", "b", "a"}, want: "*,!a,!b"},
		{name: "comma dropped", allowlist: []string{"a,*", "keep"}, want: "*,!keep"},
		{name: "bang dropped", allowlist: []string{"!central", "keep"}, want: "*,!keep"},
		{name: "star dropped", allowlist: []string{"*", "x*", "keep"}, want: "*,!keep"},
		{name: "whitespace dropped", allowlist: []string{"has space", "tab\there", "nl\n", " keep", "keep"}, want: "*,!keep"},
		{name: "empty dropped", allowlist: []string{"", "keep"}, want: "*,!keep"},
		{name: "all invalid", allowlist: []string{",", "!", "*", " "}, want: "*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mirrorOfExcept(tt.allowlist); got != tt.want {
				t.Errorf("mirrorOfExcept(%q) = %q, want %q", tt.allowlist, got, tt.want)
			}
		})
	}
}
