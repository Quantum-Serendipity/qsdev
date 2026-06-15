package contentsign

import (
	"context"
	"fmt"
	"io"
	"os"

	"aead.dev/minisign"
)

// Verifier verifies a detached signature over a content file against a set of
// candidate public keys.
type Verifier interface {
	// VerifyContent streams contentPath, then checks the detached signature sig
	// against each key in keys. The signature is the serialized .minisig bytes.
	//
	// Contract:
	//   - On the first key whose verification succeeds, it returns that key's ID,
	//     true, and a nil error.
	//   - If sig is structurally invalid (not a parseable Minisign signature), it
	//     returns ("", false, error wrapping ErrSignatureInvalid).
	//   - If sig parses but no candidate key verifies it (tampered content or an
	//     untrusted/unknown key), it returns ("", false, nil) so the caller can
	//     record the result as failed/untrusted rather than an I/O error.
	//   - It returns a non-nil error only for genuine I/O failures (e.g. the
	//     content file cannot be read).
	VerifyContent(ctx context.Context, contentPath string, sig []byte, keys []PublicKey) (KeyID, bool, error)
}

// Signer signs a content file with a Minisign private key, returning the
// serialized detached .minisig bytes.
type Signer interface {
	// SignContent streams contentPath and signs its Blake2b-512 prehash digest
	// with priv, embedding trusted and untrusted comments. It returns the
	// serialized .minisig file bytes.
	SignContent(ctx context.Context, contentPath string, priv minisign.PrivateKey, trusted, untrusted string) ([]byte, error)
}

// Backend is the swappable signing/verification engine. A backend always
// verifies; it may optionally sign. The seam exists so a future hardware- or
// service-backed signer can replace the bundled pure-Go implementation without
// touching the Sign/Verify/Corpus surface.
type Backend interface {
	Verifier
	// Signer returns the backend's signer and true when signing is supported,
	// or (nil, false) for verify-only backends.
	Signer() (Signer, bool)
	// Name identifies the backend for diagnostics.
	Name() string
}

// DefaultBackend returns the bundled pure-Go Minisign backend, which both signs
// and verifies using the streaming (Blake2b-512 prehash) code path.
func DefaultBackend() Backend {
	return pureGoBackend{}
}

// pureGoBackend implements Backend (and both Verifier and Signer) using the
// vendored aead.dev/minisign library; no external minisign binary is required.
type pureGoBackend struct{}

// Name implements Backend.
func (pureGoBackend) Name() string { return "pure-go-minisign" }

// Signer implements Backend; the pure-Go backend can sign.
func (b pureGoBackend) Signer() (Signer, bool) { return b, true }

// VerifyContent implements Verifier using the streaming Reader path.
func (pureGoBackend) VerifyContent(ctx context.Context, contentPath string, sig []byte, keys []PublicKey) (KeyID, bool, error) {
	// Reject structurally invalid signatures up front so callers can tell a
	// corrupt signature apart from a verification mismatch.
	var parsed minisign.Signature
	if err := parsed.UnmarshalText(sig); err != nil {
		return "", false, fmt.Errorf("parsing signature for %q: %w", contentPath, ErrSignatureInvalid)
	}

	r, closeFn, err := streamReader(ctx, contentPath)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = closeFn() }()
	if err := drain(ctx, r); err != nil {
		return "", false, fmt.Errorf("reading content %q: %w", contentPath, err)
	}

	for _, k := range keys {
		if r.Verify(k.inner, sig) {
			return k.ID(), true, nil
		}
	}
	return "", false, nil
}

// SignContent implements Signer using the streaming Reader path.
func (pureGoBackend) SignContent(ctx context.Context, contentPath string, priv minisign.PrivateKey, trusted, untrusted string) ([]byte, error) {
	r, closeFn, err := streamReader(ctx, contentPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = closeFn() }()
	if err := drain(ctx, r); err != nil {
		return nil, fmt.Errorf("reading content %q: %w", contentPath, err)
	}
	return r.SignWithComments(priv, trusted, untrusted), nil
}

// streamReader opens contentPath and wraps it in a Minisign streaming Reader.
// The returned closeFn closes the underlying file and must always be called.
func streamReader(ctx context.Context, contentPath string) (*minisign.Reader, func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("opening content %q: %w", contentPath, err)
	}
	f, err := os.Open(contentPath) //nolint:gosec // contentPath is a manifest-controlled corpus path.
	if err != nil {
		return nil, nil, fmt.Errorf("opening content %q: %w", contentPath, err)
	}
	return minisign.NewReader(f), f.Close, nil
}

// drain reads the entire stream so the Reader computes the full message digest.
// Content is never buffered in memory, which keeps multi-GB artifacts cheap.
func drain(ctx context.Context, r *minisign.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, r); err != nil {
		return err
	}
	return ctx.Err()
}
