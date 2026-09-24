package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// TestIsServerAttested pins F104: a signed interpreter or launcher must not
// attest a server whose arguments name a local file it runs; that file must
// be attested too.
func TestIsServerAttested(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	pub, err := contentsign.GenerateKeyPair(filepath.Join(dir, "k.pub"), filepath.Join(dir, "k.key"), "")
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	write := func(name string, sign bool) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("content of "+name), 0o644); err != nil {
			t.Fatal(err)
		}
		if sign {
			if _, err := contentsign.Sign(ctx, p, contentsign.SignOptions{KeyPath: filepath.Join(dir, "k.key")}); err != nil {
				t.Fatalf("Sign %s: %v", name, err)
			}
		}
		return p
	}
	interpreter := write("node", true)
	signedScript := write("signed.js", true)
	unsignedScript := write("unsigned.js", false)
	unsignedBinary := write("server", false)

	store := contentsign.AttestationStore{TrustedKeys: []contentsign.PublicKey{pub}}
	tests := []struct {
		name string
		def  mcpregistry.McpServerDefinition
		want bool
	}{
		{name: "signed binary, subcommand args", def: mcpregistry.McpServerDefinition{Command: interpreter, Args: []string{"mcp", "serve", "--flag"}}, want: true},
		{name: "signed interpreter runs signed script", def: mcpregistry.McpServerDefinition{Command: interpreter, Args: []string{signedScript}}, want: true},
		{name: "signed interpreter runs unsigned script", def: mcpregistry.McpServerDefinition{Command: interpreter, Args: []string{unsignedScript}}},
		{name: "unsigned binary", def: mcpregistry.McpServerDefinition{Command: unsignedBinary}},
		{name: "flag without a value is not a file", def: mcpregistry.McpServerDefinition{Command: interpreter, Args: []string{"--stdio"}}, want: true},
		{name: "flag value naming an unsigned file", def: mcpregistry.McpServerDefinition{Command: interpreter, Args: []string{"--require=" + unsignedScript}}},
		{name: "flag value naming a signed file", def: mcpregistry.McpServerDefinition{Command: interpreter, Args: []string{"--require=" + signedScript}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isServerAttested(ctx, store, &tt.def); got != tt.want {
				t.Errorf("isServerAttested = %v, want %v", got, tt.want)
			}
		})
	}
}
