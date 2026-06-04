package ecosystem_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func TestNixEscapeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain URL", in: "https://proxy.corp.com/go", want: "https://proxy.corp.com/go"},
		{name: "with double quote", in: `say "hello"`, want: `say \"hello\"`},
		{name: "with antiquotation", in: `${evil}`, want: `\${evil}`},
		{name: "with backslash", in: `path\to\dir`, want: `path\\to\\dir`},
		{name: "combined", in: `"\${x}`, want: `\"\\\${x}`},
		{name: "empty string", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ecosystem.NixEscapeString(tt.in)
			if got != tt.want {
				t.Errorf("NixEscapeString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTOMLEscapeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "https://proxy.corp.com/cargo", want: "https://proxy.corp.com/cargo"},
		{name: "with double quote", in: `key = "val"`, want: `key = \"val\"`},
		{name: "with backslash", in: `path\to`, want: `path\\to`},
		{name: "empty string", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ecosystem.TOMLEscapeString(tt.in)
			if got != tt.want {
				t.Errorf("TOMLEscapeString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestINIEscapeValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "https://proxy.corp.com/pypi/simple", want: "https://proxy.corp.com/pypi/simple"},
		{name: "with newline", in: "https://proxy.corp.com\nevil=true", want: "https://proxy.corp.comevil=true"},
		{name: "with CRLF", in: "https://proxy.corp.com\r\nevil=true", want: "https://proxy.corp.comevil=true"},
		{name: "with bare CR", in: "value\rinjection", want: "valueinjection"},
		{name: "empty string", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ecosystem.INIEscapeValue(tt.in)
			if got != tt.want {
				t.Errorf("INIEscapeValue(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGradleEscapeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "https://proxy.corp.com/maven", want: "https://proxy.corp.com/maven"},
		{name: "with single quote", in: "it's a URL", want: `it\'s a URL`},
		{name: "with backslash", in: `path\to`, want: `path\\to`},
		{name: "empty string", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ecosystem.GradleEscapeString(tt.in)
			if got != tt.want {
				t.Errorf("GradleEscapeString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
