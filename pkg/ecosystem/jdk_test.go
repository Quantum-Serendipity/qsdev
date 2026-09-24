package ecosystem

import (
	"errors"
	"testing"
)

func TestJDKPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{version: "", want: "jdk21"},
		{version: "21", want: "jdk21"},
		{version: "17", want: "jdk17"},
		{version: "11", want: "jdk11"},
		{version: "25", want: "jdk25"},
		{version: "8", want: "jdk8"},
		{version: "1.8", want: "jdk8"},
		{version: "1.8.0_292", want: "jdk8"},
		{version: "17.0.2", want: "jdk17"},
		{version: "17.0.2+8", want: "jdk17"},
		{version: " 21\n", want: "jdk21"},
		{version: "21-ea", want: "jdk21"},
		{version: "temurin-17.0.2", want: "jdk17"},
		{version: "openjdk64-11.0.2", want: "jdk11"},
		{version: "graalvm-ce-21.0.1", want: "jdk21"},
		{version: "17.0.2-tem", want: "jdk17"},
		{version: "Corretto-25", want: "jdk25"},
		// Majors nixpkgs does not package (EOL non-LTS, future, or garbage)
		// must fail instead of silently becoming jdk21.
		{version: "22", wantErr: true},
		{version: "23.0.1", wantErr: true},
		{version: "26", wantErr: true},
		{version: "7", wantErr: true},
		{version: "1.7", wantErr: true},
		{version: "system", wantErr: true},
		{version: "pkgs.jdk21", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			got, err := JDKPackage(tt.version)
			if tt.wantErr {
				if !errors.Is(err, ErrUnsupportedJDKVersion) {
					t.Fatalf("JDKPackage(%q) = %q, %v; want ErrUnsupportedJDKVersion", tt.version, got, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("JDKPackage(%q) = %q, %v; want %q", tt.version, got, err, tt.want)
			}
		})
	}
}

func TestJDKWizardOptions_CoverSupportedMajors(t *testing.T) {
	t.Parallel()

	opts := JDKWizardOptions()
	if len(opts) != len(SupportedJDKMajors) {
		t.Fatalf("got %d options, want %d", len(opts), len(SupportedJDKMajors))
	}
	for _, o := range opts {
		if _, err := JDKPackage(o.Value); err != nil {
			t.Errorf("wizard option %q is not provisionable: %v", o.Value, err)
		}
	}
}
