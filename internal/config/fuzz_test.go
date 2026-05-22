package config

import (
	"testing"
)

func FuzzParseQsdevConfigBytes(f *testing.F) {
	f.Add([]byte(`version: 1`))
	f.Add([]byte(`version: 1
tools:
  semgrep:
    enabled: true
`))
	f.Add([]byte(`version: 999`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`not yaml at all: [`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseQsdevConfigBytes(data)
	})
}

func FuzzParseVersionConstraint(f *testing.F) {
	f.Add("^0.7.0")
	f.Add("~> 0.7")
	f.Add(">= 1.0.0, < 2.0.0")
	f.Add("0.7.2")
	f.Add("")
	f.Add("not-a-version")

	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseVersionConstraint(input)
	})
}
