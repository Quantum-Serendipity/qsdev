package contentsign

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Verify checks the detached signature for the file at path against the trusted
// key set in opts (or the default keys from LoadTrustedKeys("") when
// opts.TrustedKeys is nil; a non-nil empty set means "no trusted keys").
//
// A failed verification — a missing signature, a tampered file, or a signature
// from an untrusted key — is an expected outcome reported in the returned
// VerificationResult (Verified=false), not a Go error. A non-nil error is
// returned only for genuine I/O failures (the trusted key set or the signature
// file cannot be read) or a structurally corrupt signature.
func Verify(ctx context.Context, path string, opts VerifyOptions) (VerificationResult, error) {
	return VerifyEntry(ctx, ContentManifestEntry{Path: path}, opts)
}

// VerifyEntry verifies a single manifest entry. It behaves like Verify but uses
// entry.Path and entry.SigPath(). When no signature file exists and entry has a
// recorded SHA256, it falls back to hash verification: a matching hash yields
// Status=StatusHashVerified with Verified=true (but Signed=false).
func VerifyEntry(ctx context.Context, entry ContentManifestEntry, opts VerifyOptions) (VerificationResult, error) {
	res := VerificationResult{Path: entry.Path, CheckedAt: time.Now()}

	keys, err := resolveTrustedKeys(opts)
	if err != nil {
		return res, err
	}

	sig, err := readSignature(entry.SigPath())
	if err != nil {
		return res, err
	}
	if sig == nil {
		return unsignedResult(ctx, res, entry, opts), nil
	}

	res.Signed = true
	if len(keys) == 0 {
		res.Status = StatusFailed
		res.Reason = "no trusted keys configured"
		return res, nil
	}
	keyID, trusted, err := DefaultBackend().VerifyContent(ctx, entry.Path, sig, keys)
	if err != nil {
		switch {
		case errors.Is(err, ErrSignatureInvalid):
			res.Status = StatusFailed
			res.Reason = "signature is malformed"
			return res, nil
		case errors.Is(err, ErrLegacyContentTooLarge):
			res.Status = StatusFailed
			res.Reason = err.Error()
			return res, nil
		}
		return res, err
	}
	applyVerification(&res, keyID, trusted)
	return res, nil
}

// applyVerification fills the signature-verification outcome into res.
func applyVerification(res *VerificationResult, keyID KeyID, trusted bool) {
	if trusted {
		res.Verified = true
		res.Trusted = true
		res.KeyID = keyID
		res.Status = StatusSignedVerified
		return
	}
	res.Status = StatusFailed
	res.Reason = "signature did not verify against any trusted key (tampered content or untrusted key)"
}

// unsignedResult handles an entry with no signature file, applying the SHA256
// hash fallback when a recorded hash is present.
func unsignedResult(ctx context.Context, res VerificationResult, entry ContentManifestEntry, opts VerifyOptions) VerificationResult {
	// A trust gate requires a valid cryptographic signature. Content with no
	// detached signature is unsigned regardless of whether it carries a recorded
	// hash — a manifest hash is only as trustworthy as the (unsigned) manifest
	// that records it. Fail closed here, before the hash fallback, so hash-only
	// content is never reported as Verified when RequireTrusted is set.
	if opts.RequireTrusted {
		res.Status = StatusFailed
		res.Reason = "unsigned content (no valid signature) and trust is required"
		return res
	}

	if entry.SHA256 != "" {
		ok, sum, err := hashMatches(ctx, entry.Path, entry.SHA256)
		switch {
		case err != nil:
			res.Status = StatusFailed
			res.Reason = fmt.Sprintf("hashing content: %v", err)
		case ok:
			res.Verified = true
			res.Status = StatusHashVerified
		default:
			res.Status = StatusFailed
			res.Reason = fmt.Sprintf("content hash %s does not match recorded %s", sum, strings.ToLower(entry.SHA256))
		}
		return res
	}

	res.Status = StatusUnverified
	res.Reason = "no signature"
	return res
}

// resolveTrustedKeys returns opts.TrustedKeys when the caller supplied a key
// set, or loads the default keys when it is nil. A supplied but EMPTY set is
// honored as-is: falling back to the default keys would silently widen the
// trust anchor the caller asked for.
func resolveTrustedKeys(opts VerifyOptions) ([]PublicKey, error) {
	if opts.TrustedKeys != nil {
		return opts.TrustedKeys, nil
	}
	keys, err := LoadTrustedKeys("")
	if err != nil {
		return nil, fmt.Errorf("loading trusted keys: %w", err)
	}
	return keys, nil
}

// readSignature reads the detached signature file. A missing file yields
// (nil, nil) so the caller can apply the unsigned/hash-fallback path; other
// read failures are returned as errors.
func readSignature(sigPath string) ([]byte, error) {
	b, err := os.ReadFile(sigPath) //nolint:gosec // sigPath is derived from a manifest-controlled corpus path.
	switch {
	case err == nil:
		return b, nil
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	default:
		return nil, fmt.Errorf("reading signature %q: %w", sigPath, err)
	}
}

// hashMatches streams path and compares its SHA-256 against want (hex,
// case-insensitive). It returns the match result and the computed hex digest.
func hashMatches(ctx context.Context, path, want string) (bool, string, error) {
	if err := ctx.Err(); err != nil {
		return false, "", err
	}
	f, err := os.Open(path) //nolint:gosec // path is a manifest-controlled corpus path.
	if err != nil {
		return false, "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(sum, want), sum, ctx.Err()
}
