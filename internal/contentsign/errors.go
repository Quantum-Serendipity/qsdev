package contentsign

import "errors"

// Sentinel errors for expected conditions callers may need to distinguish.
var (
	// ErrNoSignature indicates no detached .minisig file was found for the content.
	ErrNoSignature = errors.New("contentsign: no signature file found")

	// ErrUntrustedKey indicates a signature parsed and is internally consistent
	// but was not produced by any of the configured trusted keys.
	ErrUntrustedKey = errors.New("contentsign: signature from untrusted key")

	// ErrSignatureInvalid indicates the signature failed cryptographic verification
	// against the content (tampered content or corrupt signature).
	ErrSignatureInvalid = errors.New("contentsign: signature verification failed")

	// ErrSigningUnsupported indicates the active backend cannot sign (verify-only).
	ErrSigningUnsupported = errors.New("contentsign: backend cannot sign")

	// ErrNoBackend indicates no signing/verification backend is available.
	ErrNoBackend = errors.New("contentsign: no signing backend available")

	// ErrInvalidPublicKey indicates a public key string is not a valid Minisign key.
	ErrInvalidPublicKey = errors.New("contentsign: invalid public key")
)
