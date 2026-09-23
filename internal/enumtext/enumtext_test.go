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
