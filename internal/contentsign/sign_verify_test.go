package contentsign

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// keyPair holds a freshly generated key pair plus the paths it was written to.
type keyPair struct {
	pub     PublicKey
	pubPath string
	secPath string
}

// newKeyPair generates a key pair under dir with the given name and password.
func newKeyPair(t *testing.T, dir, name, password string) keyPair {
	t.Helper()
	pubPath := filepath.Join(dir, name+".pub")
	secPath := filepath.Join(dir, name+".key")
	pub, err := GenerateKeyPair(pubPath, secPath, password)
	if err != nil {
		t.Fatalf("GenerateKeyPair(%q): %v", name, err)
	}
	return keyPair{pub: pub, pubPath: pubPath, secPath: secPath}
}

// writeContent writes data to a file named name under dir and returns its path.
func writeContent(t *testing.T, dir, name, data string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatalf("writing content %q: %v", name, err)
	}
	return p
}

// hashOf returns the lowercase hex SHA-256 of data.
func hashOf(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func TestSignVerifyRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
	}{
		{name: "unencrypted key", password: ""},
		{name: "encrypted key", password: "correct horse battery staple"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			dir := t.TempDir()
			kp := newKeyPair(t, dir, "signer", tc.password)
			content := writeContent(t, dir, "db.json", "hello signed world")

			sigPath, err := Sign(ctx, content, SignOptions{
				KeyPath:        kp.secPath,
				Password:       tc.password,
				TrustedComment: "qsdev test corpus",
			})
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}
			if want := content + ".minisig"; sigPath != want {
				t.Errorf("sigPath = %q, want %q", sigPath, want)
			}
			if _, err := os.Stat(sigPath); err != nil {
				t.Fatalf("signature not written: %v", err)
			}

			res, err := Verify(ctx, content, VerifyOptions{TrustedKeys: []PublicKey{kp.pub}})
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if !res.Verified || !res.Signed || !res.Trusted {
				t.Errorf("got Verified=%v Signed=%v Trusted=%v, want all true", res.Verified, res.Signed, res.Trusted)
			}
			if res.Status != StatusSignedVerified {
				t.Errorf("Status = %q, want %q", res.Status, StatusSignedVerified)
			}
			if res.KeyID != kp.pub.ID() {
				t.Errorf("KeyID = %q, want %q", res.KeyID, kp.pub.ID())
			}
			if res.CheckedAt.IsZero() {
				t.Error("CheckedAt is zero")
			}
		})
	}
}

func TestVerifyTamperedContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	kp := newKeyPair(t, dir, "signer", "")
	content := writeContent(t, dir, "db.json", "original content")

	if _, err := Sign(ctx, content, SignOptions{KeyPath: kp.secPath}); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	// Tamper after signing.
	if err := os.WriteFile(content, []byte("tampered content"), 0o644); err != nil {
		t.Fatalf("tampering: %v", err)
	}

	res, err := Verify(ctx, content, VerifyOptions{TrustedKeys: []PublicKey{kp.pub}})
	if err != nil {
		t.Fatalf("Verify returned a Go error for tampered content (should be a result): %v", err)
	}
	if res.Verified {
		t.Error("Verified = true for tampered content, want false")
	}
	if !res.Signed {
		t.Error("Signed = false, want true (signature file present)")
	}
	if res.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", res.Status, StatusFailed)
	}
}

func TestVerifyUntrustedKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	signer := newKeyPair(t, dir, "signer", "")
	other := newKeyPair(t, dir, "other", "")
	content := writeContent(t, dir, "db.json", "signed by signer, trusting other")

	if _, err := Sign(ctx, content, SignOptions{KeyPath: signer.secPath}); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	res, err := Verify(ctx, content, VerifyOptions{TrustedKeys: []PublicKey{other.pub}})
	if err != nil {
		t.Fatalf("Verify returned a Go error (should be a result): %v", err)
	}
	if res.Verified {
		t.Error("Verified = true with untrusted key set, want false")
	}
	if res.Trusted {
		t.Error("Trusted = true, want false")
	}
	if res.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", res.Status, StatusFailed)
	}
}

func TestVerifyMissingSignature(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	kp := newKeyPair(t, dir, "signer", "")
	content := writeContent(t, dir, "db.json", "unsigned content")

	res, err := Verify(ctx, content, VerifyOptions{TrustedKeys: []PublicKey{kp.pub}})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if res.Signed {
		t.Error("Signed = true, want false (no signature file)")
	}
	if res.Verified {
		t.Error("Verified = true, want false")
	}
	if res.Status != StatusUnverified {
		t.Errorf("Status = %q, want %q", res.Status, StatusUnverified)
	}
}

func TestVerifyEntryHashFallback(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		recorded     func(data string) string
		wantVerified bool
		wantStatus   string
	}{
		{
			name:         "matching hash",
			recorded:     hashOf,
			wantVerified: true,
			wantStatus:   StatusHashVerified,
		},
		{
			name:         "wrong hash",
			recorded:     func(string) string { return hashOf("something else entirely") },
			wantVerified: false,
			wantStatus:   StatusFailed,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			dir := t.TempDir()
			data := "hashed but unsigned content"
			content := writeContent(t, dir, "db.json", data)

			entry := ContentManifestEntry{Path: content, SHA256: tc.recorded(data)}
			res, err := VerifyEntry(ctx, entry, VerifyOptions{})
			if err != nil {
				t.Fatalf("VerifyEntry: %v", err)
			}
			if res.Signed {
				t.Error("Signed = true, want false (hash fallback path)")
			}
			if res.Verified != tc.wantVerified {
				t.Errorf("Verified = %v, want %v", res.Verified, tc.wantVerified)
			}
			if res.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", res.Status, tc.wantStatus)
			}
		})
	}
}

func TestVerifyCorpus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	kp := newKeyPair(t, dir, "signer", "")

	// Entry 0: signed-good.
	good := writeContent(t, dir, "good.json", "good payload")
	if _, err := Sign(ctx, good, SignOptions{KeyPath: kp.secPath}); err != nil {
		t.Fatalf("Sign good: %v", err)
	}

	// Entry 1: tampered after signing.
	tampered := writeContent(t, dir, "tampered.json", "before tamper")
	if _, err := Sign(ctx, tampered, SignOptions{KeyPath: kp.secPath}); err != nil {
		t.Fatalf("Sign tampered: %v", err)
	}
	if err := os.WriteFile(tampered, []byte("after tamper"), 0o644); err != nil {
		t.Fatalf("tampering: %v", err)
	}

	// Entry 2: unsigned, no hash.
	unsigned := writeContent(t, dir, "unsigned.json", "no signature here")

	// Entry 3: unsigned but hash-verified.
	hashed := writeContent(t, dir, "hashed.json", "verify me by hash")

	entries := []ContentManifestEntry{
		{Path: good},
		{Path: tampered},
		{Path: unsigned},
		{Path: hashed, SHA256: hashOf("verify me by hash")},
	}
	opts := VerifyOptions{TrustedKeys: []PublicKey{kp.pub}}

	results := VerifyCorpus(ctx, entries, opts)
	if len(results) != len(entries) {
		t.Fatalf("got %d results, want %d", len(results), len(entries))
	}

	wantStatus := []string{StatusSignedVerified, StatusFailed, StatusUnverified, StatusHashVerified}
	wantVerified := []bool{true, false, false, true}
	for i := range entries {
		if results[i].Path != entries[i].Path {
			t.Errorf("result[%d].Path = %q, want %q (order not stable)", i, results[i].Path, entries[i].Path)
		}
		if results[i].Status != wantStatus[i] {
			t.Errorf("result[%d].Status = %q, want %q", i, results[i].Status, wantStatus[i])
		}
		if results[i].Verified != wantVerified[i] {
			t.Errorf("result[%d].Verified = %v, want %v", i, results[i].Verified, wantVerified[i])
		}
	}
}

func TestVerifyCorpusEmpty(t *testing.T) {
	t.Parallel()
	if got := VerifyCorpus(context.Background(), nil, VerifyOptions{}); len(got) != 0 {
		t.Errorf("VerifyCorpus(nil) returned %d results, want 0", len(got))
	}
}

func TestGenerateKeyPairPermsAndRefusal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "k.pub")
	secPath := filepath.Join(dir, "k.key")

	if _, err := GenerateKeyPair(pubPath, secPath, ""); err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	if runtime.GOOS != "windows" {
		assertPerm(t, secPath, 0o600)
		assertPerm(t, pubPath, 0o644)
	}

	// Refuse overwrite when secret key already exists (public removed).
	if err := os.Remove(pubPath); err != nil {
		t.Fatalf("removing pub: %v", err)
	}
	if _, err := GenerateKeyPair(pubPath, secPath, ""); err == nil {
		t.Error("GenerateKeyPair overwrote an existing secret key, want error")
	}
}

// assertPerm checks that path has exactly the expected permission bits.
func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%q perm = %o, want %o", path, got, want)
	}
}

func TestSignRefusesExistingSignature(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	kp := newKeyPair(t, dir, "signer", "")
	content := writeContent(t, dir, "db.json", "content")

	if _, err := Sign(ctx, content, SignOptions{KeyPath: kp.secPath}); err != nil {
		t.Fatalf("first Sign: %v", err)
	}
	if _, err := Sign(ctx, content, SignOptions{KeyPath: kp.secPath}); err == nil {
		t.Error("second Sign overwrote signature without Force, want error")
	}
	if _, err := Sign(ctx, content, SignOptions{KeyPath: kp.secPath, Force: true}); err != nil {
		t.Errorf("Sign with Force: %v", err)
	}
}
