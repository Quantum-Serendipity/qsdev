package check

import (
	"strings"
	"testing"
)

func TestStatusSymbol_NoColor(t *testing.T) {
	tests := []struct {
		status   CheckStatus
		expected string
	}{
		{StatusPass, "[PASS]"},
		{StatusFail, "[FAIL]"},
		{StatusWarn, "[WARN]"},
		{StatusSkip, "[SKIP]"},
	}

	for _, tt := range tests {
		got := statusSymbol(tt.status, false)
		if got != tt.expected {
			t.Errorf("statusSymbol(%s, false) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

func TestStatusSymbol_Color(t *testing.T) {
	tests := []struct {
		status   CheckStatus
		contains string
	}{
		{StatusPass, colorGreen},
		{StatusFail, colorRed},
		{StatusWarn, colorYellow},
		{StatusSkip, colorDim},
	}

	for _, tt := range tests {
		got := statusSymbol(tt.status, true)
		if !strings.Contains(got, tt.contains) {
			t.Errorf("statusSymbol(%s, true) = %q, should contain ANSI code", tt.status, got)
		}
		if !strings.Contains(got, colorReset) {
			t.Errorf("statusSymbol(%s, true) should end with reset code", tt.status)
		}
	}
}
