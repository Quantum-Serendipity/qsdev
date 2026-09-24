package contentsign

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"aead.dev/minisign"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
// Minisign public keys (*.pub), ~/.<app>/keys/ (e.g. ~/.qsdev/keys/). It fails
// when the home directory cannot be determined or is not absolute: a relative
// fallback would resolve against the working directory, which a cloned
// repository controls (its own .qsdev/keys would become trusted).
func DefaultTrustedKeysDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating trusted keys dir: %w", err)
	}
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("locating trusted keys dir: home directory %q is not absolute", home)
	}
	return filepath.Join(home, "."+branding.Get().AppName, "keys"), nil
}

// LoadTrustedKeys returns the set of trusted public keys: every *.pub file in
// dir plus the embedded QsdevPublicKey when it is set. The result is never nil
// (it may be empty), so it can be passed as VerifyOptions.TrustedKeys as an
// explicit key set.
//
// When dir is empty, DefaultTrustedKeysDir is used; a missing default
// directory, or an undeterminable home directory, contributes no keys. A dir
// named explicitly must exist: a mistyped --keys path is an error rather than
// an empty (or silently substituted) trust set.
func LoadTrustedKeys(dir string) ([]PublicKey, error) {
	keys := []PublicKey{}
	if QsdevPublicKey != "" {
		pk, err := ParsePublicKey(QsdevPublicKey)
		if err != nil {
			return nil, fmt.Errorf("parsing embedded public key: %w", err)
		}
		keys = append(keys, pk)
	}

	explicit := dir != ""
	if !explicit {
		defaultDir, err := DefaultTrustedKeysDir()
		if err != nil {
			slog.Debug("no user trusted keys loaded", "error", err)
			return keys, nil
		}
		dir = defaultDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
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
