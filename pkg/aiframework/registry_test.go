package aiframework

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

var _ RegistryClient = (*mockRegistryClient)(nil)

type mockRegistryClient struct {
	transports []MCPTransport
}

func (m *mockRegistryClient) FrameworkID() FrameworkID { return ClaudeCode }

func (m *mockRegistryClient) SupportedTransports() []MCPTransport { return m.transports }

func (m *mockRegistryClient) ToolCeiling() int { return 0 }

func (m *mockRegistryClient) GenerateMCPConfig(_ context.Context, _ []MCPServerSpec, _ map[string]string) ([]types.GeneratedFile, error) {
	return nil, nil
}

func (m *mockRegistryClient) FilterServers(servers []MCPServerSpec) []MCPServerSpec {
	var result []MCPServerSpec
	supported := make(map[MCPTransport]bool, len(m.transports))
	for _, t := range m.transports {
		supported[t] = true
	}
	for _, s := range servers {
		if supported[s.Transport] {
			result = append(result, s)
		}
	}
	return result
}

func (m *mockRegistryClient) ValidateServers(_ context.Context, _ []MCPServerSpec) []ValidationIssue {
	return nil
}

func TestMCPTransportRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value MCPTransport
		str   string
	}{
		{name: "stdio", value: TransportStdio, str: "stdio"},
		{name: "streamable-http", value: TransportStreamableHTTP, str: "streamable-http"},
		{name: "sse", value: TransportSSE, str: "sse"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.value.String(); got != tc.str {
				t.Errorf("String() = %q, want %q", got, tc.str)
			}

			text, err := tc.value.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(text) != tc.str {
				t.Errorf("MarshalText() = %q, want %q", string(text), tc.str)
			}

			var got MCPTransport
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestMCPTransportUnknown(t *testing.T) {
	t.Parallel()

	unknown := MCPTransport(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestMCPTransportUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var tr MCPTransport
	if err := tr.UnmarshalText([]byte("grpc")); err == nil {
		t.Error("UnmarshalText(grpc) should return error")
	}
}

func TestFilterServersHelper(t *testing.T) {
	t.Parallel()

	servers := []MCPServerSpec{
		{Name: "local-tool", Transport: TransportStdio},
		{Name: "remote-api", Transport: TransportStreamableHTTP},
		{Name: "legacy-sse", Transport: TransportSSE},
	}

	client := &mockRegistryClient{transports: []MCPTransport{TransportStdio}}
	filtered := client.FilterServers(servers)

	if len(filtered) != 1 {
		t.Fatalf("FilterServers() returned %d servers, want 1", len(filtered))
	}
	if filtered[0].Name != "local-tool" {
		t.Errorf("FilterServers()[0].Name = %q, want %q", filtered[0].Name, "local-tool")
	}
}

func TestToolCeilingConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int
	}{
		{name: "Cursor", value: ToolCeilingCursor},
		{name: "Windsurf", value: ToolCeilingWindsurf},
		{name: "Copilot", value: ToolCeilingCopilot},
	}

	expected := map[string]int{
		"Cursor":   40,
		"Windsurf": 100,
		"Copilot":  128,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := expected[tc.name]
			if tc.value != want {
				t.Errorf("ToolCeiling%s = %d, want %d", tc.name, tc.value, want)
			}
		})
	}
}
