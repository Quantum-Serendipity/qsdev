package enumtext_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
)

type color int

const (
	colorUnknown color = iota // a legitimate value whose name equals the sentinel
	colorRed
)

var colorText = enumtext.New[color]("color", "color", "unknown", []string{"unknown", "red"})

func TestEnum_MarshalText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		v       color
		want    string
		wantErr bool
	}{
		{"value named like the sentinel is valid", colorUnknown, "unknown", false},
		{"regular value", colorRed, "red", false},
		{"out of range", color(7), "", true},
		{"negative", color(-1), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := colorText.MarshalText(tt.v)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MarshalText(%d) error = %v, wantErr %v", int(tt.v), err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalText(%d) = %q, want %q", int(tt.v), got, tt.want)
			}
		})
	}
}

func TestEnum_StringAndUnmarshal(t *testing.T) {
	t.Parallel()
	if got := colorText.String(color(9)); got != "unknown" {
		t.Errorf("String(out of range) = %q, want sentinel", got)
	}
	var c color
	if err := colorText.UnmarshalText([]byte("red"), &c); err != nil || c != colorRed {
		t.Errorf("UnmarshalText(red) = %d, %v", int(c), err)
	}
	if err := colorText.UnmarshalText([]byte("blue"), &c); err == nil {
		t.Error("UnmarshalText(blue) succeeded, want error")
	}
}

// level has an unset zero value: an empty name marks index 0 as a
// non-member, so the zero value is neither marshalled nor parsed.
type level int

const (
	levelUnset level = iota
	levelLow
	levelHigh
)

var levelText = enumtext.New[level]("level", "level", "unknown", []string{levelLow: "low", levelHigh: "high"})

func TestEnum_EmptyNameIsNotAMember(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		v         level
		wantValid bool
		wantStr   string
	}{
		{"unset zero value", levelUnset, false, "unknown"},
		{"low", levelLow, true, "low"},
		{"high", levelHigh, true, "high"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := levelText.Valid(tt.v); got != tt.wantValid {
				t.Errorf("Valid(%d) = %v, want %v", int(tt.v), got, tt.wantValid)
			}
			if got := levelText.String(tt.v); got != tt.wantStr {
				t.Errorf("String(%d) = %q, want %q", int(tt.v), got, tt.wantStr)
			}
			if _, err := levelText.MarshalText(tt.v); (err != nil) == tt.wantValid {
				t.Errorf("MarshalText(%d) error = %v, want error %v", int(tt.v), err, !tt.wantValid)
			}
		})
	}
	for _, text := range []string{"", "unknown"} {
		var l level
		if err := levelText.UnmarshalText([]byte(text), &l); err == nil {
			t.Errorf("UnmarshalText(%q) = %d, want error", text, int(l))
		}
	}
}
