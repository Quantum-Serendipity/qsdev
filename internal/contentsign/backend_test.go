package contentsign

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
