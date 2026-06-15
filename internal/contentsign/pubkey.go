package contentsign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aead.dev/minisign"
)

// QsdevPublicKey is the embedded organizational Minisign public key used to
// establish trust for signed content and attested binaries.
//
// It is a placeholder until the qsdev signing key pair is generated and the
// private key stored in CI/org secrets. While empty, no key is implicitly
// trusted; trust must be supplied explicitly (e.g. via --keys or
// DefaultTrustedKeysDir).
const QsdevPublicKey = ""

// PublicKey wraps a parsed Minisign public key. It keeps the concrete crypto
// type unexported so callers depend only on this package's surface.
type PublicKey struct {
	inner minisign.PublicKey
}

// ID returns the 16-hex-digit Minisign key ID (a non-secret identifier hint).
func (k PublicKey) ID() KeyID {
	return KeyID(fmt.Sprintf("%016X", k.inner.ID()))
}

// String returns the canonical "RW..." base64 representation of the key.
func (k PublicKey) String() string {
	return k.inner.String()
}

// ParsePublicKey parses a Minisign public key from its "RW..." string form.
// The string may be the bare key or include a leading "untrusted comment:" line.
func ParsePublicKey(s string) (PublicKey, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return PublicKey{}, ErrInvalidPublicKey
	}
	var pk minisign.PublicKey
	if err := pk.UnmarshalText([]byte(s)); err != nil {
		return PublicKey{}, fmt.Errorf("%w: %v", ErrInvalidPublicKey, err)
	}
	return PublicKey{inner: pk}, nil
}

// DefaultTrustedKeysDir returns the user-global directory holding trusted
// Minisign public keys (*.pub), under ~/.qsdev/keys/.
func DefaultTrustedKeysDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".qsdev", "keys")
}

// LoadTrustedKeys returns the set of trusted public keys: every *.pub file in
// dir plus the embedded QsdevPublicKey when it is set. A missing directory is
// not an error (it yields whatever the embedded key provides). When dir is
// empty, DefaultTrustedKeysDir is used.
func LoadTrustedKeys(dir string) ([]PublicKey, error) {
	if dir == "" {
		dir = DefaultTrustedKeysDir()
	}

	var keys []PublicKey
	if QsdevPublicKey != "" {
		pk, err := ParsePublicKey(QsdevPublicKey)
		if err != nil {
			return nil, fmt.Errorf("parsing embedded public key: %w", err)
		}
		keys = append(keys, pk)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return keys, nil
		}
		return nil, fmt.Errorf("reading trusted keys dir %q: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		pk, err := minisign.PublicKeyFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading trusted key %q: %w", path, err)
		}
		keys = append(keys, PublicKey{inner: pk})
	}
	return keys, nil
}
