package contentsign

import "errors"

// Sentinel errors for expected conditions callers may need to distinguish.
var (
	// ErrSignatureInvalid indicates the signature failed cryptographic verification
	// against the content (tampered content or corrupt signature).
	ErrSignatureInvalid = errors.New("contentsign: signature verification failed")

	// ErrSigningUnsupported indicates the active backend cannot sign (verify-only).
	ErrSigningUnsupported = errors.New("contentsign: backend cannot sign")

	// ErrInvalidPublicKey indicates a public key string is not a valid Minisign key.
	ErrInvalidPublicKey = errors.New("contentsign: invalid public key")
)
