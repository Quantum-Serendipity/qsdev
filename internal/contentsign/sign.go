package contentsign

import (
	"context"
	"fmt"
	"os"

	"aead.dev/minisign"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// Sign produces a detached Minisign signature for the file at path using the
// secret key referenced by opts.KeyPath.
//
// The signature is written to path+".minisig" and that path is returned. When
// the signature file already exists, Sign refuses to overwrite it unless
// opts.Force is set. Signing streams the content (Blake2b-512 prehash), so files
// of any size are handled without buffering.
func Sign(ctx context.Context, path string, opts SignOptions) (sigPath string, err error) {
	signer, ok := DefaultBackend().Signer()
	if !ok {
		return "", ErrSigningUnsupported
	}

	priv, err := loadSecretKey(opts.KeyPath, opts.Password)
	if err != nil {
		return "", err
	}

	sigPath = path + ".minisig"
	if err := guardSigPath(sigPath, opts.Force); err != nil {
		return "", err
	}

	sig, err := signer.SignContent(ctx, path, priv, opts.TrustedComment, opts.UntrustedComment)
	if err != nil {
		return "", fmt.Errorf("signing %q: %w", path, err)
	}

	if err := fileutil.WriteFileAtomic(sigPath, sig, fileutil.ModeReadWrite); err != nil {
		return "", fmt.Errorf("writing signature %q: %w", sigPath, err)
	}
	return sigPath, nil
}

// loadSecretKey reads the Minisign secret key at keyPath, decrypting it with
// password when the file is encrypted. minisign.PrivateKeyFromFile assumes an
// encrypted key, so unencrypted keys are unmarshaled directly.
func loadSecretKey(keyPath, password string) (minisign.PrivateKey, error) {
	raw, err := os.ReadFile(keyPath) //nolint:gosec // keyPath is operator-supplied.
	if err != nil {
		return minisign.PrivateKey{}, fmt.Errorf("reading secret key %q: %w", keyPath, err)
	}

	if minisign.IsEncrypted(raw) {
		priv, err := minisign.DecryptKey(password, raw)
		if err != nil {
			return minisign.PrivateKey{}, fmt.Errorf("decrypting secret key %q: %w", keyPath, err)
		}
		return priv, nil
	}

	var priv minisign.PrivateKey
	if err := priv.UnmarshalText(raw); err != nil {
		return minisign.PrivateKey{}, fmt.Errorf("loading secret key %q: %w", keyPath, err)
	}
	return priv, nil
}

// guardSigPath ensures sigPath may be written: it must not already exist unless
// force is set.
func guardSigPath(sigPath string, force bool) error {
	if force {
		return nil
	}
	exists, err := pathExists(sigPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("signature %q already exists (use Force to overwrite)", sigPath)
	}
	return nil
}
