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

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
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
// Minisign public keys (*.pub), ~/.<app>/keys/ (e.g. ~/.qsdev/keys/, in
// projectctx.LegacyDir). It fails when the home directory cannot be
// determined or is not absolute: a relative fallback would resolve against the
// working directory, which a cloned repository controls (its own .qsdev/keys
// would become trusted).
//
// The keys are security state and stay in the legacy directory, which the
// self-protection canon guards by its .<app>/ path element, until XA-WS8
// extends the canon to the per-user state directory.
func DefaultTrustedKeysDir() (string, error) {
	legacy, err := projectctx.LegacyDir()
	if err != nil {
		return "", fmt.Errorf("locating trusted keys dir: %w", err)
	}
	return filepath.Join(legacy, "keys"), nil
}

// codeKeysSubdir is the subdirectory of DefaultTrustedKeysDir holding the keys
// trusted to attest executable code (MCP server binaries and the files they
// run), kept apart from the keys trusted to sign docs and content.
const codeKeysSubdir = "code"

// CodeTrustedKeysDir returns the user-global directory holding the Minisign
// public keys (*.pub) trusted to attest code, ~/.<app>/keys/code/. It lies
// under DefaultTrustedKeysDir, so the self-protection canon guards it too.
func CodeTrustedKeysDir() (string, error) {
	dir, err := DefaultTrustedKeysDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, codeKeysSubdir), nil
}

// LoadTrustedKeys returns the set of trusted public keys: every *.pub file in
// dir plus the embedded QsdevPublicKey when it is set. The result is never nil
// (it may be empty), so it can be passed as VerifyOptions.TrustedKeys as an
// explicit key set. These keys are trusted for docs and content; code
// attestation uses LoadCodeTrustedKeys.
//
// When dir is empty, DefaultTrustedKeysDir is used; a missing default
// directory, or an undeterminable home directory, contributes no keys. A dir
// named explicitly must exist: a mistyped --keys path is an error rather than
// an empty (or silently substituted) trust set.
func LoadTrustedKeys(dir string) ([]PublicKey, error) {
	if dir != "" {
		return loadKeys(dir, false)
	}
	return loadDefaultKeys(DefaultTrustedKeysDir)
}

// LoadCodeTrustedKeys returns the keys trusted to attest code: the embedded
// QsdevPublicKey when it is set plus every *.pub file in CodeTrustedKeysDir.
// Keys in the top-level DefaultTrustedKeysDir are not included, so a key
// trusted to sign docs cannot vouch for code that runs. A missing code
// directory, or an undeterminable home directory, contributes no keys. The
// result is never nil.
func LoadCodeTrustedKeys() ([]PublicKey, error) {
	return loadDefaultKeys(CodeTrustedKeysDir)
}

// CodeKeyringNeedsMigration reports whether the top-level trusted-keys dir
// holds *.pub keys while CodeTrustedKeysDir holds none: the layout of a user
// whose code-signing keys predate the code keyring and so no longer attest.
func CodeKeyringNeedsMigration() bool {
	topDir, err := DefaultTrustedKeysDir()
	if err != nil {
		return false
	}
	top, err := pubKeyFiles(topDir)
	if err != nil || len(top) == 0 {
		return false
	}
	code, err := pubKeyFiles(filepath.Join(topDir, codeKeysSubdir))
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	return err == nil && len(code) == 0
}

// loadDefaultKeys loads the keys in the user directory locate returns,
// treating an unlocatable or missing directory as holding no keys.
func loadDefaultKeys(locate func() (string, error)) ([]PublicKey, error) {
	dir, err := locate()
	if err != nil {
		slog.Debug("no user trusted keys loaded", "error", err)
		return embeddedKeys()
	}
	return loadKeys(dir, true)
}

// loadKeys returns the embedded key (when set) plus every *.pub file in dir.
// A missing dir is an error unless missingOK.
func loadKeys(dir string, missingOK bool) ([]PublicKey, error) {
	keys, err := embeddedKeys()
	if err != nil {
		return nil, err
	}
	paths, err := pubKeyFiles(dir)
	if err != nil {
		if missingOK && errors.Is(err, fs.ErrNotExist) {
			return keys, nil
		}
		return nil, fmt.Errorf("reading trusted keys dir %q: %w", dir, err)
	}
	for _, path := range paths {
		pk, err := minisign.PublicKeyFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading trusted key %q: %w", path, err)
		}
		keys = append(keys, PublicKey{inner: pk})
	}
	return keys, nil
}

// embeddedKeys returns the embedded QsdevPublicKey as a never-nil key set.
func embeddedKeys() ([]PublicKey, error) {
	keys := []PublicKey{}
	if QsdevPublicKey != "" {
		pk, err := ParsePublicKey(QsdevPublicKey)
		if err != nil {
			return nil, fmt.Errorf("parsing embedded public key: %w", err)
		}
		keys = append(keys, pk)
	}
	return keys, nil
}

// pubKeyFiles returns the paths of the *.pub regular entries (not
// directories) directly in dir.
func pubKeyFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	return paths, nil
}
