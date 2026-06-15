package contentsign

import (
	"context"
	"os"
	"testing"
)

func TestAttestationStoreIsAttested(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	kp := newKeyPair(t, dir, "signer", "")
	other := newKeyPair(t, dir, "other", "")

	// A fake "binary" signed by the trusted key.
	bin := writeContent(t, dir, "server", "#!/bin/sh\necho hi\n")
	if _, err := Sign(ctx, bin, SignOptions{KeyPath: kp.secPath}); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// An unsigned binary.
	unsigned := writeContent(t, dir, "unsigned-server", "#!/bin/sh\necho hi\n")

	tests := []struct {
		name   string
		path   string
		keys   []PublicKey
		mutate func(t *testing.T)
		want   bool
	}{
		{
			name: "valid signature from trusted key",
			path: bin,
			keys: []PublicKey{kp.pub},
			want: true,
		},
		{
			name: "untrusted key set",
			path: bin,
			keys: []PublicKey{other.pub},
			want: false,
		},
		{
			name: "no signature",
			path: unsigned,
			keys: []PublicKey{kp.pub},
			want: false,
		},
		{
			name: "tampered binary fails",
			path: bin,
			keys: []PublicKey{kp.pub},
			mutate: func(t *testing.T) {
				t.Helper()
				if err := os.WriteFile(bin, []byte("#!/bin/sh\necho tampered\n"), 0o644); err != nil {
					t.Fatalf("tampering: %v", err)
				}
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mutate != nil {
				tc.mutate(t)
			}
			store := AttestationStore{TrustedKeys: tc.keys}
			if got := store.IsAttested(ctx, tc.path); got != tc.want {
				t.Errorf("IsAttested(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestAttestationStoreUnresolvableCommand(t *testing.T) {
	t.Parallel()
	store := AttestationStore{}
	// A relative command that is not on PATH cannot be resolved → not attested.
	if store.IsAttested(context.Background(), "qsdev-nonexistent-command-xyzzy") {
		t.Error("IsAttested for unresolvable command = true, want false")
	}
}
