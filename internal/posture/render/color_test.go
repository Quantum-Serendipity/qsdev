package render

import (
	"os"
	"testing"
)

func TestColorSupported_NOCOLORDisables(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if ColorSupported(0) {
		t.Error("ColorSupported should return false when NO_COLOR is set")
	}
}

func TestColorSupported_NOCOLOREmptyValueDisables(t *testing.T) {
	// The NO_COLOR spec says any value, including empty, disables color.
	t.Setenv("NO_COLOR", "")

	if ColorSupported(0) {
		t.Error("ColorSupported should return false when NO_COLOR is set to empty string")
	}
}

func TestColorSupported_FORCECOLOREnables(t *testing.T) {
	// Ensure NO_COLOR is not set.
	os.Unsetenv("NO_COLOR")
	t.Setenv("FORCE_COLOR", "1")

	if !ColorSupported(0) {
		t.Error("ColorSupported should return true when FORCE_COLOR is set")
	}
}

func TestColorSupported_NOCOLORTakesPrecedence(t *testing.T) {
	// When both are set, NO_COLOR wins (checked first).
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "1")

	if ColorSupported(0) {
		t.Error("NO_COLOR should take precedence over FORCE_COLOR")
	}
}

func TestColorSupported_TERMDumbDisables(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	os.Unsetenv("FORCE_COLOR")
	t.Setenv("TERM", "dumb")

	if ColorSupported(0) {
		t.Error("ColorSupported should return false when TERM=dumb")
	}
}

func TestColorSupported_NonTTYFallback(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	os.Unsetenv("FORCE_COLOR")
	t.Setenv("TERM", "xterm-256color")

	// fd 0 in a test process is not a terminal, so this should return false.
	if ColorSupported(0) {
		t.Error("ColorSupported should return false for non-TTY fd without FORCE_COLOR")
	}
}

func TestIndicators_WithColor(t *testing.T) {
	pass, partial, skip, fail := Indicators(true)
	if pass != colorPass {
		t.Errorf("pass indicator = %q, want %q", pass, colorPass)
	}
	if partial != colorPartial {
		t.Errorf("partial indicator = %q, want %q", partial, colorPartial)
	}
	if skip != colorSkip {
		t.Errorf("skip indicator = %q, want %q", skip, colorSkip)
	}
	if fail != colorFail {
		t.Errorf("fail indicator = %q, want %q", fail, colorFail)
	}
}

func TestIndicators_WithoutColor(t *testing.T) {
	pass, partial, skip, fail := Indicators(false)
	if pass != noColorPass {
		t.Errorf("pass indicator = %q, want %q", pass, noColorPass)
	}
	if partial != noColorPartial {
		t.Errorf("partial indicator = %q, want %q", partial, noColorPartial)
	}
	if skip != noColorSkip {
		t.Errorf("skip indicator = %q, want %q", skip, noColorSkip)
	}
	if fail != noColorFail {
		t.Errorf("fail indicator = %q, want %q", fail, noColorFail)
	}
}
