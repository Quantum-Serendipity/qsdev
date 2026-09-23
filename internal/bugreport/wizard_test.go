package bugreport

import (
	"io"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
)

type stubProvider struct {
	name     string
	detected bool
}

func (p stubProvider) Name() string            { return p.name }
func (p stubProvider) DisplayName() string     { return p.name + " logs" }
func (p stubProvider) Detect(_, _ string) bool { return p.detected }
func (p stubProvider) Discover(_, _ string, _ time.Time) ([]extlog.LogFile, error) {
	return nil, nil
}
func (p stubProvider) Parse(io.Reader, string) ([]extlog.LogEntry, error) { return nil, nil }

// TestExtLogDescription proves the external-log prompt names only the sources
// that actually have logs, instead of a fixed list that advertised providers
// with nothing to collect.
func TestExtLogDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers []stubProvider
		want      string
	}{
		{"none detected", []stubProvider{{"nix", false}}, "No external tool logs detected."},
		{
			name:      "only detected sources, sorted",
			providers: []stubProvider{{"npm", true}, {"nix", false}, {"devenv", true}},
			want:      "Auto-detected logs from devenv logs, npm logs (scrubbed for secrets).",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := extlog.NewRegistry()
			for _, p := range tt.providers {
				reg.Register(p)
			}
			if got := extLogDescription(reg, "", ""); got != tt.want {
				t.Errorf("extLogDescription = %q, want %q", got, tt.want)
			}
		})
	}
}
