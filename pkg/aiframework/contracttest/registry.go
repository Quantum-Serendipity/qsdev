package contracttest

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestRegistryClient(t *testing.T, client aiframework.RegistryClient, fixtures ContractFixtures) {
	t.Helper()

	t.Run("SupportedTransportsIncludesStdio", func(t *testing.T) {
		transports := client.SupportedTransports()
		if !slices.Contains(transports, aiframework.TransportStdio) {
			t.Error("SupportedTransports() does not include Stdio")
		}
	})

	t.Run("FilterRemovesIncompatible", func(t *testing.T) {
		servers := []aiframework.MCPServerSpec{
			{Name: "stdio-server", Transport: aiframework.TransportStdio, Command: "test"},
			{Name: "sse-server", Transport: aiframework.TransportSSE, URL: "http://example.com"},
		}
		filtered := client.FilterServers(servers)
		supported := client.SupportedTransports()
		for _, s := range filtered {
			if !slices.Contains(supported, s.Transport) {
				t.Errorf("FilterServers kept server %q with unsupported transport %v", s.Name, s.Transport)
			}
		}
	})

	t.Run("ToolCeilingNonNegative", func(t *testing.T) {
		if ceiling := client.ToolCeiling(); ceiling < 0 {
			t.Errorf("ToolCeiling() = %d, want >= 0", ceiling)
		}
	})
}
