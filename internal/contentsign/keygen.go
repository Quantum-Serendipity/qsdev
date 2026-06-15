package contentsign

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"aead.dev/minisign"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// GenerateKeyPair generates a new Minisign key pair and writes it to disk.
//
// The secret key is written to secPath with 0o600 permissions; when password is
// non-empty it is encrypted (scrypt + Blake2b) before writing, otherwise it is
// stored in Minisign's unencrypted secret-key format. The public key is written
// to pubPath with 0o644 permissions. Existing files are never overwritten;
// either path already existing is an error.
//
// It returns the generated public key.
func GenerateKeyPair(pubPath, secPath, password string) (PublicKey, error) {
	if err := refuseExisting(pubPath); err != nil {
		return PublicKey{}, err
	}
	if err := refuseExisting(secPath); err != nil {
		return PublicKey{}, err
	}

	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		return PublicKey{}, fmt.Errorf("generating key pair: %w", err)
	}

	secBytes, err := marshalSecretKey(priv, password)
	if err != nil {
		return PublicKey{}, err
	}
	pubBytes, err := pub.MarshalText()
	if err != nil {
		return PublicKey{}, fmt.Errorf("marshaling public key: %w", err)
	}

	if err := os.WriteFile(secPath, secBytes, fileutil.ModePrivate); err != nil {
		return PublicKey{}, fmt.Errorf("writing secret key %q: %w", secPath, err)
	}
	if err := os.WriteFile(pubPath, pubBytes, fileutil.ModeReadWrite); err != nil {
		return PublicKey{}, fmt.Errorf("writing public key %q: %w", pubPath, err)
	}

	return PublicKey{inner: pub}, nil
}

// marshalSecretKey serializes priv, encrypting it when password is non-empty.
func marshalSecretKey(priv minisign.PrivateKey, password string) ([]byte, error) {
	if password != "" {
		b, err := minisign.EncryptKey(password, priv)
		if err != nil {
			return nil, fmt.Errorf("encrypting secret key: %w", err)
		}
		return b, nil
	}
	b, err := priv.MarshalText()
	if err != nil {
		return nil, fmt.Errorf("marshaling secret key: %w", err)
	}
	return b, nil
}

// refuseExisting returns an error if path already exists, so key generation
// never clobbers an existing key.
func refuseExisting(path string) error {
	switch _, err := os.Stat(path); {
	case err == nil:
		return fmt.Errorf("refusing to overwrite existing key file %q", path)
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("checking key file %q: %w", path, err)
	}
}
