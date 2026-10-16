package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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

// attestationHome points the user home at a fresh directory and resets the
// cached attestation store, so attestationChecker loads keys from that home.
// It returns the top-level trusted-keys dir (~/.<app>/keys).
func attestationHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	resetAttestationStore()
	t.Cleanup(resetAttestationStore)
	return filepath.Join(home, "."+branding.Get().AppName, "keys")
}

func resetAttestationStore() {
	attestationStoreOnce = sync.Once{}
	attestationStore = contentsign.AttestationStore{}
}

// signedBinary writes a file signed by a fresh key and returns the file and
// the key's .pub path.
func signedBinary(t *testing.T) (bin, pubPath string) {
	t.Helper()
	dir := t.TempDir()
	pubPath, keyPath := filepath.Join(dir, "k.pub"), filepath.Join(dir, "k.key")
	if _, err := contentsign.GenerateKeyPair(pubPath, keyPath, ""); err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	bin = filepath.Join(dir, "server")
	if err := os.WriteFile(bin, []byte("server code"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := contentsign.Sign(t.Context(), bin, contentsign.SignOptions{KeyPath: keyPath}); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return bin, pubPath
}

// installPub copies the public key at pubPath into dir as k.pub.
func installPub(t *testing.T, pubPath, dir string) {
	t.Helper()
	data, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "k.pub"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAttestation_DocsKeyDoesNotAttestCode pins U16-16: a key trusted for docs
// and content (~/.<app>/keys) must not attest the code an MCP server runs;
// only a key in the code keyring (~/.<app>/keys/code) does.
func TestAttestation_DocsKeyDoesNotAttestCode(t *testing.T) {
	bin, pubPath := signedBinary(t)
	def := &mcpregistry.McpServerDefinition{Command: bin}

	t.Run("docs keyring only", func(t *testing.T) {
		keysDir := attestationHome(t)
		installPub(t, pubPath, keysDir)
		if attestationChecker(t.Context(), def) {
			t.Error("a docs-keyring key attested MCP server code")
		}
	})

	t.Run("code keyring", func(t *testing.T) {
		keysDir := attestationHome(t)
		installPub(t, pubPath, filepath.Join(keysDir, "code"))
		if !attestationChecker(t.Context(), def) {
			t.Error("a code-keyring key did not attest MCP server code")
		}
	})
}

// TestMcpGrade_CodeKeyringMigrationHint pins U16-16: when the top-level keys
// dir holds keys but the code keyring is empty, `mcp grade` names the code
// keyring on stderr exactly once per run, even when no server reaches
// Verified (attestation, which loads the keys, is then never evaluated).
func TestMcpGrade_CodeKeyringMigrationHint(t *testing.T) {
	_, pubPath := signedBinary(t)
	// Neither server reaches Verified: npx auto-installs and sh is not a
	// provenance-verified command.
	const mcpJSON = `{"mcpServers": {
		"a": {"command": "npx", "args": ["-y", "pkg"]},
		"b": {"command": "sh", "args": ["-c", "true"]}
	}}`

	tests := []struct {
		name     string
		topLevel bool
		code     bool
		wantHint bool
	}{
		{name: "top-level keys, empty code keyring", topLevel: true, wantHint: true},
		{name: "code keyring populated", topLevel: true, code: true},
		{name: "no keys"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keysDir := attestationHome(t)
			if tt.topLevel {
				installPub(t, pubPath, keysDir)
			}
			codeDir := filepath.Join(keysDir, "code")
			if tt.code {
				installPub(t, pubPath, codeDir)
			}

			t.Chdir(writeMCPJSON(t, mcpJSON))
			cmd := mcpGradeCmd()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"--json"})
			cmd.SetContext(t.Context())
			if err := cmd.Execute(); err != nil {
				t.Fatalf("mcp grade: %v\n%s", err, stderr.String())
			}
			var entries []gradeJSONEntry
			if err := json.Unmarshal(stdout.Bytes(), &entries); err != nil {
				t.Fatalf("parsing output: %v\n%s", err, stdout.String())
			}
			for _, e := range entries {
				for _, c := range e.Criteria {
					if c.Name == "verified-provenance" && c.Passed {
						t.Fatalf("fixture server %s reached Verified provenance: %+v", e.Name, e)
					}
				}
			}

			lines := strings.Count(stderr.String(), codeDir)
			switch {
			case tt.wantHint && lines != 1:
				t.Errorf("stderr names the code keyring %d times, want exactly once:\n%s", lines, stderr.String())
			case tt.wantHint && strings.Count(strings.TrimSpace(stderr.String()), "\n") != 0:
				t.Errorf("stderr has more than the one hint line:\n%s", stderr.String())
			case !tt.wantHint && stderr.Len() != 0:
				t.Errorf("unexpected stderr output:\n%s", stderr.String())
			}
		})
	}
}
