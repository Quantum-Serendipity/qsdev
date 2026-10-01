package cmdscan

import "testing"

func TestExecutedPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command, want string
	}{
		{"cp a b", ""},             // parses: nothing to recover
		{"cp a b; fi", ""},         // error on the first line: nothing ran
		{"cp a b\nfi", "cp a b\n"}, // the complete first line ran
		{"cd x && cp a b\nrm c\ndone", "cd x && cp a b\nrm c\n"},
		{"if true; then cp a b\nfi fi", ""}, // the if spans the error line
		{"touch a\nif true; then\ntouch b\nfi fi", "touch a\n"},
		{"touch a\necho 'unclosed", "touch a\n"}, // unclosed quote
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			if got := ExecutedPrefix(tt.command); got != tt.want {
				t.Errorf("ExecutedPrefix(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
