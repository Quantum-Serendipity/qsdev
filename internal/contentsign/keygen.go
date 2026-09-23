package contentsign

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"aead.dev/minisign"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// GenerateKeyPair generates a new Minisign key pair and writes it to disk.
//
// The secret key is written to secPath with 0o600 permissions; when password is
// non-empty it is encrypted (scrypt + Blake2b) before writing, otherwise it is
// stored in Minisign's unencrypted secret-key format. The public key is written
// to pubPath with 0o644 permissions. Existing files are never overwritten;
// either path already existing is an error, and the check is atomic with file
// creation. If the public key cannot be written, the secret key is removed.
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

	if err := os.MkdirAll(filepath.Dir(secPath), secretKeyDirMode); err != nil {
		return PublicKey{}, fmt.Errorf("creating secret key directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(pubPath), fileutil.ModeDirDefault); err != nil {
		return PublicKey{}, fmt.Errorf("creating public key directory: %w", err)
	}
	if err := writeNewFile(secPath, secBytes, fileutil.ModePrivate); err != nil {
		return PublicKey{}, fmt.Errorf("writing secret key: %w", err)
	}
	if err := writeNewFile(pubPath, pubBytes, fileutil.ModeReadWrite); err != nil {
		// Do not leave a secret key behind without its public half: it would
		// be unusable yet block every retry with "refusing to overwrite".
		if rmErr := os.Remove(secPath); rmErr != nil {
			return PublicKey{}, fmt.Errorf("writing public key: %w", errors.Join(err, rmErr))
		}
		return PublicKey{}, fmt.Errorf("writing public key: %w", err)
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

// secretKeyDirMode is the mode for a secret key's parent directory when key
// generation has to create it.
const secretKeyDirMode os.FileMode = 0o700

// writeNewFile creates path with perm and writes data to it, failing if path
// already exists in any form (including a dangling symlink). O_EXCL makes the
// existence check and the creation one atomic step, so a file that appears
// after an earlier check is never clobbered. The data is synced before
// returning, and a partially written file is removed on failure.
func writeNewFile(path string, data []byte, perm os.FileMode) (retErr error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm) //nolint:gosec // key output path chosen by the operator.
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("refusing to overwrite existing key file %q", path)
		}
		return fmt.Errorf("creating %q: %w", path, err)
	}
	defer func() {
		if retErr != nil {
			_ = f.Close()
			_ = os.Remove(path)
		}
	}()

	// Apply perm exactly: OpenFile's mode is filtered by the umask.
	if err := f.Chmod(perm); err != nil {
		return fmt.Errorf("setting mode on %q: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("syncing %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", path, err)
	}
	return nil
}

// refuseExisting returns an error if path already exists, so key generation
// never clobbers an existing key.
func refuseExisting(path string) error {
	exists, err := pathExists(path)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("refusing to overwrite existing key file %q", path)
	}
	return nil
}

// pathExists reports whether path exists, distinguishing "present" (true, nil),
// "absent" (false, nil), and a stat failure (false, err). It backs the
// refuse-to-clobber guards in this package.
func pathExists(path string) (bool, error) {
	switch _, err := os.Stat(path); {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("checking %q: %w", path, err)
	}
}
