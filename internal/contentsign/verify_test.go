package contentsign

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"aead.dev/minisign"
)

// TestVerifyEntryFailureBranchesNeverVerify pins VerifyEntry's result mapping
// for a structurally corrupt signature and for a hash-fallback read error: both
// must be reported as failed and NOT Verified, with no Go error.
func TestVerifyEntryFailureBranchesNeverVerify(t *testing.T) {
	t.Parallel()
	kp := newKeyPair(t, t.TempDir(), "k", "")

	tests := []struct {
		name       string
		setup      func(t *testing.T, dir string) ContentManifestEntry
		wantSigned bool
		wantReason string
	}{
		{
			name: "malformed signature",
			setup: func(t *testing.T, dir string) ContentManifestEntry {
				t.Helper()
				content := writeContent(t, dir, "db.json", "payload")
				if err := os.WriteFile(content+".minisig", []byte("not a minisign signature"), 0o644); err != nil {
					t.Fatal(err)
				}
				return ContentManifestEntry{Path: content}
			},
			wantSigned: true,
			wantReason: "signature is malformed",
		},
		{
			name: "recorded hash but content unreadable",
			setup: func(t *testing.T, dir string) ContentManifestEntry {
				t.Helper()
				return ContentManifestEntry{Path: filepath.Join(dir, "missing.json"), SHA256: hashOf("x")}
			},
			wantReason: "hashing content",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := tt.setup(t, t.TempDir())
			res, err := VerifyEntry(context.Background(), entry, VerifyOptions{TrustedKeys: []PublicKey{kp.pub}})
			if err != nil {
				t.Fatalf("VerifyEntry: %v", err)
			}
			if res.Verified || res.Trusted {
				t.Errorf("Verified/Trusted = %v/%v, want false", res.Verified, res.Trusted)
			}
			if res.Status != StatusFailed {
				t.Errorf("Status = %q, want %q", res.Status, StatusFailed)
			}
			if res.Signed != tt.wantSigned {
				t.Errorf("Signed = %v, want %v", res.Signed, tt.wantSigned)
			}
			if !strings.Contains(res.Reason, tt.wantReason) {
				t.Errorf("Reason = %q, want it to contain %q", res.Reason, tt.wantReason)
			}
		})
	}
}

// legacySignature signs content the way the standard minisign CLI does
// (non-prehashed EdDSA).
func legacySignature(t *testing.T, kp keyPair, content string) ([]byte, uint64) {
	t.Helper()
	priv, err := loadSecretKey(kp.secPath, "")
	if err != nil {
		t.Fatalf("loadSecretKey: %v", err)
	}
	raw, err := os.ReadFile(content)
	if err != nil {
		t.Fatal(err)
	}
	sig := minisign.SignWithComments(priv, raw, "trusted", "untrusted")
	var parsed minisign.Signature
	if err := parsed.UnmarshalText(sig); err != nil {
		t.Fatal(err)
	}
	return sig, parsed.KeyID
}

// TestVerifyBufferedBounds proves a legacy signature cannot force unbounded
// buffering: an unknown key never triggers a read, and oversized content is
// refused with ErrLegacyContentTooLarge.
func TestVerifyBufferedBounds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	signer := newKeyPair(t, dir, "signer", "")
	other := newKeyPair(t, dir, "other", "")
	content := writeContent(t, dir, "db.json", "legacy signed payload")
	sig, keyID := legacySignature(t, signer, content)

	t.Run("unknown key does not read content", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "never-read")
		_, trusted, err := verifyBuffered(ctx, missing, keyID, sig, []PublicKey{other.pub}, maxBufferedContentSize)
		if err != nil || trusted {
			t.Errorf("verifyBuffered = trusted %v, err %v; want untrusted without reading content", trusted, err)
		}
	})

	t.Run("oversized content is refused", func(t *testing.T) {
		t.Parallel()
		_, trusted, err := verifyBuffered(ctx, content, keyID, sig, []PublicKey{signer.pub}, 4)
		if !errors.Is(err, ErrLegacyContentTooLarge) || trusted {
			t.Errorf("verifyBuffered = trusted %v, err %v; want ErrLegacyContentTooLarge", trusted, err)
		}
	})

	t.Run("within the bound verifies", func(t *testing.T) {
		t.Parallel()
		id, trusted, err := verifyBuffered(ctx, content, keyID, sig, []PublicKey{other.pub, signer.pub}, maxBufferedContentSize)
		if err != nil || !trusted || id != signer.pub.ID() {
			t.Errorf("verifyBuffered = %q, %v, %v; want the signer key", id, trusted, err)
		}
	})
}

// TestGenerateKeyPairNoClobberNoOrphan proves key generation never replaces a
// path that appears to be free (a dangling symlink, which os.Stat reports as
// missing) and does not leave an orphaned secret key when the public key
// cannot be written.
func TestGenerateKeyPairNoClobberNoOrphan(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs extra privileges on Windows")
	}
	dir := t.TempDir()
	secPath := filepath.Join(dir, "k.key")
	pubPath := filepath.Join(dir, "k.pub")
	if err := os.Symlink(filepath.Join(dir, "elsewhere"), pubPath); err != nil {
		t.Fatal(err)
	}

	if _, err := GenerateKeyPair(pubPath, secPath, ""); err == nil {
		t.Fatal("GenerateKeyPair replaced an existing (symlink) public key path")
	}
	if target, err := os.Readlink(pubPath); err != nil || target != filepath.Join(dir, "elsewhere") {
		t.Errorf("public key path was modified: %q, %v", target, err)
	}
	if _, err := os.Lstat(secPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("secret key left orphaned after the public key write failed (err=%v)", err)
	}
}
