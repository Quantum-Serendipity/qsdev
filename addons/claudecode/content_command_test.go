package claudecode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

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

// TestContentCmd_SignPasswordOutOfBand verifies the F-CAP-22.2-1 fix: an
// encrypted key can be generated and used to sign without ever placing the
// passphrase on argv, via --password-file (keygen) and --password-env (sign).
func TestContentCmd_SignPasswordOutOfBand(t *testing.T) {
	const passphrase = "correct-horse-battery-staple"

	dir := t.TempDir()
	keyBase := filepath.Join(dir, "signer")
	secPath := keyBase + ".key"
	pubPath := keyBase + ".pub"

	// A passphrase file (trailing newline is trimmed on read).
	passFile := filepath.Join(dir, "pass.txt")
	if err := os.WriteFile(passFile, []byte(passphrase+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// keygen with --password-file: encrypts the secret key, passphrase off argv.
	out, err := runContent(t, "keygen", "--out", keyBase, "--password-file", passFile)
	if err != nil {
		t.Fatalf("keygen --password-file failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(secPath); err != nil {
		t.Fatalf("secret key not written: %v", err)
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
	if err := os.WriteFile(content, []byte(`{"entries":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// sign with --password-env: resolves the encrypted key's passphrase from the
	// environment, so it never appears on the process command line.
	t.Setenv("QSDEV_TEST_PASSPHRASE", passphrase)
	out, err = runContent(t, "sign", content, "--key", secPath, "--password-env", "QSDEV_TEST_PASSPHRASE")
	if err != nil {
		t.Fatalf("sign --password-env failed: %v\n%s", err, out)
	}
	sigPath := content + ".minisig"
	if _, err := os.Stat(sigPath); err != nil {
		t.Fatalf("signature not written: %v", err)
	}

	// The round trip must verify against the trusted key.
	out, err = runContent(t, "verify", content, "--keys", keysDir)
	if err != nil {
		t.Fatalf("verify of out-of-band-signed content failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "verified=true") {
		t.Errorf("verify output should report verified=true, got:\n%s", out)
	}
}

// TestContentCmd_SignFlags asserts the out-of-band passphrase flags exist and
// that --password is documented as deprecated, and that the passphrase is not a
// required argument on argv (only --key is required).
func TestContentCmd_SignFlags(t *testing.T) {
	cmd := claudecode.ExportContentCmd()
	var sign *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "sign" {
			sign = sub
		}
	}
	if sign == nil {
		t.Fatal("sign subcommand not found")
	}
	if sign.Flags().Lookup("password-file") == nil {
		t.Error("sign should accept --password-file")
	}
	if sign.Flags().Lookup("password-env") == nil {
		t.Error("sign should accept --password-env")
	}
	pw := sign.Flags().Lookup("password")
	if pw == nil {
		t.Fatal("sign should still accept --password for compatibility")
	}
	if !strings.Contains(pw.Usage, "DEPRECATED") {
		t.Errorf("--password usage should be marked DEPRECATED, got: %q", pw.Usage)
	}
	// Only --key is required; no passphrase is required on argv.
	annotations := sign.Flags().Lookup("key").Annotations
	if _, required := annotations[cobra.BashCompOneRequiredFlag]; !required {
		t.Error("--key should be the required flag")
	}
	for _, name := range []string{"password", "password-file", "password-env"} {
		if _, req := sign.Flags().Lookup(name).Annotations[cobra.BashCompOneRequiredFlag]; req {
			t.Errorf("--%s must not be required", name)
		}
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
