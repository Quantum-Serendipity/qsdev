package contentsign

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aead.dev/minisign"
)

func TestDefaultBackend(t *testing.T) {
	t.Parallel()
	b := DefaultBackend()
	if b.Name() == "" {
		t.Error("Name() is empty")
	}
	if _, ok := b.Signer(); !ok {
		t.Error("default backend reports it cannot sign, want signer available")
	}
}

func TestVerifyContentMalformedSignature(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	content := filepath.Join(dir, "db.json")
	if err := os.WriteFile(content, []byte("payload"), 0o644); err != nil {
		t.Fatalf("writing content: %v", err)
	}

	_, _, err := DefaultBackend().VerifyContent(ctx, content, []byte("not a real minisig"), nil)
	if !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("VerifyContent error = %v, want wrapping ErrSignatureInvalid", err)
	}
}

func TestVerifyContentMissingFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// A structurally valid signature reaches the stream step, which then fails
	// to open a missing content file and returns an I/O error.
	dir := t.TempDir()
	kp := newKeyPair(t, dir, "signer", "")
	content := writeContent(t, dir, "db.json", "payload")
	sigPath, err := Sign(ctx, content, SignOptions{KeyPath: kp.secPath})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	sig, err := os.ReadFile(sigPath)
	if err != nil {
		t.Fatalf("reading sig: %v", err)
	}

	_, _, err = DefaultBackend().VerifyContent(ctx, filepath.Join(dir, "missing.json"), sig, []PublicKey{kp.pub})
	if err == nil {
		t.Error("VerifyContent on missing content file returned nil error, want I/O error")
	}
	if errors.Is(err, ErrSignatureInvalid) {
		t.Error("missing content file misclassified as ErrSignatureInvalid")
	}
}

func TestVerifyContentLegacyEdDSASignature(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	kp := newKeyPair(t, dir, "signer", "")
	content := writeContent(t, dir, "db.json", "payload")

	// Produce a LEGACY (non-prehashed, EdDSA) signature the way the standard
	// minisign CLI does — the package-level SignWithComments signs the raw
	// message — rather than via qsdev's streaming prehash path. VerifyContent
	// must still accept it (interoperability), not reject it as tampered.
	priv, err := loadSecretKey(kp.secPath, "")
	if err != nil {
		t.Fatalf("loadSecretKey: %v", err)
	}
	raw, err := os.ReadFile(content)
	if err != nil {
		t.Fatalf("reading content: %v", err)
	}
	sig := minisign.SignWithComments(priv, raw, "trusted comment", "untrusted comment")

	keyID, trusted, err := DefaultBackend().VerifyContent(ctx, content, sig, []PublicKey{kp.pub})
	if err != nil {
		t.Fatalf("VerifyContent: %v", err)
	}
	if !trusted {
		t.Fatal("legacy EdDSA signature from a trusted key was rejected")
	}
	if keyID != kp.pub.ID() {
		t.Errorf("keyID = %q, want %q", keyID, kp.pub.ID())
	}

	// A tampered file must still fail the legacy path.
	if err := os.WriteFile(content, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("tampering content: %v", err)
	}
	if _, trusted, _ := DefaultBackend().VerifyContent(ctx, content, sig, []PublicKey{kp.pub}); trusted {
		t.Error("tampered content verified against a legacy signature")
	}
}
