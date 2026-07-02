package contentsign

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolvePassword covers the out-of-band passphrase sources added for
// F-CAP-22.2-1: the passphrase may be read from a file or an environment
// variable so it never needs to be placed on the process command line. It is
// not parallel because subtests mutate the environment with t.Setenv.
func TestResolvePassword(t *testing.T) {
	dir := t.TempDir()
	pwFile := filepath.Join(dir, "pass.txt")
	if err := os.WriteFile(pwFile, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatalf("writing passphrase file: %v", err)
	}

	t.Run("literal when no source configured", func(t *testing.T) {
		got, err := resolvePassword(SignOptions{Password: "literal-secret"})
		if err != nil {
			t.Fatalf("resolvePassword: %v", err)
		}
		if got != "literal-secret" {
			t.Errorf("got %q, want %q", got, "literal-secret")
		}
	})

	t.Run("env overrides literal", func(t *testing.T) {
		t.Setenv("QSDEV_TEST_PASS", "env-secret")
		got, err := resolvePassword(SignOptions{PasswordEnv: "QSDEV_TEST_PASS", Password: "literal-secret"})
		if err != nil {
			t.Fatalf("resolvePassword: %v", err)
		}
		if got != "env-secret" {
			t.Errorf("got %q, want %q", got, "env-secret")
		}
	})

	t.Run("file overrides env and literal with trailing newline trimmed", func(t *testing.T) {
		t.Setenv("QSDEV_TEST_PASS", "env-secret")
		got, err := resolvePassword(SignOptions{
			PasswordFile: pwFile,
			PasswordEnv:  "QSDEV_TEST_PASS",
			Password:     "literal-secret",
		})
		if err != nil {
			t.Fatalf("resolvePassword: %v", err)
		}
		if got != "file-secret" {
			t.Errorf("got %q, want %q (trailing newline should be trimmed)", got, "file-secret")
		}
	})

	t.Run("declared env var unset is an error", func(t *testing.T) {
		if got, err := resolvePassword(SignOptions{PasswordEnv: "QSDEV_TEST_PASS_UNSET"}); err == nil {
			t.Fatalf("resolvePassword returned %q, want an error for an unset env var", got)
		}
	})

	t.Run("missing passphrase file is an error", func(t *testing.T) {
		if _, err := resolvePassword(SignOptions{PasswordFile: filepath.Join(dir, "does-not-exist.txt")}); err == nil {
			t.Fatal("resolvePassword returned nil error for a missing passphrase file")
		}
	})
}

// TestSignPassphraseOutOfBand is the regression guard for F-CAP-22.2-1: an
// encrypted secret key can be used for signing with the passphrase supplied
// entirely out-of-band (env var or file) — the in-memory Password literal stays
// empty, so nothing forces the passphrase onto argv. Not parallel: uses
// t.Setenv.
func TestSignPassphraseOutOfBand(t *testing.T) {
	ctx := context.Background()
	const passphrase = "correct horse battery staple"

	t.Run("from environment variable", func(t *testing.T) {
		dir := t.TempDir()
		kp := newKeyPair(t, dir, "signer", passphrase) // encrypted key
		content := writeContent(t, dir, "db.json", "sign me via env passphrase")

		t.Setenv("QSDEV_SIGN_PASS", passphrase)
		sigPath, err := Sign(ctx, content, SignOptions{
			KeyPath:     kp.secPath,
			PasswordEnv: "QSDEV_SIGN_PASS",
			// Password intentionally empty: the passphrase never touches argv.
		})
		if err != nil {
			t.Fatalf("Sign with env passphrase: %v", err)
		}
		assertSignedAndTrusted(t, content, sigPath, kp.pub)
		assertNotInArgv(t, passphrase)
	})

	t.Run("from passphrase file", func(t *testing.T) {
		dir := t.TempDir()
		kp := newKeyPair(t, dir, "signer", passphrase) // encrypted key
		content := writeContent(t, dir, "db.json", "sign me via passphrase file")
		pwFile := filepath.Join(dir, "pass.txt")
		if err := os.WriteFile(pwFile, []byte(passphrase+"\n"), 0o600); err != nil {
			t.Fatalf("writing passphrase file: %v", err)
		}

		sigPath, err := Sign(ctx, content, SignOptions{
			KeyPath:      kp.secPath,
			PasswordFile: pwFile,
		})
		if err != nil {
			t.Fatalf("Sign with passphrase file: %v", err)
		}
		assertSignedAndTrusted(t, content, sigPath, kp.pub)
		assertNotInArgv(t, passphrase)
	})
}

// assertSignedAndTrusted verifies that content at sigPath's target is a
// signature-verified, trusted result under the strict trust gate.
func assertSignedAndTrusted(t *testing.T, content, sigPath string, key PublicKey) {
	t.Helper()
	if want := content + ".minisig"; sigPath != want {
		t.Errorf("sigPath = %q, want %q", sigPath, want)
	}
	res, err := Verify(context.Background(), content, VerifyOptions{
		TrustedKeys:    []PublicKey{key},
		RequireTrusted: true,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !res.Verified || res.Status != StatusSignedVerified {
		t.Errorf("out-of-band signed content: got Verified=%v Status=%q, want true/%q",
			res.Verified, res.Status, StatusSignedVerified)
	}
}

// assertNotInArgv asserts the passphrase never appears in the process argv,
// which is exactly the exposure (ps(1), /proc/<pid>/cmdline, shell history) the
// out-of-band PasswordFile/PasswordEnv sources exist to prevent.
func assertNotInArgv(t *testing.T, secret string) {
	t.Helper()
	if strings.Contains(strings.Join(os.Args, "\x00"), secret) {
		t.Errorf("passphrase leaked into process argv: %q", os.Args)
	}
}
