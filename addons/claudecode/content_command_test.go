package claudecode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

func TestContentCmd_HasSubcommands(t *testing.T) {
	cmd := claudecode.ExportContentCmd()
	subs := cmd.Commands()

	expected := map[string]bool{
		"sign":   false,
		"verify": false,
		"keygen": false,
		"keys":   false,
	}
	for _, sub := range subs {
		name := sub.Name()
		if _, ok := expected[name]; !ok {
			t.Errorf("unexpected subcommand: %q", name)
		} else {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing expected subcommand: %q", name)
		}
	}
}

// runContent executes the content command with the given args and returns the
// combined stdout/stderr output and any error.
func runContent(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := claudecode.ExportContentCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestContentCmd_KeygenSignVerifyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	keyBase := filepath.Join(dir, "signer")
	secPath := keyBase + ".key"
	pubPath := keyBase + ".pub"

	// keygen
	out, err := runContent(t, "keygen", "--out", keyBase)
	if err != nil {
		t.Fatalf("keygen failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Public key:") {
		t.Errorf("keygen output should print the public key, got:\n%s", out)
	}
	if !strings.Contains(out, "WARNING") {
		t.Errorf("keygen output should warn about protecting the secret key, got:\n%s", out)
	}
	if _, err := os.Stat(secPath); err != nil {
		t.Fatalf("secret key not written: %v", err)
	}
	if _, err := os.Stat(pubPath); err != nil {
		t.Fatalf("public key not written: %v", err)
	}

	// A trusted-keys dir holding the generated public key.
	keysDir := filepath.Join(dir, "trusted")
	if err := os.MkdirAll(keysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "signer.pub"), pubBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// A content file to sign.
	content := filepath.Join(dir, "db.json")
	if err := os.WriteFile(content, []byte(`{"entries":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// sign
	out, err = runContent(t, "sign", content, "--key", secPath, "--comment", "qsdev test")
	if err != nil {
		t.Fatalf("sign failed: %v\n%s", err, out)
	}
	sigPath := content + ".minisig"
	if !strings.Contains(out, sigPath) {
		t.Errorf("sign output should print signature path %q, got:\n%s", sigPath, out)
	}
	if _, err := os.Stat(sigPath); err != nil {
		t.Fatalf("signature not written: %v", err)
	}

	// verify (success)
	out, err = runContent(t, "verify", content, "--keys", keysDir)
	if err != nil {
		t.Fatalf("verify of signed content failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "verified=true") {
		t.Errorf("verify output should report verified=true, got:\n%s", out)
	}

	// verify --json (success)
	out, err = runContent(t, "verify", content, "--keys", keysDir, "--json")
	if err != nil {
		t.Fatalf("verify --json failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"verified": true`) {
		t.Errorf("verify --json output should contain \"verified\": true, got:\n%s", out)
	}
}

func TestContentCmd_VerifyTamperedFails(t *testing.T) {
	dir := t.TempDir()
	keyBase := filepath.Join(dir, "signer")
	secPath := keyBase + ".key"
	pubPath := keyBase + ".pub"

	if _, err := runContent(t, "keygen", "--out", keyBase); err != nil {
		t.Fatalf("keygen failed: %v", err)
	}

	keysDir := filepath.Join(dir, "trusted")
	if err := os.MkdirAll(keysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "signer.pub"), pubBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	content := filepath.Join(dir, "db.json")
	if err := os.WriteFile(content, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runContent(t, "sign", content, "--key", secPath); err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	// Tamper after signing.
	if err := os.WriteFile(content, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runContent(t, "verify", content, "--keys", keysDir)
	if err == nil {
		t.Fatalf("verify of tampered content should return a non-nil error (non-zero exit), output:\n%s", out)
	}
	if !strings.Contains(err.Error(), "content verification failed") {
		t.Errorf("error should mention 'content verification failed', got: %v", err)
	}
}

func TestContentCmd_SignRequiresKey(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "db.json")
	if err := os.WriteFile(content, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runContent(t, "sign", content); err == nil {
		t.Fatal("sign without --key should fail")
	}
}

func TestContentCmd_KeysListsTrusted(t *testing.T) {
	dir := t.TempDir()
	keysDir := filepath.Join(dir, "trusted")
	if err := os.MkdirAll(keysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keyBase := filepath.Join(dir, "signer")
	if _, err := runContent(t, "keygen", "--out", keyBase); err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	pubBytes, err := os.ReadFile(keyBase + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "signer.pub"), pubBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runContent(t, "keys", "--keys", keysDir)
	if err != nil {
		t.Fatalf("keys failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Trusted Keys (1)") {
		t.Errorf("keys output should list one trusted key, got:\n%s", out)
	}

	out, err = runContent(t, "keys", "--keys", keysDir, "--json")
	if err != nil {
		t.Fatalf("keys --json failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"id"`) || !strings.Contains(out, `"key"`) {
		t.Errorf("keys --json output should contain id and key fields, got:\n%s", out)
	}
}
